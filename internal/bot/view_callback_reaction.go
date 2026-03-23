package bot

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/botkit"
	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/notifier"
)

type ReactionStorage interface {
	SaveReaction(ctx context.Context, userID int64, articleID int64, reaction int) error
}

func ViewCallbackReaction(storage ReactionStorage) botkit.ViewFunc {
	return func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
		callback := update.CallbackQuery
		articleID, reaction, ok := notifier.ParseReactionCallback(callback.Data)
		if !ok {
			_, _ = bot.Request(tgbotapi.NewCallback(callback.ID, "Не удалось распознать реакцию"))
			return nil
		}

		if err := storage.SaveReaction(ctx, callback.From.ID, articleID, reaction); err != nil {
			return err
		}

		text := "Учту это в следующих рекомендациях"
		if reaction < 0 {
			text = "Понял, таких новостей будет меньше"
		}

		_, err := bot.Request(tgbotapi.NewCallback(callback.ID, text))
		return err
	}
}
