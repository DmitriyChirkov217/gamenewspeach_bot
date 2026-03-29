package bot

import (
	"context"
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/botkit"
	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/botkit/markup"
	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/model"
)

type SourceProvider interface {
	SourceByID(ctx context.Context, id int64) (*model.Source, error)
}

// ViewCmdGetSource создает обработчик команды /getsource: он получает ID источника,
// загружает запись через SourceProvider.SourceByID и форматирует ответ через formatSource.
func ViewCmdGetSource(provider SourceProvider) botkit.ViewFunc {
	return func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
		idStr := update.Message.CommandArguments()

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return err
		}

		source, err := provider.SourceByID(ctx, id)
		if err != nil {
			return err
		}

		reply := tgbotapi.NewMessage(update.Message.Chat.ID, formatSource(*source))
		reply.ParseMode = parseModeMarkdownV2

		if _, err := bot.Send(reply); err != nil {
			return err
		}

		return nil
	}
}

// formatSource собирает человекочитаемое описание источника и экранирует поля через markup.EscapeForMarkdown;
// эту функцию используют ViewCmdGetSource и ViewCmdListSource при выводе информации пользователю.
func formatSource(source model.Source) string {
	return fmt.Sprintf(
		"🌐 *%s*\nID: `%d`\nURL фида: %s\nПриоритет: %d",
		markup.EscapeForMarkdown(source.Name),
		source.ID,
		markup.EscapeForMarkdown(source.FeedURL),
		source.Priority,
	)
}
