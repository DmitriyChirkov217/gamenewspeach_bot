// Программа запускает Telegram-бота для новостного канала: загружает конфигурацию, подключается к PostgreSQL,
// поднимает сервисы сбора RSS-новостей и публикации анонсов, регистрирует административные команды бота,
// стартует HTTP healthcheck и затем обрабатывает входящие обновления Telegram до завершения контекста.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/bot"
	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/bot/middleware"
	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/botkit"
	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/config"
	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/fetcher"
	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/notifier"
	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/storage"
	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/summary"
	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/tagger"
)

// main — точка входа приложения: она связывает конфигурацию из config.Get,
// хранилища storage.NewArticleStorage/storage.NewSourceStorage, фоновые сервисы fetcher.New и notifier.New,
// а также Telegram-команды из пакета bot, после чего запускает их в общом жизненном цикле приложения.
func main() {
	botAPI, err := tgbotapi.NewBotAPI(config.Get().TelegramBotToken)
	if err != nil {
		log.Printf("[ERROR] failed to create botAPI: %v", err)
		return
	}

	db, err := connectDB(config.Get().DatabaseDSN)
	if err != nil {
		log.Printf("[ERROR] failed to connect to db: %v", err)
		return
	}
	defer db.Close()

	if err := storage.Migrate(context.Background(), db); err != nil {
		log.Printf("[ERROR] failed to migrate db: %v", err)
		return
	}

	var (
		articleStorage = storage.NewArticleStorage(db)
		sourceStorage  = storage.NewSourceStorage(db)
		userStorage    = storage.NewUserStorage(db)
		tagger         = tagger.New(
			config.Get().OpenAIKey,
			config.Get().OpenAIModel,
		)
		fetcher = fetcher.New(
			articleStorage,
			sourceStorage,
			tagger,
			config.Get().FetchInterval,
			config.Get().FilterKeywords,
		)
		summarizer = summary.NewOpenAISummarizer(
			config.Get().OpenAIKey,
			config.Get().OpenAIModel,
			config.Get().OpenAIPrompt,
		)
		notifier = notifier.New(
			articleStorage,
			userStorage,
			summarizer,
			botAPI,
			config.Get().NotificationInterval,
			config.Get().NotificationLookback,
		)
	)

	newsBot := botkit.New(botAPI)
	newsBot.RegisterCmdView("start", bot.ViewCmdStart(userStorage))
	newsBot.RegisterCmdView(
		"addsource",
		middleware.AdminsOnly(
			config.Get().TelegramChannelID,
			bot.ViewCmdAddSource(sourceStorage),
		),
	)
	newsBot.RegisterCmdView(
		"setpriority",
		middleware.AdminsOnly(
			config.Get().TelegramChannelID,
			bot.ViewCmdSetPriority(sourceStorage),
		),
	)
	newsBot.RegisterCmdView(
		"getsource",
		middleware.AdminsOnly(
			config.Get().TelegramChannelID,
			bot.ViewCmdGetSource(sourceStorage),
		),
	)
	newsBot.RegisterCmdView(
		"listsources",
		middleware.AdminsOnly(
			config.Get().TelegramChannelID,
			bot.ViewCmdListSource(sourceStorage),
		),
	)
	newsBot.RegisterCmdView(
		"deletesource",
		middleware.AdminsOnly(
			config.Get().TelegramChannelID,
			bot.ViewCmdDeleteSource(sourceStorage),
		),
	)
	newsBot.RegisterCallbackView("reaction:", bot.ViewCallbackReaction(articleStorage))

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go func(ctx context.Context) {
		if err := fetcher.Start(ctx); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Printf("[ERROR] failed to run fetcher: %v", err)
				return
			}

			log.Printf("[INFO] fetcher stopped")
		}
	}(ctx)

	go func(ctx context.Context) {
		if err := notifier.Start(ctx); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Printf("[ERROR] failed to run notifier: %v", err)
				return
			}

			log.Printf("[INFO] notifier stopped")
		}
	}(ctx)

	go func(ctx context.Context) {
		if err := http.ListenAndServe("0.0.0.0:8080", mux); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Printf("[ERROR] failed to run http server: %v", err)
				return
			}

			log.Printf("[INFO] http server stopped")
		}
	}(ctx)

	if err := newsBot.Run(ctx); err != nil {
		log.Printf("[ERROR] failed to run botkit: %v", err)
	}
}

func connectDB(dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err == nil {
		return db, nil
	}

	fallbackDSN, ok := localhostFallbackDSN(dsn)
	if !ok {
		return nil, err
	}

	log.Printf("[WARN] failed to connect using host from dsn, retrying with localhost")

	db, fallbackErr := sqlx.Connect("postgres", fallbackDSN)
	if fallbackErr == nil {
		return db, nil
	}

	return nil, errors.Join(err, fallbackErr)
}

func localhostFallbackDSN(dsn string) (string, bool) {
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Hostname() != "db" {
		return "", false
	}

	port := parsed.Port()
	if port == "" {
		port = "5432"
	}

	// When the app runs on the host machine and Postgres lives in Docker,
	// we expose the container on localhost:5433 to avoid conflicts with a
	// locally installed PostgreSQL service on 5432.
	if port == "5432" {
		port = "5433"
	}

	parsed.Host = "localhost:" + port

	return parsed.String(), !strings.EqualFold(dsn, parsed.String())
}
