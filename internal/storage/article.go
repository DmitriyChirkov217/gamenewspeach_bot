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

func NewArticleStorage(db *sqlx.DB) *ArticlePostgresStorage {
	return &ArticlePostgresStorage{db: db}
}

func (s *ArticlePostgresStorage) Store(ctx context.Context, article model.Article) error {
	conn, err := s.db.Connx(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	var articleID int64
	row := tx.QueryRowxContext(
		ctx,
		`INSERT INTO articles (source_id, title, link, summary, published_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (link) DO UPDATE SET
		 	source_id = EXCLUDED.source_id
		 RETURNING id;`,
		article.SourceID,
		article.Title,
		article.Link,
		article.Summary,
		article.PublishedAt,
	)
	if err := row.Scan(&articleID); err != nil {
		_ = tx.Rollback()
		return err
	}

	if len(article.Tags) > 0 {
		for _, tag := range article.Tags {
			if _, err := tx.ExecContext(
				ctx,
				`INSERT INTO article_tags (article_id, tag, weight)
				 VALUES ($1, $2, $3)
				 ON CONFLICT (article_id, tag) DO UPDATE SET weight = EXCLUDED.weight;`,
				articleID,
				tag.Tag,
				tag.Weight,
			); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

func (s *ArticlePostgresStorage) RecommendForUser(
	ctx context.Context,
	userID int64,
	since time.Time,
	limit uint64,
) ([]model.Article, error) {
	conn, err := s.db.Connx(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var articles []dbArticleWithScore

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
			a.created_at AS a_created_at,
			COALESCE(SUM(ats.weight * uts.score), 0) AS recommendation_score
		FROM articles a
		JOIN sources s ON s.id = a.source_id
		LEFT JOIN article_tags ats ON ats.article_id = a.id
		LEFT JOIN user_tag_scores uts ON uts.user_id = $1 AND uts.tag = ats.tag
		LEFT JOIN article_deliveries ad ON ad.user_id = $1 AND ad.article_id = a.id
		WHERE ad.article_id IS NULL
			AND a.published_at >= $2::timestamp
		GROUP BY a.id, s.priority, s.id, a.title, a.link, a.summary, a.published_at, a.posted_at, a.created_at
		ORDER BY recommendation_score DESC, s.priority DESC, a.published_at DESC, a.created_at DESC
		LIMIT $3;`,
		userID,
		since.UTC().Format(time.RFC3339),
		limit,
	); err != nil {
		return nil, err
	}

	return lo.Map(articles, func(article dbArticleWithScore, _ int) model.Article {
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

func (s *ArticlePostgresStorage) RecordDelivery(
	ctx context.Context,
	userID int64,
	articleID int64,
	messageID int,
) error {
	conn, err := s.db.Connx(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.ExecContext(
		ctx,
		`INSERT INTO article_deliveries (user_id, article_id, message_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, article_id) DO NOTHING;`,
		userID,
		articleID,
		messageID,
	)

	return err
}

func (s *ArticlePostgresStorage) SaveReaction(
	ctx context.Context,
	userID int64,
	articleID int64,
	reaction int,
) error {
	conn, err := s.db.Connx(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	var previous sql.NullInt16
	if err := tx.GetContext(
		ctx,
		&previous,
		`SELECT reaction
		 FROM article_reactions
		 WHERE user_id = $1 AND article_id = $2;`,
		userID,
		articleID,
	); err != nil && err != sql.ErrNoRows {
		_ = tx.Rollback()
		return err
	}

	delta := reaction
	if previous.Valid {
		delta -= int(previous.Int16)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO article_reactions (user_id, article_id, reaction)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, article_id) DO UPDATE SET
		 	reaction = EXCLUDED.reaction,
		 	updated_at = NOW();`,
		userID,
		articleID,
		reaction,
	); err != nil {
		_ = tx.Rollback()
		return err
	}

	if delta != 0 {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO user_tag_scores (user_id, tag, score)
			 SELECT $1, at.tag, at.weight * $2
			 FROM article_tags at
			 WHERE at.article_id = $3
			 ON CONFLICT (user_id, tag) DO UPDATE SET
			 	score = user_tag_scores.score + EXCLUDED.score,
			 	updated_at = NOW();`,
			userID,
			float64(delta),
			articleID,
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

type dbArticleWithScore struct {
	ID                  int64          `db:"a_id"`
	SourcePriority      int64          `db:"s_priority"`
	SourceID            int64          `db:"s_id"`
	Title               string         `db:"a_title"`
	Link                string         `db:"a_link"`
	Summary             sql.NullString `db:"a_summary"`
	PublishedAt         time.Time      `db:"a_published_at"`
	PostedAt            sql.NullTime   `db:"a_posted_at"`
	CreatedAt           time.Time      `db:"a_created_at"`
	RecommendationScore float64        `db:"recommendation_score"`
}
