package fetcher

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/tomakado/containers/set"

	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/model"
	src "github.com/DmitriyChirkov217/gamenewspeach_bot/internal/source"
)

//go:generate moq --out=mocks/mock_article_storage.go --pkg=mocks . ArticleStorage
type ArticleStorage interface {
	Store(ctx context.Context, article model.Article) error
}

//go:generate moq --out=mocks/mock_sources_provider.go --pkg=mocks . SourcesProvider
type SourcesProvider interface {
	Sources(ctx context.Context) ([]model.Source, error)
}

//go:generate moq --out=mocks/mock_source.go --pkg=mocks . Source
type Source interface {
	ID() int64
	Name() string
	Fetch(ctx context.Context) ([]model.Item, error)
}

type Fetcher struct {
	articles ArticleStorage
	sources  SourcesProvider
	tagger   Tagger

	fetchInterval  time.Duration
	filterKeywords []string
}

type Tagger interface {
	Tags(ctx context.Context, item model.Item) ([]model.ArticleTag, error)
}

// New создает сервис сбора новостей, который получает список источников из SourcesProvider,
// сохраняет статьи через ArticleStorage и запускается из main методом Start.
func New(
	articleStorage ArticleStorage,
	sourcesProvider SourcesProvider,
	tagger Tagger,
	fetchInterval time.Duration,
	filterKeywords []string,
) *Fetcher {
	return &Fetcher{
		articles:       articleStorage,
		sources:        sourcesProvider,
		tagger:         tagger,
		fetchInterval:  fetchInterval,
		filterKeywords: filterKeywords,
	}
}

// Start запускает периодический цикл обновления лент: он вызывает Fetch сразу и затем по таймеру,
// пока контекст приложения из main не будет отменен.
func (f *Fetcher) Start(ctx context.Context) error {
	ticker := time.NewTicker(f.fetchInterval)
	defer ticker.Stop()

	if err := f.Fetch(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := f.Fetch(ctx); err != nil {
				return err
			}
		}
	}
}

// Fetch запрашивает список источников, для каждого создает src.NewRSSSourceFromModel,
// параллельно загружает новости через Source.Fetch и передает результаты в processItems.
func (f *Fetcher) Fetch(ctx context.Context) error {
	sources, err := f.sources.Sources(ctx)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	for _, source := range sources {
		wg.Add(1)

		go func(source Source) {
			defer wg.Done()

			items, err := source.Fetch(ctx)
			if err != nil {
				log.Printf("[ERROR] failed to fetch items from source %q: %v", source.Name(), err)
				return
			}

			if err := f.processItems(ctx, source, items); err != nil {
				log.Printf("[ERROR] failed to process items from source %q: %v", source.Name(), err)
				return
			}
		}(src.NewRSSSourceFromModel(source))
	}

	wg.Wait()

	return nil
}

// processItems нормализует время публикации, фильтрует элементы через itemShouldBeSkipped
// и сохраняет подходящие статьи в постоянное хранилище через ArticleStorage.Store.
func (f *Fetcher) processItems(ctx context.Context, source Source, items []model.Item) error {
	for _, item := range items {
		item.Date = item.Date.UTC()

		if f.itemShouldBeSkipped(item) {
			log.Printf("[INFO] item %q (%s) from source %q should be skipped", item.Title, item.Link, source.Name())
			continue
		}

		tags, err := f.tagger.Tags(ctx, item)
		if err != nil {
			log.Printf("[WARN] failed to tag item %q: %v", item.Title, err)
		}

		if err := f.articles.Store(ctx, model.Article{
			SourceID:    source.ID(),
			Title:       item.Title,
			Link:        item.Link,
			Summary:     item.Summary,
			Tags:        tags,
			PublishedAt: item.Date,
		}); err != nil {
			return err
		}
	}

	return nil
}

// itemShouldBeSkipped проверяет категории и заголовок на совпадение с filterKeywords;
// эту функцию использует processItems, чтобы не сохранять нежелательные новости.
func (f *Fetcher) itemShouldBeSkipped(item model.Item) bool {
	categoriesSet := set.New(item.Categories...)

	for _, keyword := range f.filterKeywords {
		if categoriesSet.Contains(keyword) || strings.Contains(strings.ToLower(item.Title), keyword) {
			return true
		}
	}

	return false
}
