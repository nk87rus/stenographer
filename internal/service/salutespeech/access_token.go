package salutespeech

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const (
	oauthURL       = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
	accessTokenTTL = 30 * time.Minute
	scope          = "SALUTE_SPEECH_PERS"
)

type Token struct {
	sync.Mutex
	authKey string
	value   string
}

type TokenData struct {
	Token     string `json:"access_token"`
	ExpiresAt int64  `json:"expires_at"`
}

func NewTokenManager(authKey string) *Token {
	return &Token{authKey: authKey}
}

func (at *Token) Get(ctx context.Context) (string, error) {
	log.Debug().Msg("получение токена авторизации")
	at.Lock()
	defer at.Unlock()

	if at.value == "" {
		tokenData, err := at.Fetch(ctx)
		if err != nil {
			return "", fmt.Errorf("ошибка при запросе нового токена: %w", err)
		}
		at.value = tokenData.Token
		log.Debug().Msgf("токен успешно получен, действителен до %s", time.UnixMilli(tokenData.ExpiresAt).String())

		// Remove expired token
		tDur := time.UnixMilli(tokenData.ExpiresAt).Sub(time.Now().Add(-1 * time.Minute))
		time.AfterFunc(tDur, func() {
			log.Debug().Msg("удаление устарешего токена")
			at.Lock()
			defer at.Lock()
			at.value = ""
		})
	}

	return at.value, nil
}

func (at *Token) Fetch(ctx context.Context) (*TokenData, error) {
	log.Debug().Msg("запрос токена авторизации с сервера")
	client := resty.New().
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	var result TokenData
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetHeader("Accept", "application/json").
		SetHeader("RqUID", uuid.NewString()).
		SetAuthScheme("Bearer").SetAuthToken(at.authKey).
		SetFormData(map[string]string{"scope": scope}).
		SetResult(&result).
		Post(oauthURL)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return &result, fmt.Errorf("запрос токена завершился с кодом %s", resp.Status())
	}

	return &result, nil
}
