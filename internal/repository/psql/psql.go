// Модуль psql реализует функцонал хранения данных в БД PostgreSQL
package psql

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	psqldrv "github.com/nk87rus/stenographer/internal/db/psql"
)

// PSQLDriver - описывает методы необходимые для взаимодействия с БД PostgreSQL
//
//go:generate go run github.com/vektra/mockery/v2@v2.53.6 --name=PSQLDriver --inpackage --testonly
type PSQLDriver interface {
	GetConnConfig() *pgx.ConnConfig
	Insert(ctx context.Context, req string, args ...any) error
	Exec(ctx context.Context, req string, args ...any) error
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
	req := `INSERT INTO public.users(id, username) VALUES ($1, $2);`
	ctxInsert, cancelInsert := context.WithTimeout(ctx, 5*time.Second)
	defer cancelInsert()
	if err := s.db.Insert(ctxInsert, req, userID, userName); err != nil {
		return fmt.Errorf("RegisterUser: %w", err)
	}
	return nil
}

// // AddBatch - ддобавлет набор записей в хранилище
// //
// // Args:
// //   - data - итератор по добавляемым записям
// func (s *PSQLStorage) AddBatch(ctx context.Context, data iter.Seq[model.StorageRecord]) error {
// 	req := `INSERT INTO public.urls(uuid, short_url, original_url, user_id) VALUES (@uuidValue, @shortURL, @origURL, @userID);`
// 	var args = []pgx.NamedArgs{}

// 	for rec := range data {
// 		args = append(args, pgx.NamedArgs{"uuidValue": rec.UUID, "shortURL": rec.ShortURL, "origURL": rec.OrigURL, "userID": rec.UserID})
// 	}

// 	ctxInsert, cancelInsert := context.WithTimeout(ctx, reqTimeout(len(args)))
// 	defer cancelInsert()
// 	if err := s.db.InsertBatch(ctxInsert, req, args); err != nil {
// 		return fmt.Errorf("AddBatch: %w", err)
// 	}
// 	return nil
// }

// func reqTimeout(value int) time.Duration {
// 	if value <= 5 {
// 		return 5 * time.Second
// 	}
// 	return time.Duration(value+value/2) * time.Second
// }

// // GetShortURL - возвращает короткий URL по его базовому значению
// func (s *PSQLStorage) GetShortURL(ctx context.Context, origURL string) (string, error) {
// 	req := "SELECT short_url FROM public.urls WHERE original_url = $1;"
// 	ctxSelect, cancelSelect := context.WithTimeout(ctx, 5*time.Second)
// 	defer cancelSelect()
// 	result, err := s.db.SelectString(ctxSelect, req, origURL)
// 	if err != nil {
// 		return "", fmt.Errorf("GetSortURL: %w", err)
// 	}
// 	return result, nil
// }

// // func (s *Storage) IDExists(ctx context.Context, sURL string) bool {
// // 	return false
// // }

// // LoadData - загружает данный из хранилища в указанный приёмник
// //
// // Args:
// //   - rcv - приёмник данных из хранилища
// func (s *PSQLStorage) LoadData(ctx context.Context, rcv any) error {
// 	req := `SELECT json_agg(row_to_json(r)) as data FROM (SELECT * FROM public.urls ORDER BY uuid ASC ) r`
// 	ctx, cancelFunc := context.WithTimeout(ctx, 5*time.Second)
// 	defer cancelFunc()
// 	rawData, err := s.db.SelectBytes(ctx, req)
// 	if err != nil {
// 		return fmt.Errorf("LoadData: %w", err)
// 	}

// 	if len(rawData) == 0 {
// 		rawData = []byte("[]")
// 	}

// 	if err := json.Unmarshal(rawData, rcv); err != nil {
// 		return fmt.Errorf("LoadData: %w", err)
// 	}

// 	return nil
// }

// // DelURLs - удаляет записи из хранилища
// //
// // Args:
// //   - uid - идентификатор пользователя-владельца записей
// //   - urls - список удаляемых записей
// func (s *PSQLStorage) DelURLs(ctx context.Context, uid string, urls []string) error {
// 	req := `UPDATE public.urls SET is_deleted = true WHERE user_id = $1 AND short_url = any($2);`
// 	return s.db.Exec(ctx, req, uid, urls)
// }

// // GetStats - собирает статистику
// func (s *PSQLStorage) GetStats(ctx context.Context) (*model.Stats, error) {
// 	req := `SELECT count(*) as urls, count(distinct(user_id)) as users FROM public.urls`
// 	dbCtx, cancelFunc := context.WithTimeout(ctx, 5*time.Second)
// 	defer cancelFunc()

// 	rawData, errDB := s.db.SelectToMap(dbCtx, req)
// 	if errDB != nil {
// 		return nil, errDB
// 	}
// 	var result model.Stats
// 	errDecode := mapstructure.Decode(rawData, &result)
// 	return &result, errDecode
// }
