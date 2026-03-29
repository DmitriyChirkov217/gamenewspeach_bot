package tagger

import (
	"errors"
	"testing"
)

func TestNormalizeTag(t *testing.T) {
	if got := normalizeTag("  RPG Game  "); got != "rpg-game" {
		t.Fatalf("unexpected normalized tag: %q", got)
	}
	if got := normalizeTag("x"); got != "" {
		t.Fatalf("expected short tag to be dropped, got %q", got)
	}
}

func TestExtractKeywordTags(t *testing.T) {
	got := extractKeywordTags("New PS5 shooter update hits Game Pass today")
	for _, tag := range []string{"playstation", "shooter", "update", "subscription"} {
		if _, ok := got[tag]; !ok {
			t.Fatalf("expected tag %q in %+v", tag, got)
		}
	}
}

func TestShouldDisableAI(t *testing.T) {
	tagger := &Tagger{enabled: true}
	err := errors.New("status code: 429 exceeded your current quota")

	if !tagger.shouldDisableAI(err) {
		t.Fatal("expected AI to be disabled")
	}
	if tagger.enabled {
		t.Fatal("expected enabled flag to be turned off")
	}
}

func TestMergeTagWeights(t *testing.T) {
	got := mergeTagWeights(
		map[string]float64{"rpg": 0.6, "indie": 0.5},
		map[string]float64{"rpg": 0.9},
	)

	if got["rpg"] != 0.9 || got["indie"] != 0.5 {
		t.Fatalf("unexpected merged tags: %+v", got)
	}
}

func TestShouldDisableAIReturnsFalseForOtherErrors(t *testing.T) {
	tagger := &Tagger{enabled: true}
	if tagger.shouldDisableAI(errors.New("boom")) {
		t.Fatal("expected ordinary errors not to disable AI")
	}
	if !tagger.enabled {
		t.Fatal("expected enabled to stay true")
	}
}
