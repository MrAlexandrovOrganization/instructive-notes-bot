package handlers

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/keyboards"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/state"
)

// StartHandler handles /start and /help commands.
type StartHandler struct {
	*Base
}

// NewStartHandler creates a new StartHandler.
func NewStartHandler(base *Base) *StartHandler {
	return &StartHandler{Base: base}
}

// HandleStart processes the /start command.
func (h *StartHandler) HandleStart(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) error {
	h.States.Reset(msg.From.ID)
	greeting := fmt.Sprintf("Привет, %s! 👋\n\nДобро пожаловать в систему заметок.", user.Name)

	reply := tgbotapi.NewMessage(msg.Chat.ID, greeting)
	reply.ReplyMarkup = keyboards.MainMenu(user.Role)
	_, err := h.Bot.Send(reply)
	return err
}

// HandleHelp processes the /help command.
func (h *StartHandler) HandleHelp(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) error {
	helpText := "📖 *Помощь*\n\n"
	helpText += "Используйте кнопки меню для навигации.\n\n"
	helpText += "*Доступные действия:*\n"
	helpText += "• 📝 Новая заметка — создать заметку о участнике\n"
	helpText += "• 📋 Заметки — просмотр ваших заметок\n"
	helpText += "• 👥 Участники — список участников\n"

	if user.Role == usersv1.Role_ROLE_ADMIN || user.Role == usersv1.Role_ROLE_ROOT {
		helpText += "• ⚙️ Управление — администрирование системы\n"
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, helpText)
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = keyboards.MainMenu(user.Role)
	_, err := h.Bot.Send(reply)
	return err
}

// HandleCancel resets the user state and returns to main menu.
func (h *StartHandler) HandleCancel(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) error {
	h.States.Reset(msg.From.ID)
	reply := tgbotapi.NewMessage(msg.Chat.ID, "Отменено.")
	reply.ReplyMarkup = keyboards.MainMenu(user.Role)
	_, err := h.Bot.Send(reply)
	return err
}

// HandleNoAccess sends the "no access" message to an unknown user.
func HandleNoAccess(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "⛔ У вас нет доступа к этому боту.\nОбратитесь к администратору.")
	_, _ = bot.Send(msg)
}

// HandleUnknownState handles messages that arrive in an unexpected state.
func (h *StartHandler) HandleUnknownState(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) error {
	h.States.Set(msg.From.ID, &state.UserContext{State: state.StateIdle})
	reply := tgbotapi.NewMessage(msg.Chat.ID, "Не понимаю. Используйте меню.")
	reply.ReplyMarkup = keyboards.MainMenu(user.Role)
	_, err := h.Bot.Send(reply)
	return err
}
