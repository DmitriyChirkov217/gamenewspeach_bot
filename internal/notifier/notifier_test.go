package notifier

import (
	"strings"
	"testing"

	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/model"
)

func TestParseReactionCallback(t *testing.T) {
	articleID, reaction, ok := ParseReactionCallback("reaction:like:42")
	if !ok || articleID != 42 || reaction != 1 {
		t.Fatalf("unexpected parse result: id=%d reaction=%d ok=%v", articleID, reaction, ok)
	}

	articleID, reaction, ok = ParseReactionCallback("reaction:dislike:42")
	if !ok || articleID != 42 || reaction != -1 {
		t.Fatalf("unexpected parse result: id=%d reaction=%d ok=%v", articleID, reaction, ok)
	}

	if _, _, ok := ParseReactionCallback("reaction:other:42"); ok {
		t.Fatal("expected invalid reaction to fail")
	}
}

func TestCleanupText(t *testing.T) {
	got := cleanupText("line1\n\n\n\nline2")
	if got != "line1\nline2" {
		t.Fatalf("unexpected cleaned text: %q", got)
	}
}

func TestFormatArticleUsesFallbackSummary(t *testing.T) {
	article := model.Article{
		Title: "Patch 1.2",
		Link:  "https://example.com/news?id=1",
	}

	got := formatArticle(article, "   ")
	if got == "" {
		t.Fatal("expected formatted article")
	}
	if !strings.Contains(got, "https://example\\.com/news?id\\=1") {
		t.Fatalf("expected escaped link in %q", got)
	}
}
