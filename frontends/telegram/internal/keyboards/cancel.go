package keyboards

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// CancelInline returns an inline keyboard with a single Cancel button.
func CancelInline() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel"),
		),
	)
}
