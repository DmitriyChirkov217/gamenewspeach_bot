package notifier

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-shiori/go-readability"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/botkit/markup"
	"github.com/DmitriyChirkov217/gamenewspeach_bot/internal/model"
)

const (
	reactionLike    = "like"
	reactionDislike = "dislike"
)

type ArticleProvider interface {
	RecommendForUser(ctx context.Context, userID int64, since time.Time, limit uint64) ([]model.Article, error)
	RecordDelivery(ctx context.Context, userID int64, articleID int64, messageID int) error
	SaveReaction(ctx context.Context, userID int64, articleID int64, reaction int) error
}

type UserProvider interface {
	Subscribers(ctx context.Context) ([]model.User, error)
}

type Summarizer interface {
	Summarize(text string) (string, error)
}

type Notifier struct {
	articles         ArticleProvider
	users            UserProvider
	summarizer       Summarizer
	bot              *tgbotapi.BotAPI
	sendInterval     time.Duration
	lookupTimeWindow time.Duration
}

func New(
	articleProvider ArticleProvider,
	userProvider UserProvider,
	summarizer Summarizer,
	bot *tgbotapi.BotAPI,
	sendInterval time.Duration,
	lookupTimeWindow time.Duration,
) *Notifier {
	return &Notifier{
		articles:         articleProvider,
		users:            userProvider,
		summarizer:       summarizer,
		bot:              bot,
		sendInterval:     sendInterval,
		lookupTimeWindow: lookupTimeWindow,
	}
}

func (n *Notifier) Start(ctx context.Context) error {
	ticker := time.NewTicker(n.sendInterval)
	defer ticker.Stop()

	if err := n.SendPersonalizedArticles(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
			if err := n.SendPersonalizedArticles(ctx); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (n *Notifier) SendPersonalizedArticles(ctx context.Context) error {
	users, err := n.users.Subscribers(ctx)
	if err != nil {
		return err
	}

	for _, user := range users {
		if err := n.sendNextArticleToUser(ctx, user); err != nil {
			log.Printf("[ERROR] failed to send article to user %d: %v", user.TelegramUserID, err)
		}
	}

	return nil
}

func (n *Notifier) sendNextArticleToUser(ctx context.Context, user model.User) error {
	candidates, err := n.articles.RecommendForUser(ctx, user.TelegramUserID, time.Now().Add(-n.lookupTimeWindow), 1)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		return nil
	}

	article := candidates[0]
	summary, err := n.extractSummary(article)
	if err != nil {
		log.Printf("[WARN] failed to extract summary for article %d: %v", article.ID, err)
	}

	msg := tgbotapi.NewMessage(user.ChatID, formatArticle(article, summary))
	msg.ParseMode = "MarkdownV2"
	msg.ReplyMarkup = reactionKeyboard(article.ID)

	sentMessage, err := n.bot.Send(msg)
	if err != nil {
		return err
	}

	return n.articles.RecordDelivery(ctx, user.TelegramUserID, article.ID, sentMessage.MessageID)
}

func reactionKeyboard(articleID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👍", reactionCallbackData(reactionLike, articleID)),
			tgbotapi.NewInlineKeyboardButtonData("👎", reactionCallbackData(reactionDislike, articleID)),
		),
	)
}

func reactionCallbackData(reaction string, articleID int64) string {
	return "reaction:" + reaction + ":" + strconv.FormatInt(articleID, 10)
}

func ParseReactionCallback(data string) (articleID int64, reaction int, ok bool) {
	parts := strings.Split(data, ":")
	if len(parts) != 3 || parts[0] != "reaction" {
		return 0, 0, false
	}

	switch parts[1] {
	case reactionLike:
		reaction = 1
	case reactionDislike:
		reaction = -1
	default:
		return 0, 0, false
	}

	articleID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return 0, 0, false
	}

	return articleID, reaction, true
}

var redundantNewLines = regexp.MustCompile(`\n{3,}`)

func (n *Notifier) extractSummary(article model.Article) (string, error) {
	var r io.Reader

	if article.Summary != "" {
		r = strings.NewReader(article.Summary)
	} else {
		resp, err := http.Get(article.Link)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		r = resp.Body
	}

	doc, err := readability.FromReader(r, nil)
	if err != nil {
		return "", err
	}

	summary, err := n.summarizer.Summarize(cleanupText(doc.TextContent))
	if err != nil {
		return "", err
	}

	return summary, nil
}

func cleanupText(text string) string {
	return redundantNewLines.ReplaceAllString(text, "\n")
}

func formatArticle(article model.Article, summary string) string {
	const msgFormat = "*%s*\n\n%s\n\n%s"

	safeSummary := "_Краткое описание пока недоступно_"
	if strings.TrimSpace(summary) != "" {
		safeSummary = markup.EscapeForMarkdown(summary)
	}

	return fmt.Sprintf(
		msgFormat,
		markup.EscapeForMarkdown(article.Title),
		safeSummary,
		markup.EscapeForMarkdown(article.Link),
	)
}
