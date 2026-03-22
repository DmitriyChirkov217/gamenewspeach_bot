package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/samber/lo"

	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/model"
)

type ArticlePostgresStorage struct {
	db *sqlx.DB
}

// NewArticleStorage создает PostgreSQL-реализацию хранилища статей, которую используют
// fetcher.New для сохранения новостей и notifier.New для поиска/пометки опубликованных записей.
func NewArticleStorage(db *sqlx.DB) *ArticlePostgresStorage {
	return &ArticlePostgresStorage{db: db}
}

// Store сохраняет новую статью в таблицу articles и вызывается из fetcher.processItems,
// чтобы зафиксировать результаты очередного обхода RSS-источников.
func (s *ArticlePostgresStorage) Store(ctx context.Context, article model.Article) error {
	conn, err := s.db.Connx(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(
		ctx,
		`INSERT INTO articles (source_id, title, link, summary, published_at)
	    				VALUES ($1, $2, $3, $4, $5)
	    				ON CONFLICT DO NOTHING;`,
		article.SourceID,
		article.Title,
		article.Link,
		article.Summary,
		article.PublishedAt,
	); err != nil {
		return err
	}

	return nil
}

// AllNotPosted возвращает еще не опубликованные статьи за заданный интервал;
// его использует notifier.SelectAndSendArticle, чтобы выбрать следующий пост в канал.
func (s *ArticlePostgresStorage) AllNotPosted(ctx context.Context, since time.Time, limit uint64) ([]model.Article, error) {
	conn, err := s.db.Connx(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var articles []dbArticleWithPriority

	if err := conn.SelectContext(
		ctx,
		&articles,
		`SELECT 
				a.id AS a_id, 
				s.priority AS s_priority,
				s.id AS s_id,
				a.title AS a_title,
				a.link AS a_link,
				a.summary AS a_summary,
				a.published_at AS a_published_at,
				a.posted_at AS a_posted_at,
				a.created_at AS a_created_at
			FROM articles a JOIN sources s ON s.id = a.source_id
			WHERE a.posted_at IS NULL 
				AND a.published_at >= $1::timestamp
			ORDER BY a.created_at DESC, s_priority DESC LIMIT $2;`,
		since.UTC().Format(time.RFC3339),
		limit,
	); err != nil {
		return nil, err
	}

	return lo.Map(articles, func(article dbArticleWithPriority, _ int) model.Article {
		return model.Article{
			ID:          article.ID,
			SourceID:    article.SourceID,
			Title:       article.Title,
			Link:        article.Link,
			Summary:     article.Summary.String,
			PublishedAt: article.PublishedAt,
			CreatedAt:   article.CreatedAt,
		}
	}), nil
}

// MarkAsPosted отмечает статью как опубликованную после успешной отправки;
// этот метод вызывается из notifier.SelectAndSendArticle.
func (s *ArticlePostgresStorage) MarkAsPosted(ctx context.Context, article model.Article) error {
	conn, err := s.db.Connx(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(
		ctx,
		`UPDATE articles SET posted_at = $1::timestamp WHERE id = $2;`,
		time.Now().UTC().Format(time.RFC3339),
		article.ID,
	); err != nil {
		return err
	}

	return nil
}

type dbArticleWithPriority struct {
	ID             int64          `db:"a_id"`
	SourcePriority int64          `db:"s_priority"`
	SourceID       int64          `db:"s_id"`
	Title          string         `db:"a_title"`
	Link           string         `db:"a_link"`
	Summary        sql.NullString `db:"a_summary"`
	PublishedAt    time.Time      `db:"a_published_at"`
	PostedAt       sql.NullTime   `db:"a_posted_at"`
	CreatedAt      time.Time      `db:"a_created_at"`
}
