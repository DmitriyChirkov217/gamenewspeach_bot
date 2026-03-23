package storage

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/samber/lo"

	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/model"
)

type SourcePostgresStorage struct {
	db *sqlx.DB
}

// NewSourceStorage создает PostgreSQL-хранилище источников, которое main передает
// в fetcher.New и в команды управления источниками из пакета bot.
func NewSourceStorage(db *sqlx.DB) *SourcePostgresStorage {
	return &SourcePostgresStorage{db: db}
}

// Sources возвращает все зарегистрированные источники и используется в fetcher.Fetch
// и bot.ViewCmdListSource для фонового обхода и административного списка.
func (s *SourcePostgresStorage) Sources(ctx context.Context) ([]model.Source, error) {
	conn, err := s.db.Connx(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var sources []dbSource
	if err := conn.SelectContext(ctx, &sources, `SELECT * FROM sources`); err != nil {
		return nil, err
	}

	return lo.Map(sources, func(source dbSource, _ int) model.Source { return model.Source(source) }), nil
}

// SourceByID возвращает один источник по ID; этот метод используется обработчиком bot.ViewCmdGetSource.
func (s *SourcePostgresStorage) SourceByID(ctx context.Context, id int64) (*model.Source, error) {
	conn, err := s.db.Connx(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var source dbSource
	if err := conn.GetContext(ctx, &source, `SELECT * FROM sources WHERE id = $1`, id); err != nil {
		return nil, err
	}

	return (*model.Source)(&source), nil
}

// Add сохраняет новый источник RSS в базе и используется командой bot.ViewCmdAddSource,
// чтобы пополнять список лент для дальнейшей работы fetcher.Fetch.
func (s *SourcePostgresStorage) Add(ctx context.Context, source model.Source) (int64, error) {
	conn, err := s.db.Connx(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	var id int64

	row := conn.QueryRowxContext(
		ctx,
		`INSERT INTO sources (name, feed_url, priority)
					VALUES ($1, $2, $3) RETURNING id;`,
		source.Name, source.FeedURL, source.Priority,
	)

	if err := row.Err(); err != nil {
		return 0, err
	}

	if err := row.Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

// SetPriority обновляет приоритет источника; этот метод вызывает bot.ViewCmdSetPriority,
// чтобы повлиять на порядок выбора новостей в notifier.AllNotPosted.
func (s *SourcePostgresStorage) SetPriority(ctx context.Context, id int64, priority int) error {
	conn, err := s.db.Connx(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.ExecContext(ctx, `UPDATE sources SET priority = $1 WHERE id = $2`, priority, id)

	return err
}

// Delete удаляет источник из базы; его использует bot.ViewCmdDeleteSource
// для административного исключения ленты из дальнейшего обхода fetcher.Fetch.
func (s *SourcePostgresStorage) Delete(ctx context.Context, id int64) error {
	conn, err := s.db.Connx(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `DELETE FROM sources WHERE id = $1`, id); err != nil {
		return err
	}

	return nil
}

type dbSource struct {
	ID        int64     `db:"id"`
	Name      string    `db:"name"`
	FeedURL   string    `db:"feed_url"`
	Priority  int       `db:"priority"`
	CreatedAt time.Time `db:"created_at"`
}
