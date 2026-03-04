package handlers

import (
	"context"
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	groupsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/groups/v1"
	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/keyboards"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/state"
)

// AdminHandler handles administration interactions.
type AdminHandler struct {
	*Base
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(base *Base) *AdminHandler {
	return &AdminHandler{Base: base}
}

// HandleAdminPanel shows the admin panel.
func (h *AdminHandler) HandleAdminPanel(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) error {
	reply := tgbotapi.NewMessage(msg.Chat.ID, "⚙️ *Управление*")
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = keyboards.AdminPanel()
	_, err := h.Bot.Send(reply)
	return err
}

// HandleAdminCallback handles admin panel callbacks.
func (h *AdminHandler) HandleAdminCallback(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, action string) error {
	h.answerCallback(cb.ID, "")
	switch action {
	case "users":
		return h.showUsers(ctx, cb)
	case "groups":
		return h.showGroups(ctx, cb)
	case "add_user":
		return h.startAddUser(ctx, cb, user)
	case "add_group":
		return h.startAddGroup(ctx, cb, user)
	default:
		return nil
	}
}

func (h *AdminHandler) showUsers(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	resp, err := h.Clients.Users.ListUsers(ctx, &usersv1.ListUsersRequest{})
	if err != nil {
		return h.sendErrorCB(cb.Message.Chat.ID, "Не удалось загрузить пользователей.")
	}

	text := "👥 *Пользователи*\n\n"
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, u := range resp.Users {
		roleLabel := roleLabel(u.Role)
		text += fmt.Sprintf("• %s (@%s) — %s\n", u.Name, u.Username, roleLabel)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%s (%s)", u.Name, roleLabel),
				"user:manage:"+u.Id,
			),
		))
	}

	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, text)
	edit.ParseMode = "Markdown"
	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	edit.ReplyMarkup = &kb
	_, err = h.Bot.Send(edit)
	return err
}

func (h *AdminHandler) showGroups(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	resp, err := h.Clients.Groups.ListGroups(ctx, &groupsv1.ListGroupsRequest{})
	if err != nil {
		return h.sendErrorCB(cb.Message.Chat.ID, "Не удалось загрузить группы.")
	}

	text := "🏷 *Группы*\n\n"
	for _, g := range resp.Groups {
		text += fmt.Sprintf("• %s", g.Name)
		if g.Description != "" {
			text += fmt.Sprintf(" — %s", g.Description)
		}
		text += "\n"
	}

	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, text)
	edit.ParseMode = "Markdown"
	_, err = h.Bot.Send(edit)
	return err
}

func (h *AdminHandler) startAddUser(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User) error {
	h.States.SetState(cb.From.ID, state.StateAddingUserName)
	reply := tgbotapi.NewMessage(cb.Message.Chat.ID, "Введите имя нового пользователя:")
	reply.ReplyMarkup = keyboards.CancelKeyboard()
	_, err := h.Bot.Send(reply)
	return err
}

func (h *AdminHandler) startAddGroup(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User) error {
	h.States.SetState(cb.From.ID, state.StateAddingGroupName)
	reply := tgbotapi.NewMessage(cb.Message.Chat.ID, "Введите название новой группы:")
	reply.ReplyMarkup = keyboards.CancelKeyboard()
	_, err := h.Bot.Send(reply)
	return err
}

// HandleUserNameInput handles the user name input during user creation.
func (h *AdminHandler) HandleUserNameInput(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) error {
	h.States.Set(msg.From.ID, &state.UserContext{
		State:       state.StateAddingUserTelegramID,
		PendingData: msg.Text,
	})
	reply := tgbotapi.NewMessage(msg.Chat.ID, "Введите Telegram ID нового пользователя (число):")
	reply.ReplyMarkup = keyboards.CancelKeyboard()
	_, err := h.Bot.Send(reply)
	return err
}

// HandleUserTelegramIDInput handles the Telegram ID input during user creation.
func (h *AdminHandler) HandleUserTelegramIDInput(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) error {
	telegramID, err := strconv.ParseInt(msg.Text, 10, 64)
	if err != nil {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Неверный формат ID. Введите число:")
		_, err = h.Bot.Send(reply)
		return err
	}

	userCtx := h.States.Get(msg.From.ID)
	name := userCtx.PendingData
	h.States.Reset(msg.From.ID)

	// Create user with organizer role.
	newUser, _, err2 := h.createUserDirectly(ctx, telegramID, name)
	if err2 != nil {
		return h.sendError(msg.Chat.ID, "Не удалось создать пользователя.")
	}

	text := fmt.Sprintf("✅ Пользователь *%s* (ID: %d) добавлен как Организатор.",
		escapeMarkdown(newUser.Name), newUser.TelegramId)

	// Show role selection buttons.
	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Организатор", "user:role:"+newUser.Id+":organizer"),
			tgbotapi.NewInlineKeyboardButtonData("Куратор", "user:role:"+newUser.Id+":curator"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Администратор", "user:role:"+newUser.Id+":admin"),
		),
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "MarkdownV2"
	reply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	_, err = h.Bot.Send(reply)
	return err
}

func (h *AdminHandler) createUserDirectly(ctx context.Context, telegramID int64, name string) (*usersv1.User, bool, error) {
	resp, err := h.Clients.Users.GetOrCreateUser(ctx, &usersv1.GetOrCreateUserRequest{
		TelegramId: telegramID,
		Name:       name,
	})
	if err != nil {
		return nil, false, err
	}
	return resp.User, resp.Created, nil
}

// HandleGroupNameInput handles the group name input.
func (h *AdminHandler) HandleGroupNameInput(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) error {
	h.States.Reset(msg.From.ID)

	g, err := h.Clients.Groups.CreateGroup(ctx, &groupsv1.CreateGroupRequest{
		Name: msg.Text,
	})
	if err != nil {
		return h.sendError(msg.Chat.ID, "Не удалось создать группу.")
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ Группа *%s* создана!", escapeMarkdown(g.Name)))
	reply.ParseMode = "MarkdownV2"
	reply.ReplyMarkup = keyboards.MainMenu(user.Role)
	_, err = h.Bot.Send(reply)
	return err
}

// HandleUserRoleUpdate handles role change callback.
func (h *AdminHandler) HandleUserRoleUpdate(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, userID, roleStr string) error {
	roleMap := map[string]usersv1.Role{
		"organizer": usersv1.Role_ROLE_ORGANIZER,
		"curator":   usersv1.Role_ROLE_CURATOR,
		"admin":     usersv1.Role_ROLE_ADMIN,
		"root":      usersv1.Role_ROLE_ROOT,
	}
	role, ok := roleMap[roleStr]
	if !ok {
		h.answerCallback(cb.ID, "Неизвестная роль")
		return nil
	}

	_, err := h.Clients.Users.UpdateUserRole(ctx, &usersv1.UpdateUserRoleRequest{
		Id:   userID,
		Role: role,
	})
	if err != nil {
		h.answerCallback(cb.ID, "Ошибка")
		return err
	}
	h.answerCallback(cb.ID, "✅ Роль обновлена")
	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID,
		fmt.Sprintf("✅ Роль пользователя обновлена на *%s*", escapeMarkdown(roleStr)))
	edit.ParseMode = "MarkdownV2"
	_, err = h.Bot.Send(edit)
	return err
}

func (h *AdminHandler) sendError(chatID int64, text string) error {
	_, err := h.Bot.Send(tgbotapi.NewMessage(chatID, "❌ "+text))
	return err
}

func (h *AdminHandler) sendErrorCB(chatID int64, text string) error {
	_, err := h.Bot.Send(tgbotapi.NewMessage(chatID, "❌ "+text))
	return err
}

func (h *AdminHandler) answerCallback(callbackID, text string) {
	_, _ = h.Bot.Request(tgbotapi.NewCallback(callbackID, text))
}

func roleLabel(role usersv1.Role) string {
	switch role {
	case usersv1.Role_ROLE_ROOT:
		return "Root"
	case usersv1.Role_ROLE_ADMIN:
		return "Admin"
	case usersv1.Role_ROLE_CURATOR:
		return "Curator"
	default:
		return "Organizer"
	}
}
