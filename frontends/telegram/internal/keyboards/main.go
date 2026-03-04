package keyboards

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
)

// MainMenu returns the role-aware main menu keyboard.
func MainMenu(role usersv1.Role) tgbotapi.ReplyKeyboardMarkup {
	switch role {
	case usersv1.Role_ROLE_ADMIN, usersv1.Role_ROLE_ROOT:
		return tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("👥 Участники"),
				tgbotapi.NewKeyboardButton("📝 Новая заметка"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("📋 Все заметки"),
				tgbotapi.NewKeyboardButton("⚙️ Управление"),
			),
		)
	case usersv1.Role_ROLE_CURATOR:
		return tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("👥 Моя группа"),
				tgbotapi.NewKeyboardButton("📝 Новая заметка"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("📋 Мои заметки"),
				tgbotapi.NewKeyboardButton("🔍 Все участники"),
			),
		)
	default: // ORGANIZER
		return tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("📝 Новая заметка"),
				tgbotapi.NewKeyboardButton("👥 Участники"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("📋 Мои заметки"),
			),
		)
	}
}

// AdminPanel returns the admin panel inline keyboard.
func AdminPanel() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👥 Пользователи", "admin:users"),
			tgbotapi.NewInlineKeyboardButtonData("🏷 Группы", "admin:groups"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Все заметки", "admin:all_notes"),
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить пользователя", "admin:add_user"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить группу", "admin:add_group"),
		),
	)
}

// CancelKeyboard returns a simple cancel keyboard.
func CancelKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("❌ Отмена"),
		),
	)
}
