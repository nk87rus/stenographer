package telegram

import (
	"context"
	"fmt"
	"iter"
	"os"
	"strings"

	"github.com/nk87rus/transcriptor/internal/model"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	tele "gopkg.in/telebot.v3"
)

type AudioHandler interface {
	RecognizeAudio(ctx context.Context, srcFile *model.AudioFile) iter.Seq2[string, error]
	SaveTranscription(ctx context.Context, m *model.Transcription) error
}

func (tb *TeleBot) OnAudio(ctx tele.Context) error {
	msg := Message{
		MsgType: MsgAudio,
		MsgCtx:  ctx,
	}
	tb.inChan <- msg

	return nil
}

func (tb *TeleBot) OnVoice(ctx tele.Context) error {
	msg := Message{
		MsgType: MsgVoice,
		MsgCtx:  ctx,
	}
	tb.inChan <- msg

	return nil

}

type DataSource interface {
	MediaFile() *tele.File
	MediaType() string
}

func (tb *TeleBot) ProcessAudio(ctx context.Context, rawMsg Message) error {
	var (
		msg    = rawMsg.MsgCtx.Message()
		sender = rawMsg.MsgCtx.Sender()
	)

	log.Debug().Int("ID", msg.ID).Str("UserName", sender.Username).Msg("обработка аудио")
	defer log.Debug().Int("ID", msg.ID).Str("UserName", sender.Username).Msg("обработка аудио завершена")

	var ds DataSource
	switch rawMsg.MsgType {
	case MsgAudio:
		ds = msg.Audio
	case MsgVoice:
		ds = msg.Voice
	default:
		ds = nil
	}

	if ds == nil {
		return fmt.Errorf("не удалось получить аудиофайл")
	}

	srcFile := ds.MediaFile()
	localFilePath, err := downloadFile(tb.bot, srcFile.FileID, ds.MediaType())
	if err != nil {
		return fmt.Errorf("ошибка скачивания аудио: %w", err)
	}
	log.Debug().Int("ID", msg.ID).Str("UserName", sender.Username).Str("tmpFile", localFilePath).Str("file", srcFile.FilePath).Msg("Файл успешно получен")
	defer os.Remove(localFilePath)

	af, errAF := model.MakeAudioFile(localFilePath)
	if errAF != nil {
		return errAF
	}

	var dataChan = make(chan string)

	eg := new(errgroup.Group)
	eg.Go(func() error {
		return tb.Sender(rawMsg.MsgCtx, dataChan)
	})

	var rcgResult strings.Builder
	for r, errRcg := range tb.hdlr.RecognizeAudio(ctx, af) {
		if errRcg != nil {
			close(dataChan)
			return errRcg
		}
		rcgResult.WriteString(r + " ")
		dataChan <- r

	}
	m := model.Transcription{
		Id:        int64(msg.ID),
		TimeStamp: msg.Unixtime,
		AuthorID:  sender.ID,
		Data:      rcgResult.String(),
	}

	if errDB := tb.hdlr.SaveTranscription(ctx, &m); errDB != nil {
		dataChan <- fmt.Sprintf("ошибка при сохранении результатов обработки в БД: %v", errDB)
	}

	return eg.Wait()
}
