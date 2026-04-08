// Модуль psql реализует функцонал хранения данных в БД PostgreSQL
package psql

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	psqldrv "github.com/nk87rus/transcriptor/internal/db/psql"
	"github.com/nk87rus/transcriptor/internal/model"
)

// PSQLDriver - описывает методы необходимые для взаимодействия с БД PostgreSQL
//
//go:generate go run github.com/vektra/mockery/v2@v2.53.6 --name=PSQLDriver --inpackage --testonly
type PSQLDriver interface {
	GetConnConfig() *pgx.ConnConfig
	Insert(ctx context.Context, req string, args ...any) error
	Exec(ctx context.Context, req string, args ...any) error
	SelectRow(ctx context.Context, req string, args ...any) pgx.Row
	SelectRows(ctx context.Context, req string, args ...any) (pgx.Rows, error)
}

// PSQLRepo - структура хранилища PostgreSQL
type PSQLRepo struct {
	db PSQLDriver // подключение к БД PostgreSQL
}

// NewPSQLRepo - инициализирует новое PostgreSQL хранилище
func NewPSQLRepo(ctx context.Context, dsn string) (*PSQLRepo, error) {
	dbDrv, errDB := psqldrv.InitPSQL(ctx, dsn)
	if errDB != nil {
		return nil, errDB
	}

	if err := applyMigrations(ctx, dbDrv.GetConnConfig()); err != nil {
		return nil, err
	}
	return &PSQLRepo{db: dbDrv}, nil
}

func (s *PSQLRepo) RegisterUser(ctx context.Context, userID int64, userName string) error {
	req := `INSERT INTO public.users(id, user_name) VALUES ($1, $2);`
	ctxInsert, cancelInsert := context.WithTimeout(ctx, 5*time.Second)
	defer cancelInsert()
	if err := s.db.Insert(ctxInsert, req, userID, userName); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			fmt.Println("Обработана ошибка: Нарушение уникальности (duplicate key)")
			return fmt.Errorf("🤝 В повторной регистрации нет необходимости, Вы уже зарегистрированы.")
		}
		return fmt.Errorf("RegisterUser: %w", err)
	}
	return nil
}

func (s *PSQLRepo) GetTranscriptionsList(ctx context.Context) ([]model.TranscriptionListItem, error) {
	req := "SELECT id, ts, (SELECT user_name FROM public.users u WHERE u.id = m.user_id) as user_name FROM public.transcriptions m"
	ctxSelect, cancelSelect := context.WithTimeout(ctx, 5*time.Second)
	defer cancelSelect()

	rawData, errDB := s.db.SelectRows(ctxSelect, req)
	if errDB != nil {
		return nil, errDB
	}

	return pgx.CollectRows(rawData, pgx.RowToStructByName[model.TranscriptionListItem])

}

func (s *PSQLRepo) GetTranscription(ctx context.Context, mID int64) (*model.Transcription, error) {
	req := "SELECT id, ts, (SELECT user_name FROM public.users u WHERE u.id = m.user_id) as user_name, data FROM public.transcriptions m WHERE id = $1;"
	ctxSelect, cancelSelect := context.WithTimeout(ctx, 5*time.Second)
	defer cancelSelect()

	rawData, errDB := s.db.SelectRows(ctxSelect, req, mID)
	if errDB != nil {
		return nil, errDB
	}
	result, err := pgx.CollectOneRow(rawData, pgx.RowToStructByName[model.Transcription])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &result, err
}

func (s PSQLRepo) SaveTranscription(ctx context.Context, m *model.Transcription) error {
	req := "INSERT INTO public.transcriptions( id, ts, user_id, data) VALUES ($1, $2, $3, $4);"
	ctxInsert, cancelInsert := context.WithTimeout(ctx, 5*time.Second)
	defer cancelInsert()

	if err := s.db.Insert(ctxInsert, req, m.Id, m.TimeStamp, m.AuthorID, m.Data); err != nil {
		return fmt.Errorf("SaveTranscription: %w", err)
	}
	return nil

}

func (s PSQLRepo) SearchTranscriptions(ctx context.Context, wordList []string) ([]model.TranscriptionListItem, error) {
	req := `SELECT id, ts, (SELECT user_name FROM public.users u WHERE u.id = m.user_id) as user_name FROM public.transcriptions m WHERE m.data ~ $1;`
	ctxSelect, cancelSelect := context.WithTimeout(ctx, 5*time.Second)
	defer cancelSelect()

	argWL := strings.Join(wordList, "|")
	rawData, errDB := s.db.SelectRows(ctxSelect, req, argWL)
	if errDB != nil {
		return nil, errDB
	}

	return pgx.CollectRows(rawData, pgx.RowToStructByName[model.TranscriptionListItem])
}
