package app

import (
	"context"
	"os"

	"github.com/nk87rus/stenographer/internal/config"
	"github.com/nk87rus/stenographer/internal/handler"

	"github.com/nk87rus/stenographer/internal/logger"
	"github.com/nk87rus/stenographer/internal/repository/psql"
	"github.com/nk87rus/stenographer/internal/service/salutespeech"
	"github.com/nk87rus/stenographer/internal/service/telegram"
	"golang.org/x/sync/errgroup"
)

type TaskProvider interface {
	Run(context.Context) error
}

type SpeechRecognizer interface {
}

type App struct {
	tp TaskProvider
	sr SpeechRecognizer
}

func Init(ctx context.Context) (*App, error) {
	logger.Init()

	cfg, err := config.InitConfig(os.Args)
	if err != nil {
		return nil, err
	}

	repo, errRepo := psql.NewPSQLRepo(ctx, cfg.DBDSN)
	if errRepo != nil {
		return nil, errRepo
	}

	teleBot, errTR := telegram.InitBot(ctx, cfg.TaskProvToken, handler.Init(repo))
	if errTR != nil {
		return nil, errTR
	}

	salutSpeech, errSR := salutespeech.Init(ctx, cfg.SpeechRecKey)
	if errSR != nil {
		return nil, errSR
	}

	return &App{tp: teleBot, sr: salutSpeech}, nil
}

func (a *App) Run(ctx context.Context) error {
	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		a.tp.Run(egCtx)
		return nil
	})

	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}
