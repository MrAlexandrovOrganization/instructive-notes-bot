package keyboards

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
)

// MainMenu returns the role-aware main menu as an inline keyboard.
func MainMenu(role usersv1.Role) tgbotapi.InlineKeyboardMarkup {
	switch role {
	case usersv1.Role_ROLE_ADMIN, usersv1.Role_ROLE_ROOT:
		return tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("👥 Участники", "menu:participants"),
				tgbotapi.NewInlineKeyboardButtonData("🏷 Отряды", "menu:groups"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📋 Все заметки", "menu:all_notes"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⚙️ Управление", "menu:admin"),
			),
		)
	case usersv1.Role_ROLE_CURATOR:
		return tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("👥 Мой отряд", "menu:my_group"),
				tgbotapi.NewInlineKeyboardButtonData("🔍 Все участники", "menu:participants"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📋 Мои заметки", "menu:notes"),
			),
		)
	default: // ORGANIZER
		return tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("👥 Участники", "menu:participants"),
				tgbotapi.NewInlineKeyboardButtonData("📋 Мои заметки", "menu:notes"),
			),
		)
	}
}

// AdminPanel returns the admin panel inline keyboard.
func AdminPanel() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👥 Пользователи", "admin:users"),
			tgbotapi.NewInlineKeyboardButtonData("🏷 Отряды", "admin:groups"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "back:menu"),
		),
	)
}
