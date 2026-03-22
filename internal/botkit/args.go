package botkit

import (
	"encoding/json"
)

// ParseJSON обобщенно декодирует JSON-аргументы команды Telegram и используется
// в bot.ViewCmdAddSource и bot.ViewCmdSetPriority для разбора update.Message.CommandArguments().
func ParseJSON[T any](src string) (T, error) {
	var args T

	if err := json.Unmarshal([]byte(src), &args); err != nil {
		return *(new(T)), err
	}

	return args, nil
}
