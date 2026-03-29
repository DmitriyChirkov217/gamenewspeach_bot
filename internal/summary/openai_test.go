package summary

import "testing"

func TestNewOpenAISummarizerDisabledWithoutAPIKey(t *testing.T) {
	s := NewOpenAISummarizer("", "gpt-test", "prompt")
	if s.enabled {
		t.Fatal("expected summarizer to be disabled without api key")
	}
}
