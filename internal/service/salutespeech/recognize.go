package salutespeech

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"iter"
	"os"

	"github.com/google/uuid"
	pbs "github.com/nk87rus/stenographer/internal/service/salutespeech/proto/storage"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const chunkSize = 4 * 1024 * 1024

func (s *SaluteSpeechClient) Recognize(ctx context.Context, src string) error {
	reqID := uuid.NewString()
	log.Info().Str("id", reqID).Msgf("начало процедуры распознавания речи для %q", src)
	defer log.Info().Str("id", reqID).Msgf("завершение процедуры распознавания речи для %q", src)

	// получение токена
	token, err := s.token.Get(ctx)
	if err != nil {
		return err
	}

	conn, err := grpc.NewClient(address)
	if err != nil {
		return err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Err(err)
		}
	}()

	md := metadata.New(map[string]string{
		"authorization": fmt.Sprintf("Bearer %s", token),
		"x-request-id":  reqID,
	})
	reqCtx := metadata.NewOutgoingContext(ctx, md)
	uploadFile(reqCtx, reqID, conn, src)

	return nil
}

func uploadFile(ctx context.Context, reqID string, conn *grpc.ClientConn, filePath string) error {
	file, fileErr := os.Open(filePath)
	if fileErr != nil {
		return fmt.Errorf("ошибка открытия файла: %w", fileErr)
	}
	defer file.Close()

	cli := pbs.NewSmartSpeechClient(conn)
	stream, streamErr := cli.Upload(ctx)
	if streamErr != nil {
		return streamErr
	}

	var chunkNum uint = 0
	for chunk := range fileChunks(file, chunkSize) {
		chunkNum++
		log.Debug().Str("id", reqID).Msgf("отправка %d чанка", chunkNum)
		req := pbs.UploadRequest{}
		req.SetFileChunk(chunk)
		if err := stream.Send(&req); err != nil {
			return fmt.Errorf("ошибка отправки %d чанка: %w", chunkNum, err)
		}
	}

	return nil
}

func fileChunks(file *os.File, chunkSize int64) iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {

		reader := bufio.NewReader(file)
		buf := make([]byte, chunkSize)

		for {
			n, err := reader.Read(buf)
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, buf[:n])
				if !yield(chunk) {
					return // итератор прерван
				}
			}
			if err != nil {
				if err != io.EOF {
					// Логируем ошибку, но не прерываем yield (можно и обработать)
					fmt.Fprintf(os.Stderr, "ошибка чтения: %v\n", err)
				}
				return
			}
		}
	}
}
