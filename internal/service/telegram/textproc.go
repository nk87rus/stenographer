package telegram

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nk87rus/transcriptor/internal/model"
	"github.com/rs/zerolog/log"
	tele "gopkg.in/telebot.v3"
)

var reCmd = regexp.MustCompile(`^\/(start|get|list|find|chat)`)

type TextHandler interface {
	RegisterUser(ctx context.Context, userID int64, userName string) error
	GetTranscriptionsList(ctx context.Context) ([]model.TranscriptionListItem, error)
	GetTranscription(ctx context.Context, mID int64) (*model.Transcription, error)
	SearchTranscriptions(ctx context.Context, wordList []string) ([]model.TranscriptionListItem, error)
	AIChat(ctx context.Context, request string) (string, error)
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
		default:
			tb.outChan <- Response{MsgCtx: msg, Data: fmt.Sprintf("Команда %q не поддерживается", msg.Text())}
		}
	}
	return nil
}

func (tb *TeleBot) CmdStart(ctx tele.Context) error {
	if errHdlr := tb.hdlr.RegisterUser(tb.ctx, ctx.Sender().ID, ctx.Sender().Username); errHdlr != nil {
		return errHdlr
	}

	tb.outChan <- Response{MsgCtx: ctx, Data: fmt.Sprintf("Пользователь %q успешно зарегистрирован", ctx.Sender().Username)}
	return nil
}

func (tb *TeleBot) CmdList(ctx tele.Context) error {
	data, errHdlr := tb.hdlr.GetTranscriptionsList(tb.ctx)
	if errHdlr != nil {
		return errHdlr
	}

	var resp = Response{MsgCtx: ctx}
	if len(data) == 0 {
		resp.Data = "не найдено ни одной сохранённой встречи"
	} else {
		var sb strings.Builder
		for i, m := range data {
			sb.WriteString("📝 ")
			sb.WriteString(m.String())
			if i != len(data) {
				sb.WriteString("\n")
			}
		}
		resp.Data = sb.String()
	}

	tb.outChan <- resp
	return nil
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

	var sb strings.Builder
	if data != nil {
		fmt.Fprintf(&sb, "📝 Транскрипция встречи %d от %s\n", data.Id, time.Unix(data.TimeStamp, 0).String())
		fmt.Fprintf(&sb, "Автор: %s\n", data.Author)
		sb.WriteString(strings.Repeat("-", 15) + "\n")
		sb.WriteString(data.Data)
	} else {
		fmt.Fprintf(&sb, "❗️транскрипция с идентификатором %d не найдена.", tcrID)
	}
	tb.outChan <- Response{MsgCtx: ctx, Data: sb.String()}
	return nil
}

func (tb *TeleBot) CmdFind(ctx tele.Context) error {
	wordCount := strings.Count(ctx.Message().Payload, ",")
	if wordCount == 0 {
		wordCount = 1
	}

	var wordList []string
	for w := range strings.SplitSeq(ctx.Message().Payload, ",") {
		wordList = append(wordList, strings.TrimSpace(w))
	}

	data, errHdlr := tb.hdlr.SearchTranscriptions(tb.ctx, wordList)
	if errHdlr != nil {
		return errHdlr
	}

	var resp = Response{MsgCtx: ctx}
	if len(data) == 0 {
		resp.Data = fmt.Sprintf("не найдено ни одной встречи по ключевым словам: %v", wordList)
	} else {
		var sb strings.Builder
		for i, m := range data {
			sb.WriteString("📝 ")
			sb.WriteString(m.String())
			if i != len(data) {
				sb.WriteString("\n")
			}
		}
		resp.Data = sb.String()
	}

	tb.outChan <- resp
	return nil
}

func (tb *TeleBot) CmdChat(ctx tele.Context) error {
	if ctx.Message().Payload == "" {
		return fmt.Errorf("не найден текст запроса")
	}

	result, errHdlr := tb.hdlr.AIChat(tb.ctx, ctx.Message().Payload)
	if errHdlr != nil {
		return errHdlr
	}

	tb.outChan <- Response{MsgCtx: ctx, Data: result}

	return nil
}
