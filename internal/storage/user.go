package storage

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/samber/lo"

	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/model"
)

type UserPostgresStorage struct {
	db *sqlx.DB
}

func NewUserStorage(db *sqlx.DB) *UserPostgresStorage {
	return &UserPostgresStorage{db: db}
}

func (s *UserPostgresStorage) Upsert(ctx context.Context, user model.User) error {
	conn, err := s.db.Connx(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.ExecContext(
		ctx,
		`INSERT INTO users (telegram_user_id, chat_id, username, first_name)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (telegram_user_id) DO UPDATE SET
		 	chat_id = EXCLUDED.chat_id,
		 	username = EXCLUDED.username,
		 	first_name = EXCLUDED.first_name,
		 	updated_at = NOW();`,
		user.TelegramUserID,
		user.ChatID,
		user.Username,
		user.FirstName,
	)

	return err
}

func (s *UserPostgresStorage) Subscribers(ctx context.Context) ([]model.User, error) {
	conn, err := s.db.Connx(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var users []dbUser
	if err := conn.SelectContext(
		ctx,
		&users,
		`SELECT telegram_user_id, chat_id, username, first_name, created_at, updated_at
		 FROM users
		 ORDER BY created_at ASC;`,
	); err != nil {
		return nil, err
	}

	return lo.Map(users, func(user dbUser, _ int) model.User {
		return model.User{
			TelegramUserID: user.TelegramUserID,
			ChatID:         user.ChatID,
			Username:       user.Username,
			FirstName:      user.FirstName,
			CreatedAt:      user.CreatedAt,
			UpdatedAt:      user.UpdatedAt,
		}
	}), nil
}

type dbUser struct {
	TelegramUserID int64     `db:"telegram_user_id"`
	ChatID         int64     `db:"chat_id"`
	Username       string    `db:"username"`
	FirstName      string    `db:"first_name"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}
