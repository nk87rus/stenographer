package gigachat

import (
	"context"
	"crypto/tls"
	"fmt"
	"iter"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const chatURI = "https://gigachat.devices.sberbank.ru/api/v1/chat/completions"

type (
	ChatRequest struct {
		Model    string       `json:"model"` // https://developers.sber.ru/docs/ru/gigachat/models/main
		Messages []ReqMessage `json:"messages"`
		// FunctionCall string        `json:"function_call"`
		// Functions    UserFunctions `json:"functions"`
		// Temperature       float32       `json:"temperature"`
		// TopP              float32       `json:"top_p"`
		Stream bool `json:"stream"`
		// MaxTokens         int32         `json:"max_tokens"`
		// RepetitionPenalty float32       `json:"repetition_penalty"`
		UpdateInterval int `json:"update_interval"`
	}

	ReqMessage struct {
		Role    string `json:"role"` // user, system, assistant, function
		Content string `json:"content"`
		// FuncStateID string `json:"functions_state_id"`
		// Attachments []string `json:"attachments"`
	}

	// UserFunctions struct{}
)

type (
	ChatResponse struct {
		Choices []Choice  `json:"choices"`
		Created int64     `json:"created"`
		Model   string    `json:"model"`
		Usage   UsageData `json:"usage"`
		Object  string    `json:"object"`
	}

	Choice struct {
		Message      RespMessage `json:"message"`
		Index        int32       `json:"index"`
		FinishReason string      `json:"finish_reason"`
	}

	RespMessage struct {
		Role         string           `json:"role"` // user, system, assistant, function
		Content      string           `json:"content"`
		Created      int64            `json:"created"`
		Name         string           `json:"name"`
		FuncStateID  string           `json:"functions_state_id"`
		FunctionCall FunctionCallData `json:"function_call"`
	}

	FunctionCallData struct {
		Name string         `json:"name"`
		Args map[string]any `json:"arguments"`
	}

	UsageData struct {
		PromptTokens          int32 `json:"prompt_tokens"`
		CompletionTokens      int32 `json:"completion_tokens"`
		PrecachedPromptTokens int32 `json:"precached_prompt_tokens"`
		TotalTokens           int32 `json:"total_tokens"`
	}
)

func (g *GigaChatClient) Chat(ctx context.Context, request string) iter.Seq2[string, error] {
	reqID := uuid.NewString()
	log.Info().Str("service", "gigachat").Str("id", reqID).Msg("начало обработки запроса пользователя")
	defer log.Info().Str("service", "gigachat").Str("id", reqID).Msg("завершение обработки запроса пользователя")

	req := ChatRequest{
		Model:          "GigaChat-2-Max",
		Messages:       []ReqMessage{{Role: "user", Content: request}},
		Stream:         false,
		UpdateInterval: 0,
	}

	token, errToken := g.token.Get(ctx)
	if errToken != nil {
		return func(yield func(string, error) bool) {
			yield("", errToken)
		}
	}

	var result ChatResponse
	client := resty.New().SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("X-Request-ID", reqID).
		SetHeader("Accept", "application/json").
		SetAuthScheme("Bearer").SetAuthToken(token).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&result).
		Post(chatURI)

	if err != nil {
		return func(yield func(string, error) bool) {
			yield("", errToken)
		}
	}

	if resp.StatusCode() != 200 {
		return func(yield func(string, error) bool) {
			yield("", fmt.Errorf("%s", resp.String()))
		}
	}

	return parseChatResponse(&result)
}

func parseChatResponse(data *ChatResponse) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		for _, c := range data.Choices {
			if c.Message.Content != "" {
				if !yield(c.Message.Content, nil) {
					return
				}
			}
		}
	}
}
