package middleware

import (
	"context"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestAdminsOnlyWrapsHandler(t *testing.T) {
	nextCalled := false
	view := AdminsOnly(1, func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
		nextCalled = true
		return nil
	})

	if view == nil {
		t.Fatal("expected middleware view")
	}
	if nextCalled {
		t.Fatal("next handler should not be called during setup")
	}
}
