package handler

import (
	"context"
	"iter"

	"github.com/nk87rus/transcriptor/internal/model"
)

type RepoMgr interface {
	RegisterUser(ctx context.Context, userID int64, userName string) *model.DBResponse
	GetTranscriptionsList(ctx context.Context) iter.Seq2[model.TranscriptionListItem, error]
	GetTranscription(ctx context.Context, mID int64) (*model.Transcription, error)
	SaveTranscription(ctx context.Context, m *model.Transcription) error
	SearchTranscriptions(ctx context.Context, wordList []string) iter.Seq2[model.TranscriptionListItem, error]
}

type SpeechRecognizer interface {
	Recognize(ctx context.Context, srcFile *model.AudioFile) iter.Seq2[string, error]
}

type AIChat interface {
	Chat(ctx context.Context, request string) iter.Seq2[string, error]
}

type Handler struct {
	repo   RepoMgr
	sr     SpeechRecognizer
	aiChat AIChat
}

func Init(storage RepoMgr, sr SpeechRecognizer, aiChat AIChat) *Handler {
	return &Handler{repo: storage, sr: sr, aiChat: aiChat}
}

func (h *Handler) RegisterUser(ctx context.Context, userID int64, userName string) *model.DBResponse {
	return h.repo.RegisterUser(ctx, userID, userName)
}

func (h *Handler) GetTranscriptionsList(ctx context.Context) iter.Seq2[model.TranscriptionListItem, error] {
	return h.repo.GetTranscriptionsList(ctx)
}

func (h *Handler) GetTranscription(ctx context.Context, mID int64) (*model.Transcription, error) {
	return h.repo.GetTranscription(ctx, mID)
}

func (h *Handler) SearchTranscriptions(ctx context.Context, wordList []string) iter.Seq2[model.TranscriptionListItem, error] {
	return h.repo.SearchTranscriptions(ctx, wordList)
}

func (h *Handler) SaveTranscription(ctx context.Context, m *model.Transcription) error {
	return h.repo.SaveTranscription(ctx, m)
}

func (h *Handler) RecognizeAudio(ctx context.Context, srcFile *model.AudioFile) iter.Seq2[string, error] {
	return h.sr.Recognize(ctx, srcFile)
}

func (h *Handler) RecognizeVoice(ctx context.Context, srcFile *model.AudioFile) iter.Seq2[string, error] {
	return h.sr.Recognize(ctx, srcFile)
}

func (h *Handler) AIChat(ctx context.Context, request string) iter.Seq2[string, error] {
	return h.aiChat.Chat(ctx, request)
}
