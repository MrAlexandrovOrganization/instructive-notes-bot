package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/keyboards"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/state"
	groupsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/groups/v1"
	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
)

// AdminHandler handles administration interactions.
type AdminHandler struct {
	*Base
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(base *Base) *AdminHandler {
	return &AdminHandler{Base: base}
}

// HandleAdminPanel shows the admin panel by editing the current message.
func (h *AdminHandler) HandleAdminPanel(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User) error {
	kb := keyboards.AdminPanel()
	return h.EditMD(cb.Message.Chat.ID, cb.Message.MessageID, "⚙️ *Управление*", &kb)
}

// HandleAdminCallback handles admin panel callbacks.
func (h *AdminHandler) HandleAdminCallback(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, action string) error {
	h.AnswerCallback(cb.ID, "")
	switch action {
	case "users":
		return h.showUsers(ctx, cb)
	case "groups":
		return h.showGroups(ctx, cb)
	case "add_user":
		return h.startAddUser(ctx, cb)
	case "add_group":
		return h.startAddGroup(ctx, cb)
	default:
		return nil
	}
}

func (h *AdminHandler) showUsers(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	resp, err := h.Clients.Users.ListUsers(ctx, &usersv1.ListUsersRequest{})
	if err != nil {
		return h.SendError(cb.Message.Chat.ID, "Не удалось загрузить пользователей.")
	}

	var sb strings.Builder
	sb.WriteString("👥 *Пользователи*\n\n")
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, u := range resp.Users {
		label := roleLabel(u.Role)
		fmt.Fprintf(&sb, "• %s \\(@%s\\) — %s\n", EscapeMarkdown(u.Name), EscapeMarkdown(u.Username), EscapeMarkdown(label))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%s (%s)", u.Name, label),
				"user:manage:"+u.Id,
			),
		))
	}
	text := sb.String()
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("➕ Добавить пользователя", "admin:add_user"),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "back:admin"),
	))

	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	return h.EditMD(cb.Message.Chat.ID, cb.Message.MessageID, text, &kb)
}

func (h *AdminHandler) showGroups(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	resp, err := h.Clients.Groups.ListGroups(ctx, &groupsv1.ListGroupsRequest{})
	if err != nil {
		return h.SendError(cb.Message.Chat.ID, "Не удалось загрузить отряды.")
	}

	kb := keyboards.GroupsListForBrowse(resp.Groups)
	return h.EditMD(cb.Message.Chat.ID, cb.Message.MessageID, "🏷 *Отряды*", &kb)
}

func (h *AdminHandler) startAddUser(_ context.Context, cb *tgbotapi.CallbackQuery) error {
	h.States.SetState(cb.From.ID, state.StateAddingUserName)
	return h.SendPlain(cb.Message.Chat.ID, "Введите имя нового пользователя:", keyboards.CancelInline())
}

func (h *AdminHandler) startAddGroup(_ context.Context, cb *tgbotapi.CallbackQuery) error {
	h.States.SetState(cb.From.ID, state.StateAddingGroupName)
	return h.SendPlain(cb.Message.Chat.ID, "Введите название новой отряды:", keyboards.CancelInline())
}

// HandleUserNameInput handles the user name input during user creation.
func (h *AdminHandler) HandleUserNameInput(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) error {
	h.States.Set(msg.From.ID, &state.UserContext{
		State:       state.StateAddingUserTelegramID,
		PendingData: msg.Text,
	})
	return h.SendPlain(msg.Chat.ID, "Введите Telegram ID нового пользователя (число):", keyboards.CancelInline())
}

// HandleUserTelegramIDInput handles the Telegram ID input during user creation.
func (h *AdminHandler) HandleUserTelegramIDInput(_ context.Context, msg *tgbotapi.Message, _ *usersv1.User) error {
	telegramID, err := strconv.ParseInt(msg.Text, 10, 64)
	if err != nil {
		return h.SendPlain(msg.Chat.ID, "Неверный формат ID. Введите число:", keyboards.CancelInline())
	}

	userCtx := h.States.Get(msg.From.ID)
	h.States.Set(msg.From.ID, &state.UserContext{
		State:        state.StateAddingUserRole,
		PendingData:  userCtx.PendingData,
		PendingData2: strconv.FormatInt(telegramID, 10),
	})

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Организатор", "user:create_role:organizer"),
			tgbotapi.NewInlineKeyboardButtonData("Куратор", "user:create_role:curator"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Администратор", "user:create_role:admin"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel"),
		),
	)
	return h.SendPlain(msg.Chat.ID, "Выберите роль для нового пользователя:", kb)
}

// HandleUserCreateRole creates a new user with the chosen role.
func (h *AdminHandler) HandleUserCreateRole(ctx context.Context, cb *tgbotapi.CallbackQuery, _ *usersv1.User, roleStr string) error {
	roleMap := map[string]usersv1.Role{
		"organizer": usersv1.Role_ROLE_ORGANIZER,
		"curator":   usersv1.Role_ROLE_CURATOR,
		"admin":     usersv1.Role_ROLE_ADMIN,
	}
	role, ok := roleMap[roleStr]
	if !ok {
		h.AnswerCallback(cb.ID, "Неизвестная роль")
		return nil
	}

	userCtx := h.States.Get(cb.From.ID)
	name := userCtx.PendingData
	telegramID, err := strconv.ParseInt(userCtx.PendingData2, 10, 64)
	if err != nil {
		h.States.Reset(cb.From.ID)
		h.AnswerCallback(cb.ID, "Ошибка")
		return h.EditMD(cb.Message.Chat.ID, cb.Message.MessageID, "❌ Ошибка: неверный Telegram ID\\.", nil)
	}
	h.States.Reset(cb.From.ID)

	newUser, _, err := h.createUserDirectly(ctx, telegramID, name)
	if err != nil {
		h.AnswerCallback(cb.ID, "Ошибка")
		return h.EditMD(cb.Message.Chat.ID, cb.Message.MessageID, "❌ Не удалось создать пользователя\\.", nil)
	}

	if role != usersv1.Role_ROLE_ORGANIZER {
		_, err = h.Clients.Users.UpdateUserRole(ctx, &usersv1.UpdateUserRoleRequest{
			Id:   newUser.Id,
			Role: role,
		})
		if err != nil {
			h.AnswerCallback(cb.ID, "Ошибка роли")
			return h.EditMD(cb.Message.Chat.ID, cb.Message.MessageID,
				fmt.Sprintf("⚠️ Пользователь *%s* создан, но не удалось установить роль\\.", EscapeMarkdown(newUser.Name)), nil)
		}
	}

	h.AnswerCallback(cb.ID, "✅ Готово")
	text := fmt.Sprintf("✅ Пользователь *%s* \\(ID: %d\\) создан как *%s*\\.",
		EscapeMarkdown(newUser.Name), newUser.TelegramId, EscapeMarkdown(roleLabel(role)))
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👥 Пользователи", "admin:users"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Управление", "back:admin"),
		),
	)
	return h.EditMD(cb.Message.Chat.ID, cb.Message.MessageID, text, &kb)
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
func (h *AdminHandler) HandleGroupNameInput(ctx context.Context, msg *tgbotapi.Message, _ *usersv1.User) error {
	h.States.Reset(msg.From.ID)

	g, err := h.Clients.Groups.CreateGroup(ctx, &groupsv1.CreateGroupRequest{
		Name: msg.Text,
	})
	if err != nil {
		return h.SendError(msg.Chat.ID, "Не удалось создать отряд.")
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏷 Отряды", "admin:groups"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Управление", "back:admin"),
		),
	)
	text := fmt.Sprintf("✅ Отряд *%s* создан\\!", EscapeMarkdown(g.Name))
	return h.SendMD(msg.Chat.ID, text, kb)
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
		h.AnswerCallback(cb.ID, "Неизвестная роль")
		return nil
	}

	_, err := h.Clients.Users.UpdateUserRole(ctx, &usersv1.UpdateUserRoleRequest{
		Id:   userID,
		Role: role,
	})
	if err != nil {
		h.AnswerCallback(cb.ID, "Ошибка")
		return err
	}
	h.AnswerCallback(cb.ID, "✅ Роль обновлена")
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👥 Пользователи", "admin:users"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Управление", "back:admin"),
		),
	)
	return h.EditMD(cb.Message.Chat.ID, cb.Message.MessageID,
		fmt.Sprintf("✅ Роль пользователя обновлена на *%s*\\.", EscapeMarkdown(roleLabel(role))), &kb)
}

var roleLabelMap = map[usersv1.Role]string{
	usersv1.Role_ROLE_ROOT:      "Root",
	usersv1.Role_ROLE_ADMIN:     "Администратор",
	usersv1.Role_ROLE_CURATOR:   "Куратор",
	usersv1.Role_ROLE_ORGANIZER: "Организатор",
}

func roleLabel(role usersv1.Role) string {
	if label, ok := roleLabelMap[role]; ok {
		return label
	}
	return "Организатор"
}
