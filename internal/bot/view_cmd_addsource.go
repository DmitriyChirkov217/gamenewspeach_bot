package bot

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/botkit"
	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/model"
)

type SourceStorage interface {
	Add(ctx context.Context, source model.Source) (int64, error)
}

// ViewCmdAddSource создает обработчик команды /addsource: он разбирает аргументы через botkit.ParseJSON,
// сохраняет новый источник через SourceStorage.Add и отправляет пользователю подтверждение.
func ViewCmdAddSource(storage SourceStorage) botkit.ViewFunc {
	type addSourceArgs struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		Priority int    `json:"priority"`
	}

	return func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
		args, err := botkit.ParseJSON[addSourceArgs](update.Message.CommandArguments())
		if err != nil {
			msg := tgbotapi.NewMessage(
				update.Message.Chat.ID,
				"Использование:\n/addsource {\"name\":\"Steam\",\"url\":\"https://store.steampowered.com/feeds/news.xml\",\"priority\":10}",
			)
			_, _ = bot.Send(msg)
			return nil
		}

		source := model.Source{
			Name:     args.Name,
			FeedURL:  args.URL,
			Priority: args.Priority,
		}

		sourceID, err := storage.Add(ctx, source)
		if err != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Не удалось добавить источник")
			_, _ = bot.Send(msg)
			return err
		}

		var (
			msgText = fmt.Sprintf(
				"Источник добавлен с ID: `%d`\\. Используйте этот ID для обновления источника или удаления\\.",
				sourceID,
			)
			reply = tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
		)

		reply.ParseMode = parseModeMarkdownV2

		if _, err := bot.Send(reply); err != nil {
			return err
		}

		return nil
	}
}
