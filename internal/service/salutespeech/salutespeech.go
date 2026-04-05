package salutespeech

import (
	"context"
)

const address = "smartspeech.sber.ru"

type Tokenizer interface {
	Get(context.Context) (string, error)
}

type SaluteSpeechClient struct {
	token Tokenizer
}

func Init(ctx context.Context, authKey string) (*SaluteSpeechClient, error) {
	return &SaluteSpeechClient{token: NewTokenManager(authKey)}, nil
}

func (s SaluteSpeechClient) GetAccessToken(ctx context.Context) error {
	return nil
}
