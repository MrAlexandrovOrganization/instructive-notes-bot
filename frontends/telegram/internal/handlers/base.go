package handlers

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/client"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/state"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/whisper"
)

// Base holds shared dependencies for all handlers.
type Base struct {
	Bot     *tgbotapi.BotAPI
	Clients *client.Clients
	States  *state.Manager
	Whisper *whisper.Client // optional; nil if not configured
}

// NewBase creates a new Base with shared dependencies.
func NewBase(bot *tgbotapi.BotAPI, clients *client.Clients, states *state.Manager, wc *whisper.Client) *Base {
	return &Base{
		Bot:     bot,
		Clients: clients,
		States:  states,
		Whisper: wc,
	}
}

// EscapeMarkdown escapes all MarkdownV2 special characters in user-provided text.
func EscapeMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]",
		"(", "\\(", ")", "\\)", "~", "\\~", "`", "\\`",
		">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
		"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}",
		".", "\\.", "!", "\\!",
	)
	return replacer.Replace(s)
}

// Bold wraps escaped text in MarkdownV2 bold markers.
func Bold(s string) string {
	return "*" + EscapeMarkdown(s) + "*"
}

// SendError sends a plain-text error message.
func (b *Base) SendError(chatID int64, text string) error {
	_, err := b.Bot.Send(tgbotapi.NewMessage(chatID, "❌ "+text))
	return err
}

// AnswerCallback silently answers a callback query with an optional toast.
func (b *Base) AnswerCallback(callbackID, text string) {
	_, _ = b.Bot.Request(tgbotapi.NewCallback(callbackID, text))
}

// SendMD sends a MarkdownV2-formatted message with an optional reply markup.
func (b *Base) SendMD(chatID int64, text string, markup any) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "MarkdownV2"
	if markup != nil {
		msg.ReplyMarkup = markup
	}
	_, err := b.Bot.Send(msg)
	return err
}

// SendPlain sends a plain-text message with an optional reply markup.
func (b *Base) SendPlain(chatID int64, text string, markup any) error {
	msg := tgbotapi.NewMessage(chatID, text)
	if markup != nil {
		msg.ReplyMarkup = markup
	}
	_, err := b.Bot.Send(msg)
	return err
}

// EditMD edits an existing message with MarkdownV2 formatting and optional inline keyboard.
func (b *Base) EditMD(chatID int64, msgID int, text string, markup *tgbotapi.InlineKeyboardMarkup) error {
	edit := tgbotapi.NewEditMessageText(chatID, msgID, text)
	edit.ParseMode = "MarkdownV2"
	edit.ReplyMarkup = markup
	_, err := b.Bot.Send(edit)
	return err
}
