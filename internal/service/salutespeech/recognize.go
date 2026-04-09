package salutespeech

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/nk87rus/transcriptor/internal/model"
	"github.com/rs/zerolog/log"
)

const (
	uploadURI      = "https://smartspeech.sber.ru/rest/v1/data:upload"
	createTaskURI  = "https://smartspeech.sber.ru/rest/v1/speech:async_recognize"
	pollTaskURI    = "https://smartspeech.sber.ru/rest/v1/task:get"
	fetchResultURI = "https://smartspeech.sber.ru/rest/v1/data:download"
)

type (
	FileUploadResponse struct {
		Status int          `json:"status"`
		Result FUResultData `json:"result"`
	}

	FUResultData struct {
		FileID string `json:"request_file_id"`
	}
)

type (
	TaskReq struct {
		Options   TReqOptions `json:"options"`
		ReqFileID string      `json:"request_file_id"`
	}

	TReqOptions struct {
		Model                    string           `json:"model"`          // general, callcenter
		Encoding                 string           `json:"audio_encoding"` // PCM_S16LE, OPUS, MP3, FLAC, ALAW, MULAW
		SampleRate               int              `json:"sample_rate"`
		Lang                     string           `json:"language"` // ru-RU, en-US, kk-KZ, ky-KG, uz-UZ; default: ru-RU
		ProfanityFilter          bool             `json:"enable_profanity_filter"`
		HypothesesCount          int              `json:"hypotheses_count"`   // default: 1
		NoSpeechTimeout          string           `json:"no_speech_timeout"`  // default: 7s
		MaxSpeechTimeout         string           `json:"max_speech_timeout"` // default: 20s
		Hints                    TaskReqOptHints  `json:"hints"`
		ChannelsCount            int              `json:"channels_count"` // default: 1
		SpeakerSeparationOptions TaskReqOptSSOpts `json:"speaker_separation_options"`
		InsightModels            []string         `json:"insight_models"` // csi; call_features; is_solved; csi, call_features, is_solved.
	}

	TaskReqOptHints struct {
		Words         []string `json:"words"`
		EnableLetters bool     `json:"enable_letters"` // default: false
		EOUTimeout    string   `json:"eou_timeout"`    // default: 1
	}

	TaskReqOptSSOpts struct {
		Enable          bool `json:"enable"`                   // default: false
		OnlyMainSpeaker bool `json:"enable_only_main_speaker"` // default: false
		Count           int  `json:"count"`                    // default: 1
	}
)

type (
	RecognizeResult struct {
		Results []RecognizeResultItem
	}

	RecognizeResultItem struct {
		Text     string `json:"text"`
		NormText string `json:"normalized_text"`
	}
)

type (
	TaskResp struct {
		Status int            `json:"status"`
		Result TaskRespResult `json:"result"`
	}
	TaskRespResult struct {
		ID        string    `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Status    string    `json:"status"` // NEW, RUNNING, CANCELED, DONE, ERROR
	}
)

type (
	TaskPollResp struct {
		Status int `json:"status"`
		Result TaskPollResult
	}
	TaskPollResult struct {
		TaskRespResult
		RespFileID string `json:"response_file_id"`
	}
)

func (s *SaluteSpeechClient) Recognize(ctx context.Context, srcFile *model.AudioFile) iter.Seq2[string, error] {
	reqID := uuid.NewString()
	log.Info().Str("service", "salutespeech").Str("id", reqID).Msgf("начало процедуры распознавания речи для %q", srcFile.LocalFilePath)
	defer log.Info().Str("service", "salutespeech").Str("id", reqID).Msgf("завершение процедуры распознавания речи для %q", srcFile.LocalFilePath)

	log.Debug().Str("service", "salutespeech").Str("id", reqID).Str("fileName", srcFile.LocalFilePath).Msg("загрузка файла на сервер")
	fileID, errUpload := s.uploadFile(ctx, reqID, srcFile)
	if errUpload != nil {
		return func(yield func(string, error) bool) {
			yield("", errUpload)
		}
	}

	log.Debug().Str("service", "salutespeech").Str("id", reqID).Str("uploadedFileID", fileID).Msg("создание задачи на распознавание текста")
	taskID, errCreateTask := s.createTask(ctx, reqID, fileID, srcFile.Encoding)
	if errCreateTask != nil {
		return func(yield func(string, error) bool) {
			yield("", errCreateTask)
		}
	}

	var rFileID string
polling:
	for {
		log.Debug().Str("service", "salutespeech").Str("id", reqID).Str("taskID", taskID).Msg("запрос статуса задачи")
		taskState, errPoll := s.pollTask(ctx, taskID)
		if errPoll != nil {
			return func(yield func(string, error) bool) {
				yield("", errPoll)
			}
		}
		log.Debug().Str("service", "salutespeech").Str("id", reqID).Str("taskID", taskID).Str("state", taskState.Result.Status).Msg("получен статус задачи")
		switch taskState.Result.Status {
		case "DONE":
			rFileID = taskState.Result.RespFileID
			break polling
		case "ERROR":
			return func(yield func(string, error) bool) {
				yield("", fmt.Errorf("распознание текста завершилось ошибкой"))
			}
		case "CANCELED":
			return func(yield func(string, error) bool) {
				yield("", fmt.Errorf("распознание текста отменено"))
			}
		default:
			time.Sleep(3 * time.Second)
		}
	}

	log.Debug().Str("service", "salutespeech").Str("id", reqID).Str("resultFileID", rFileID).Msg("получение результата распознавания текста")
	return s.fetchResult(ctx, rFileID)
}

func (s *SaluteSpeechClient) uploadFile(ctx context.Context, reqID string, file *model.AudioFile) (string, error) {
	token, errToken := s.token.Get(ctx)
	if errToken != nil {
		return "", errToken
	}

	fileData, fileErr := os.ReadFile(file.LocalFilePath)
	if fileErr != nil {
		return "", fmt.Errorf("ошибка открытия файла: %w", fileErr)
	}

	client := resty.New().
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	var result FileUploadResponse
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("X-Request-ID", reqID).
		SetHeader("Accept", "application/json").
		SetAuthScheme("Bearer").SetAuthToken(token).
		SetHeader("Content-Type", file.MIME).
		SetBody(fileData).
		SetResult(&result).
		Post(uploadURI)
	if err != nil {
		return "", err
	}

	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("%s", resp.String())
	}

	return result.Result.FileID, nil
}

func (s *SaluteSpeechClient) createTask(ctx context.Context, reqID, fileID, audioEncoding string) (string, error) {
	token, errToken := s.token.Get(ctx)
	if errToken != nil {
		return "", errToken
	}

	payload := TaskReq{
		Options: TReqOptions{
			Model:            "general",
			Encoding:         audioEncoding,
			SampleRate:       8000,
			Lang:             "ru-RU",
			HypothesesCount:  1,
			NoSpeechTimeout:  "2s",
			MaxSpeechTimeout: "20s",
			Hints: TaskReqOptHints{
				EOUTimeout: "1s",
			},
			ChannelsCount: 1,
			SpeakerSeparationOptions: TaskReqOptSSOpts{
				Count: 1,
			},
		},
		ReqFileID: fileID,
	}

	client := resty.New().
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	var result TaskResp
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("X-Request-ID", reqID).
		SetHeader("Accept", "application/json").
		SetAuthScheme("Bearer").SetAuthToken(token).
		SetBody(payload).
		SetResult(&result).
		Post(createTaskURI)

	if err != nil {
		return "", err
	}

	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("%s", resp.String())
	}

	return result.Result.ID, nil
}

func (s *SaluteSpeechClient) pollTask(ctx context.Context, taskID string) (*TaskPollResp, error) {
	token, errToken := s.token.Get(ctx)
	if errToken != nil {
		return nil, errToken
	}

	client := resty.New().
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	var result TaskPollResp
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/octet-stream").
		SetAuthScheme("Bearer").SetAuthToken(token).
		SetQueryParam("id", taskID).
		SetResult(&result).
		Get(pollTaskURI)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("%s", resp.String())
	}

	return &result, nil
}

func (s *SaluteSpeechClient) fetchResult(ctx context.Context, fileID string) iter.Seq2[string, error] {
	token, errToken := s.token.Get(ctx)
	if errToken != nil {
		return func(yield func(string, error) bool) {
			yield("", errToken)
		}
	}

	client := resty.New().
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/octet-stream").
		SetAuthScheme("Bearer").SetAuthToken(token).
		SetQueryParam("response_file_id", fileID).
		SetDoNotParseResponse(true).
		Get(fetchResultURI)

	if err != nil {
		return func(yield func(string, error) bool) {
			yield("", err)
		}
	}

	if resp.StatusCode() != 200 {
		return func(yield func(string, error) bool) {
			yield("", fmt.Errorf("%s", resp.String()))
			resp.RawBody().Close()
		}
	}

	return parseResults(resp.RawBody(), resp.RawBody().Close)
}

func parseResults(rawData io.Reader, finalCallback func() error) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		var data []RecognizeResult
		if errDecode := json.NewDecoder(rawData).Decode(&data); errDecode != nil {
			yield("", errDecode)
			finalCallback()
			return
		}

		for _, r := range data {
			for _, ri := range r.Results {
				if ri.NormText != "" {
					if !yield(ri.NormText, nil) {
						finalCallback()
						return
					}
				}
			}
		}
	}
}
