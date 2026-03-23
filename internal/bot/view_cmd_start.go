package bot

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/botkit"
	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/model"
)

type UserStorage interface {
	Upsert(ctx context.Context, user model.User) error
}

func ViewCmdStart(storage UserStorage) botkit.ViewFunc {
	return func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
		user := update.SentFrom()
		chat := update.FromChat()

		if err := storage.Upsert(ctx, model.User{
			TelegramUserID: user.ID,
			ChatID:         chat.ID,
			Username:       user.UserName,
			FirstName:      user.FirstName,
		}); err != nil {
			return err
		}

		msg := tgbotapi.NewMessage(
			chat.ID,
			"Подписка включена. Теперь я буду присылать новости в личные сообщения и учитывать ваши реакции 👍/👎 для рекомендаций.",
		)

		_, err := bot.Send(msg)
		return err
	}
}
