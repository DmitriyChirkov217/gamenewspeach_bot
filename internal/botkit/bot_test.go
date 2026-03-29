package botkit

import (
	"context"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestRegisterViews(t *testing.T) {
	bot := New(nil)
	cmdView := func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error { return nil }
	callbackView := func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error { return nil }

	bot.RegisterCmdView("start", cmdView)
	bot.RegisterCallbackView("reaction:", callbackView)

	if got := bot.cmdViews["start"]; got == nil {
		t.Fatal("expected command view to be registered")
	}
	if got := bot.callbackViews["reaction:"]; got == nil {
		t.Fatal("expected callback view to be registered")
	}
}
