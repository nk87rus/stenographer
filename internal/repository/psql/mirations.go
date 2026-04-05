package psql

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/nk87rus/stenographer/migrations"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/rs/zerolog/log"
)

func applyMigrations(ctx context.Context, connCfg *pgx.ConnConfig) error {
	var db = stdlib.OpenDB(*connCfg)
	defer func() {
		if err := db.Close(); err != nil {
			log.Err(err)
		}
	}()

	migs, err := fs.Sub(migrations.EmbedPSQLMigrations, "psql")
	if err != nil {
		return fmt.Errorf("ошибка при получении списка миграций: %w", err)
	}

	store, err := database.NewStore(goose.DialectPostgres, "stenographer_goose_db_version")
	if err != nil {
		return fmt.Errorf("ошибка при ингициализации хранилища миграций: %w", err)
	}

	provider, err := goose.NewProvider("", db, migs,
		goose.WithDisableGlobalRegistry(true),
		goose.WithAllowOutofOrder(false),
		goose.WithStore(store),
	)
	if err != nil {
		if strings.EqualFold(err.Error(), "no migratiopns found") {
			log.Info().Msg("Не найдено ни одной миграции")
			return nil
		}
		return fmt.Errorf("ошибка при инициализации провайдера миграций: %w", err)
	}

	hasPendingMigrations, err := provider.HasPending(ctx)
	if err != nil {
		return fmt.Errorf("ошибка при формировании списка миграций к применению: %w", err)
	}

	if !hasPendingMigrations {
		log.Info().Msg("Не найдено миграций к применнению")
		return nil
	}

	migResults, err := provider.Up(ctx)
	if err != nil {
		rbmErr := rollbackMigration(ctx, provider)
		return errors.Join(err, rbmErr)
	}

	for _, m := range migResults {
		log.Info().Str("migration", m.Source.Path).Dur("duration", m.Duration.Abs()).Msg("Миграция применена успешно")
	}
	return nil
}

func rollbackMigration(ctx context.Context, mProv *goose.Provider) error {
	log.Warn().Msg("Инициирована процедура отказа применённых миграций")
	resDown, err := mProv.Down(ctx)
	if err != nil {
		return fmt.Errorf("ошибка при откате применённых миграций: %w", err)
	}
	log.Warn().Str("migration", resDown.Source.Path).Msg("Миграция успешно отменена")
	return nil
}
