package markup

import "strings"

var (
	replacer = strings.NewReplacer(
		"-",
		"\\-",
		"_",
		"\\_",
		"*",
		"\\*",
		"[",
		"\\[",
		"]",
		"\\]",
		"(",
		"\\(",
		")",
		"\\)",
		"~",
		"\\~",
		"`",
		"\\`",
		">",
		"\\>",
		"#",
		"\\#",
		"+",
		"\\+",
		"=",
		"\\=",
		"|",
		"\\|",
		"{",
		"\\{",
		"}",
		"\\}",
		".",
		"\\.",
		"!",
		"\\!",
	)
)

// EscapeForMarkdown экранирует спецсимволы MarkdownV2 и используется при формировании сообщений
// в bot.formatSource и notifier.sendArticle, чтобы Telegram корректно отобразил текст.
func EscapeForMarkdown(src string) string {
	return replacer.Replace(src)
}
