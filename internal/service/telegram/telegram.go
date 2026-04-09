package telegram

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/proxy"
	tele "gopkg.in/telebot.v3"
)

const (
	procsCount   = 2
	chunkTimeout = 500 * time.Millisecond
)

type Handler interface {
	TextHandler
	AudioHandler
}

type MessageType int

const (
	MsgText MessageType = iota
	MsgVoice
	MsgAudio
)

type Message struct {
	MsgType MessageType
	MsgCtx  tele.Context
}

type Response struct {
	MsgCtx tele.Context
	Data   any
}

type TeleBot struct {
	bot    *tele.Bot
	hdlr   Handler
	ctx    context.Context
	inChan chan Message
}

func InitBot(ctx context.Context, token string, hdlr Handler) (*TeleBot, error) {
	log.Debug().Msg("инициализация клиента telegram бота")
	defer log.Debug().Msg("инициализация клиента telegram бота завершена")

	dialSocksProxy, err := proxy.SOCKS5("tcp", "127.0.0.1:10808", nil, proxy.Direct)
	if err != nil {
		log.Err(err).Msg("Error connecting to proxy")
	}

	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
		Client: &http.Client{Transport: &http.Transport{
			Dial:                dialSocksProxy.Dial,
			TLSHandshakeTimeout: 30 * time.Second,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		}},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	// b.Use(middleware.Logger())

	newTB := TeleBot{
		bot:    b,
		hdlr:   hdlr,
		inChan: make(chan Message, 100),
	}

	newTB.bot.Handle(tele.OnText, newTB.OnText)
	newTB.bot.Handle(tele.OnVoice, newTB.OnVoice)
	newTB.bot.Handle(tele.OnAudio, newTB.OnAudio)

	return &newTB, nil
}

func (tb *TeleBot) Run(ctx context.Context) error {
	log.Debug().Msg("запуск клиента telegram бота")
	tb.ctx = ctx
	botStopped := make(chan struct{})
	go func(srvCtx context.Context) {
		<-srvCtx.Done()
		tb.bot.Stop()
		close(botStopped)
	}(ctx)

	for i := range procsCount {
		go tb.Processor(ctx, i)
		// go tb.Sender(ctx, i)
	}

	tb.bot.Start()
	<-botStopped
	defer log.Debug().Msg("клиент telegram бота остановлен")
	return nil
}

func (tb *TeleBot) Sender(ctx tele.Context, dataChan <-chan string) error {
	log.Debug().Int("userID", int(ctx.Sender().ID)).Msgf("начало отправки ответа пользователю %s", ctx.Sender().Username)
	defer log.Debug().Int("userID", int(ctx.Sender().ID)).Msgf("отправка ответа пользователю %s завершена", ctx.Sender().Username)

	msg, err := ctx.Bot().Send(ctx.Chat(), "⏳ Генерирую ответ...")
	if err != nil {
		return err
	}

	var fullResponse strings.Builder
	lastUpdate := time.Now()
	chunkNum := 0
	for {
		select {
		case <-tb.ctx.Done():
			log.Debug().Str("reporter", "Sender").Str("username", ctx.Chat().Username).Msg("получен сигнал остановки. завершение работы отправки")
			return tb.ctx.Err()
		case chunk, ok := <-dataChan:
			if !ok {
				return nil
			}
			chunkNum++
			log.Debug().Str("reporter", "Sender").Str("username", ctx.Chat().Username).Int("chunkNum", chunkNum).Msg("отправка очередной части сообщения")

			fullResponse.WriteString(chunk)
			if chunkNum > 1 && time.Since(lastUpdate) < chunkTimeout {
				waitingTime := time.Until(lastUpdate.Add(chunkTimeout))
				time.After(waitingTime)
				log.Debug().Str("reporter", "Sender").Str("username", ctx.Chat().Username).Int("chunkNum", chunkNum).Str("timeout", waitingTime.String()).Msg("пауза перед отправкой очередной части сообщения")
			}

			if _, err := ctx.Bot().Edit(msg, fullResponse.String()); err != nil {
				log.Err(err).Str("chunk", chunk).Str("username", ctx.Chat().Username).Int("chunkNum", chunkNum).Msg("Ошибка при дополнении сообщения")
				return err
			}
			lastUpdate = time.Now()
		}
	}
}

func (tb *TeleBot) Processor(ctx context.Context, procID int) {
	log.Debug().Int("prcoID", procID).Msg("запуск обработчика")
	defer log.Debug().Int("prcoID", procID).Msg("обработчик остановлен")

	for {
		select {
		case <-ctx.Done():
			log.Debug().Int("prcoID", procID).Msg("получен сигнал остановки. завершение работы обработчика")
			return
		case msg := <-tb.inChan:
			log.Debug().Int("prcoID", procID).Int("MsgID", msg.MsgCtx.Message().ID).Msg("в обработку поступило новое сообщение")

			switch msg.MsgType {
			case MsgText:
				if err := tb.ProcessText(ctx, msg.MsgCtx); err != nil {
					log.Error().Int("prcoID", procID).Err(err).Int("MsgID", msg.MsgCtx.Message().ID).Msg("ошибка при обработке сообщения")
					msg.MsgCtx.Send(fmt.Sprintf("ошибка: %v", err.Error()))
				}
			case MsgAudio, MsgVoice:
				if err := tb.ProcessAudio(ctx, msg); err != nil {
					log.Error().Int("prcoID", procID).Err(err).Int("MsgID", msg.MsgCtx.Message().ID).Msg("ошибка при обработке сообщения")
					msg.MsgCtx.Send(fmt.Sprintf("ошибка: %v", err.Error()))
				}
			}
		}
	}
}

func downloadFile(b *tele.Bot, fileID string, prefix string) (string, error) {
	file, err := b.FileByID(fileID)
	if err != nil {
		return "", fmt.Errorf("не удалось получить информацию о файле: %w", err)
	}

	tmpFile, err := os.CreateTemp("", prefix+"_*."+fileExtension(file.FilePath))
	if err != nil {
		return "", fmt.Errorf("не удалось создать временный файл: %w", err)
	}
	defer tmpFile.Close()

	if err := b.Download(&file, tmpFile.Name()); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("ошибка при скачивании: %w", err)
	}

	return tmpFile.Name(), nil
}

func fileExtension(filePath string) string {
	ext := filepath.Ext(filePath)
	if ext == "" {
		return "bin"
	}
	return ext[1:]
}
