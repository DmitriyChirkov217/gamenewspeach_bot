package model

import (
	"time"
)

type Item struct {
	Title      string
	Categories []string
	Link       string
	Date       time.Time
	Summary    string
	SourceName string
}

type Source struct {
	ID        int64
	Name      string
	FeedURL   string
	Priority  int
	CreatedAt time.Time
}

type Article struct {
	ID          int64
	SourceID    int64
	Title       string
	Link        string
	Summary     string
	Tags        []ArticleTag
	PublishedAt time.Time
	PostedAt    time.Time
	CreatedAt   time.Time
}

type ArticleTag struct {
	Tag    string
	Weight float64
}

type User struct {
	TelegramUserID int64
	ChatID         int64
	Username       string
	FirstName      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
