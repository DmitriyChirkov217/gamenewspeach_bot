package main

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	db, err := sqlx.Connect("postgres", "postgres://postgres:postgres@localhost:5433/news_feed_bot?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var (
		sourcesCount   int
		articlesCount  int
		notPostedCount int
	)

	if err := db.Get(&sourcesCount, `SELECT COUNT(*) FROM sources`); err != nil {
		log.Fatal(err)
	}

	if err := db.Get(&articlesCount, `SELECT COUNT(*) FROM articles`); err != nil {
		log.Fatal(err)
	}

	if err := db.Get(&notPostedCount, `SELECT COUNT(*) FROM articles WHERE posted_at IS NULL`); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("sources=%d\narticles=%d\nnot_posted=%d\n", sourcesCount, articlesCount, notPostedCount)
}
