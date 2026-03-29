package bot

import (
	"strings"
	"testing"

	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/model"
	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/notifier"
)

func TestFormatSource(t *testing.T) {
	got := formatSource(model.Source{
		ID:       12,
		Name:     "Steam News",
		FeedURL:  "https://example.com/feed.xml",
		Priority: 10,
	})

	if !strings.Contains(got, "Steam News") {
		t.Fatalf("expected source name in %q", got)
	}
	if !strings.Contains(got, "`12`") {
		t.Fatalf("expected source id in %q", got)
	}
}

func TestMarkdownParseModeConstant(t *testing.T) {
	if parseModeMarkdownV2 != "MarkdownV2" {
		t.Fatalf("unexpected parse mode: %q", parseModeMarkdownV2)
	}
}

func TestReactionParserIntegration(t *testing.T) {
	articleID, reaction, ok := notifier.ParseReactionCallback("reaction:like:5")
	if !ok || articleID != 5 || reaction != 1 {
		t.Fatalf("unexpected parse result: id=%d reaction=%d ok=%v", articleID, reaction, ok)
	}
}
