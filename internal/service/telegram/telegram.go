package telegram

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/proxy"
	tele "gopkg.in/telebot.v3"
)

const procsCount = 2

type Handler interface {
	TextHandler
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
	bot     *tele.Bot
	hdlr    Handler
	ctx     context.Context
	inChan  chan Message
	outChan chan Response
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
		bot:     b,
		hdlr:    hdlr,
		inChan:  make(chan Message, 100),
		outChan: make(chan Response, 100),
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
		go tb.Sender(ctx, i)
	}

	tb.bot.Start()
	<-botStopped
	defer log.Debug().Msg("клиент telegram бота остановлен")
	return nil
}

func (tb *TeleBot) Sender(ctx context.Context, senderID int) error {
	log.Debug().Int("senderID", senderID).Msg("запуск отправителя")
	for {
		select {
		case <-ctx.Done():
			log.Debug().Int("senderID", senderID).Msg("получен сигнал остановки. завершение работы отправителя")
			return ctx.Err()
		case resp := <-tb.outChan:

			log.Debug().Int("senderID", senderID).Int("MsgID", resp.MsgCtx.Message().ID).Msg("получен ответ на сообщение")
			if errSend := resp.MsgCtx.Send(resp.Data); errSend != nil {
				log.Error().Int("senderID", senderID).Int("MsgID", resp.MsgCtx.Message().ID).Err(errSend)
				return errSend
			}
			log.Debug().Int("senderID", senderID).Int("MsgID", resp.MsgCtx.Message().ID).Msg("ответ успешно отправлен")
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
			switch msg.MsgType {
			case MsgText:
				log.Debug().Int("prcoID", procID).Int("MsgID", msg.MsgCtx.Message().ID).Msg("в обработку поступило новое сообщение")
				if err := tb.ProcessText(ctx, msg.MsgCtx); err != nil {
					log.Error().Int("prcoID", procID).Err(err).Int("MsgID", msg.MsgCtx.Message().ID).Msg("ошибка при обработки сообщения")
					tb.outChan <- Response{MsgCtx: msg.MsgCtx, Data: fmt.Sprintf("ошибка: %v", err.Error())}
				}
			}
		}
	}
}
