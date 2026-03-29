package tagger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/model"
	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/openaiapi"
)

type Tagger struct {
	client  *openaiapi.Client
	model   string
	enabled bool
	mu      sync.Mutex
}

func New(apiKey, model string) *Tagger {
	t := &Tagger{
		model: model,
	}

	if apiKey != "" {
		t.client = openaiapi.New(apiKey)
		t.enabled = true
	}

	return t
}

func (t *Tagger) Tags(ctx context.Context, item model.Item) ([]model.ArticleTag, error) {
	tags := mergeTagWeights(
		extractCategoryTags(item.Categories),
		extractKeywordTags(item.Title+" "+item.Summary),
	)

	if !t.enabled {
		return toArticleTags(tags), nil
	}

	aiTags, err := t.aiTags(ctx, item)
	if err != nil {
		if t.shouldDisableAI(err) {
			log.Printf("[WARN] disabling AI tagging after OpenAI quota/rate-limit error: %v", err)
		} else {
			log.Printf("[WARN] failed to extract AI tags for %q: %v", item.Title, err)
		}
		return toArticleTags(tags), nil
	}

	return toArticleTags(mergeTagWeights(tags, aiTags)), nil
}

func (t *Tagger) aiTags(ctx context.Context, item model.Item) (map[string]float64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	systemPrompt := `You classify game news articles into concise lowercase tags.
Return only valid JSON in the form {"tags":[{"tag":"tag-name","weight":0.0}]}.
Rules:
- Use 3 to 8 tags.
- Tags must be short, lowercase, and use hyphens instead of spaces.
- Prefer tags about platform, genre, topic, mode, business model, and event type.
- Example tags: pc, playstation, xbox, nintendo, mobile, rpg, shooter, strategy, indie, esports, update, release, dlc, patch, review, hardware, vr, free-to-play, multiplayer.
- Weights must be between 0.3 and 1.0.`

	userPrompt := fmt.Sprintf(
		"Title: %s\nCategories: %s\nSummary: %s",
		item.Title,
		strings.Join(item.Categories, ", "),
		item.Summary,
	)

	limitedCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	content, err := t.client.CreateChatCompletion(
		limitedCtx,
		t.model,
		[]openaiapi.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		300,
		0.2,
	)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Tags []struct {
			Tag    string  `json:"tag"`
			Weight float64 `json:"weight"`
		} `json:"tags"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil, err
	}

	result := make(map[string]float64, len(payload.Tags))
	for _, tag := range payload.Tags {
		normalized := normalizeTag(tag.Tag)
		if normalized == "" {
			continue
		}

		weight := tag.Weight
		if weight < 0.3 {
			weight = 0.3
		}
		if weight > 1 {
			weight = 1
		}

		if current, ok := result[normalized]; !ok || weight > current {
			result[normalized] = weight
		}
	}

	return result, nil
}

func (t *Tagger) shouldDisableAI(err error) bool {
	var apiErr *openaiapi.APIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == 429 || strings.Contains(strings.ToLower(apiErr.Message), "exceeded your current quota") {
			t.enabled = false
			return true
		}
	}

	if strings.Contains(err.Error(), "status code: 429") ||
		strings.Contains(strings.ToLower(err.Error()), "exceeded your current quota") {
		t.enabled = false
		return true
	}

	return false
}

func toArticleTags(tags map[string]float64) []model.ArticleTag {
	names := make([]string, 0, len(tags))
	for tag := range tags {
		names = append(names, tag)
	}
	sort.Strings(names)

	result := make([]model.ArticleTag, 0, len(names))
	for _, tag := range names {
		result = append(result, model.ArticleTag{
			Tag:    tag,
			Weight: tags[tag],
		})
	}

	return result
}

func mergeTagWeights(groups ...map[string]float64) map[string]float64 {
	result := make(map[string]float64)
	for _, group := range groups {
		for tag, weight := range group {
			if current, ok := result[tag]; !ok || weight > current {
				result[tag] = weight
			}
		}
	}
	return result
}

func extractCategoryTags(categories []string) map[string]float64 {
	result := make(map[string]float64)
	for _, category := range categories {
		tag := normalizeTag(category)
		if tag == "" {
			continue
		}
		result[tag] = 0.7
	}
	return result
}

var tagRules = map[string][]string{
	"pc":           {"pc", "steam", "windows"},
	"playstation":  {"playstation", "ps4", "ps5", "sony"},
	"xbox":         {"xbox", "game pass", "microsoft"},
	"nintendo":     {"nintendo", "switch", "joy-con"},
	"mobile":       {"android", "ios", "mobile"},
	"vr":           {"vr", "virtual reality", "meta quest"},
	"rpg":          {"rpg", "role-playing", "jrpg"},
	"shooter":      {"shooter", "fps", "third-person shooter"},
	"strategy":     {"strategy", "tactics", "4x"},
	"simulation":   {"simulation", "simulator"},
	"sports":       {"sports", "football", "soccer", "basketball", "racing"},
	"horror":       {"horror"},
	"survival":     {"survival"},
	"indie":        {"indie"},
	"multiplayer":  {"multiplayer", "co-op", "coop", "online"},
	"singleplayer": {"single-player", "single player", "singleplayer"},
	"free-to-play": {"free-to-play", "f2p"},
	"update":       {"update", "updated", "hotfix", "patch notes"},
	"patch":        {"patch"},
	"release":      {"release", "launch", "available now", "out now"},
	"dlc":          {"dlc", "expansion"},
	"announcement": {"announce", "announcement", "revealed", "reveal"},
	"trailer":      {"trailer", "teaser"},
	"review":       {"review", "hands-on", "impressions"},
	"rumor":        {"rumor", "leak", "reportedly"},
	"hardware":     {"gpu", "cpu", "console", "hardware"},
	"esports":      {"esports", "tournament", "championship"},
	"subscription": {"subscription", "game pass", "ps plus"},
}

func extractKeywordTags(text string) map[string]float64 {
	result := make(map[string]float64)
	lower := strings.ToLower(text)

	for tag, variants := range tagRules {
		for _, variant := range variants {
			if strings.Contains(lower, variant) {
				result[tag] = 0.9
				break
			}
		}
	}

	return result
}

var nonTagChars = regexp.MustCompile(`[^a-z0-9-]+`)

func normalizeTag(tag string) string {
	tag = strings.ToLower(strings.TrimSpace(tag))
	tag = strings.ReplaceAll(tag, "_", "-")
	tag = strings.ReplaceAll(tag, " ", "-")
	tag = nonTagChars.ReplaceAllString(tag, "")
	tag = strings.Trim(tag, "-")

	if len(tag) < 2 {
		return ""
	}

	return tag
}
