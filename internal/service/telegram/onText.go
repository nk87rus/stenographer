package telegram

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	tele "gopkg.in/telebot.v3"
)

type TextHandler interface {
	RegisterUser(ctx context.Context, userID int64, userName string) error
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
		log.Debug().Int64("ID", msg.Sender().ID).Str("UserName", msg.Sender().Username).Msgf("обработка команды %q", msg.Text())
		defer log.Debug().Int64("ID", msg.Sender().ID).Str("UserName", msg.Sender().Username).Msgf("обработка команды %q завершена", msg.Text())

		switch msg.Text() {
		case "/start":
			return tb.CmdStart(msg)
		case "/list":
		case "/get":
		case "/find":
		case "/chat":
		default:
			// tb.SendResponse(ctx)
		}
	}
	return nil
}

func (tb *TeleBot) CmdStart(ctx tele.Context) error {
	// TODO: добавить обработку ошибки при попытке повторной регистрации пользователя
	if errHdlr := tb.hdlr.RegisterUser(tb.ctx, ctx.Sender().ID, ctx.Sender().Username); errHdlr != nil {
		return errHdlr
	}

	tb.outChan <- Response{MsgCtx: ctx, Data: fmt.Sprintf("Пользователь %q успешно зарегистрирован", ctx.Sender().Username)}
	return nil
}

func (tb *TeleBot) CmdList(ctx tele.Context) error {
	log.Debug().Int64("ID", ctx.Sender().ID).Str("UserName", ctx.Sender().Username).Msg("обработка команды /start")
	return ctx.Send("список сохраненных встреч.")
}
func HdlrCmdGet(c tele.Context) error {
	return c.Send("получение текста встречи.")
}
func HdlrCmdFind(c tele.Context) error {
	return c.Send("поиск встречи по ключевым словам.")
}
func HdlrCmdChat(c tele.Context) error {
	return c.Send("запрос к GigaChat")
}
