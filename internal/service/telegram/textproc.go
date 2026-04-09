package telegram

import (
	"context"
	"fmt"
	"iter"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nk87rus/transcriptor/internal/model"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	tele "gopkg.in/telebot.v3"
)

var reCmd = regexp.MustCompile(`^\/(start|get|list|find|chat|help)`)

type TextHandler interface {
	RegisterUser(ctx context.Context, userID int64, userName string) *model.DBResponse
	GetTranscriptionsList(ctx context.Context) iter.Seq2[model.TranscriptionListItem, error]
	GetTranscription(ctx context.Context, mID int64) (*model.Transcription, error)
	SearchTranscriptions(ctx context.Context, wordList []string) iter.Seq2[model.TranscriptionListItem, error]
	AIChat(ctx context.Context, request string) iter.Seq2[string, error]
}

func (tb *TeleBot) OnText(ctx tele.Context) error {
	msg := Message{
		MsgType: MsgText,
		MsgCtx:  ctx,
	}
	tb.inChan <- msg

	return nil
}

func (tb *TeleBot) ProcessText(ctx context.Context, msg tele.Context) error {
	if len(msg.Entities()) > 0 && msg.Entities()[0].Type == tele.EntityCommand {
		log.Debug().Int("ID", msg.Message().ID).Str("UserName", msg.Sender().Username).Msgf("обработка команды %q", msg.Text())
		defer log.Debug().Int("ID", msg.Message().ID).Str("UserName", msg.Sender().Username).Msgf("обработка команды %q завершена", msg.Text())

		switch reCmd.FindString(msg.Text()) {
		case "/start":
			return tb.CmdStart(msg)
		case "/list":
			return tb.CmdList(msg)
		case "/get":
			return tb.CmdGet(msg)
		case "/find":
			return tb.CmdFind(msg)
		case "/chat":
			return tb.CmdChat(msg)
		case "/help":
			return tb.CmdHelp(msg)
		default:
			msg.Send(fmt.Sprintf("Команда %q не поддерживается", msg.Text()))
		}
	}
	return nil
}

func (tb *TeleBot) CmdStart(ctx tele.Context) error {
	dbResp := tb.hdlr.RegisterUser(tb.ctx, ctx.Sender().ID, ctx.Sender().Username)
	if dbResp.Err != nil {
		return dbResp.Err
	}
	return ctx.Send(dbResp.Data)
}

func (tb *TeleBot) CmdList(ctx tele.Context) error {
	var dataChan = make(chan string)

	eg := new(errgroup.Group)
	eg.Go(func() error {
		return tb.Sender(ctx, dataChan)
	})

	for data, errHdlr := range tb.hdlr.GetTranscriptionsList(tb.ctx) {
		if errHdlr != nil {
			close(dataChan)
			return errHdlr
		}
		dataChan <- fmt.Sprintf("📝 %s\n", data.String())
	}
	close(dataChan)

	return eg.Wait()
}
func (tb *TeleBot) CmdGet(ctx tele.Context) error {
	tcrID, errID := strconv.ParseInt(ctx.Message().Payload, 10, 64)
	if errID != nil {
		return fmt.Errorf("не корректный формат идентификатора транскрипции")
	}

	data, errHdlr := tb.hdlr.GetTranscription(tb.ctx, tcrID)
	if errHdlr != nil {
		return errHdlr
	}

	var dataChan = make(chan string)

	eg := new(errgroup.Group)
	eg.Go(func() error {
		return tb.Sender(ctx, dataChan)
	})

	if !data.IsEmpty() {
		dataChan <- fmt.Sprintf("📝 Транскрипция встречи %d от %s\n", data.Id, time.Unix(data.TimeStamp, 0).String())
		dataChan <- fmt.Sprintf("Автор: %s\n", data.Author)
		dataChan <- strings.Repeat("-", 15) + "\n"
		dataChan <- data.Data
	} else {
		dataChan <- fmt.Sprintf("❗️транскрипция с идентификатором %d не найдена.", tcrID)
	}

	return eg.Wait()
}

func (tb *TeleBot) CmdFind(ctx tele.Context) error {
	var wordList []string
	for w := range strings.SplitSeq(ctx.Message().Payload, ",") {
		wordList = append(wordList, strings.TrimSpace(w))
	}

	var dataChan = make(chan string)
	eg := new(errgroup.Group)
	eg.Go(func() error {
		return tb.Sender(ctx, dataChan)
	})

	resCount := 0
	for data, errHdlr := range tb.hdlr.SearchTranscriptions(tb.ctx, wordList) {
		if errHdlr != nil {
			close(dataChan)
			return errHdlr
		}
		resCount++
		dataChan <- fmt.Sprintf("📝 %s\n", data.String())
	}
	if resCount == 0 {
		dataChan <- fmt.Sprintf("не найдено ни одной встречи по ключевым словам: %v", wordList)
	}
	close(dataChan)

	return eg.Wait()
}

func (tb *TeleBot) CmdChat(ctx tele.Context) error {
	if ctx.Message().Payload == "" {
		return fmt.Errorf("не найден текст запроса")
	}

	var dataChan = make(chan string)
	eg := new(errgroup.Group)
	eg.Go(func() error {
		return tb.Sender(ctx, dataChan)
	})

	for result, errHdlr := range tb.hdlr.AIChat(tb.ctx, ctx.Message().Payload) {
		if errHdlr != nil {
			close(dataChan)
			return errHdlr
		}
		dataChan <- result
	}
	close(dataChan)

	return eg.Wait()
}

func (tb *TeleBot) CmdHelp(ctx tele.Context) error {
	var dataChan = make(chan string)

	eg := new(errgroup.Group)
	eg.Go(func() error {
		return tb.Sender(ctx, dataChan)
	})

	data := []string{
		"❓ Допустимые команды:",
		"/start – регистрация пользователя",
		"/list  – список сохраненных встреч",
		"/get   - получение текста встречи (например: /get 1234)",
		"/find  – поиск встречи по ключевым словам (например: /find ну, так, рыба)",
		"/chat  – запрос к GigaChat",
	}

	for _, line := range data {
		dataChan <- "•" + line + "\n"
	}
	close(dataChan)

	return eg.Wait()
}
