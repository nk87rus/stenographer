package config

import (
	"fmt"
	"strings"

	"github.com/caarlos0/env/v6"
)

type ConfigData struct {
	TaskProvToken string `env:"TASK_PROVIDER_TOKEN"`
	SpeechRecKey  string `env:"SPEECH_REC_KEY"`
	DBDSN         string `env:"DATABASE_DSN"`
}

func InitConfig(args []string) (*ConfigData, error) {
	var cfg ConfigData
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}

	if strings.TrimSpace(cfg.SpeechRecKey) == "" {
		return nil, fmt.Errorf("не найден DATABASE_DSN")
	}

	if strings.TrimSpace(cfg.TaskProvToken) == "" {
		return nil, fmt.Errorf("не найден TASK_PROVIDER_TOKEN")
	}

	if strings.TrimSpace(cfg.SpeechRecKey) == "" {
		return nil, fmt.Errorf("не найден SPEECH_REC_TOKEN")
	}

	return &cfg, nil
}
