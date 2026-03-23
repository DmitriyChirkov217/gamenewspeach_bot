package source

import (
	"context"
	"strings"

	"github.com/SlyMarbo/rss"
	"github.com/samber/lo"

	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/model"
)

type RSSSource struct {
	URL        string
	SourceID   int64
	SourceName string
}

// NewRSSSourceFromModel преобразует model.Source в адаптер RSSSource, который затем используется
// в fetcher.Fetch как конкретная реализация интерфейса Source для загрузки элементов из RSS-ленты.
func NewRSSSourceFromModel(m model.Source) RSSSource {
	return RSSSource{
		URL:        m.FeedURL,
		SourceID:   m.ID,
		SourceName: m.Name,
	}
}

// Fetch загружает RSS-ленту через loadFeed, превращает элементы rss.Item в model.Item
// и подготавливает их для дальнейшей обработки в fetcher.processItems.
func (s RSSSource) Fetch(ctx context.Context) ([]model.Item, error) {
	feed, err := s.loadFeed(ctx, s.URL)
	if err != nil {
		return nil, err
	}

	return lo.Map(feed.Items, func(item *rss.Item, _ int) model.Item {
		return model.Item{
			Title:      item.Title,
			Categories: item.Categories,
			Link:       item.Link,
			Date:       item.Date,
			SourceName: s.SourceName,
			Summary:    strings.TrimSpace(item.Summary),
		}
	}), nil
}

// ID возвращает идентификатор источника; его вызывает fetcher.processItems,
// когда сохраняет статьи в storage.ArticlePostgresStorage.Store.
func (s RSSSource) ID() int64 {
	return s.SourceID
}

// Name возвращает человекочитаемое имя источника; оно используется в fetcher.Fetch
// и fetcher.processItems для диагностических сообщений и логирования.
func (s RSSSource) Name() string {
	return s.SourceName
}

// loadFeed выполняет блокирующую загрузку RSS через отдельную goroutine и позволяет Fetch
// отменить ожидание по context.Context, чтобы сборщик fetcher.Start мог корректно останавливаться.
func (s RSSSource) loadFeed(ctx context.Context, url string) (*rss.Feed, error) {
	var (
		feedCh = make(chan *rss.Feed)
		errCh  = make(chan error)
	)

	go func() {
		feed, err := rss.Fetch(url)
		if err != nil {
			errCh <- err
			return
		}
		feedCh <- feed
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errCh:
		return nil, err
	case feed := <-feedCh:
		return feed, nil
	}
}
