package summary

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/openaiapi"
)

type OpenAISummarizer struct {
	client  *openaiapi.Client
	prompt  string
	model   string
	enabled bool
	mu      sync.Mutex
}

func NewOpenAISummarizer(apiKey, model, prompt string) *OpenAISummarizer {
	s := &OpenAISummarizer{
		client: openaiapi.New(apiKey),
		prompt: prompt,
		model:  model,
	}

	log.Printf("openai summarizer is enabled: %v", apiKey != "")

	if apiKey != "" {
		s.enabled = true
	}

	return s
}

func (s *OpenAISummarizer) Summarize(text string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.enabled {
		return "", fmt.Errorf("openai summarizer is disabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	content, err := s.client.CreateChatCompletion(
		ctx,
		s.model,
		[]openaiapi.Message{
			{Role: "system", Content: s.prompt},
			{Role: "user", Content: text},
		},
		1024,
		1,
	)
	if err != nil {
		return "", err
	}

	rawSummary := strings.TrimSpace(content)
	if strings.HasSuffix(rawSummary, ".") {
		return rawSummary, nil
	}

	sentences := strings.Split(rawSummary, ".")
	if len(sentences) < 2 {
		return rawSummary, nil
	}

	return strings.Join(sentences[:len(sentences)-1], ".") + ".", nil
}
