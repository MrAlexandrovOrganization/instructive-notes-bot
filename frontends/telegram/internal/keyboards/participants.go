package keyboards

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	participantsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/participants/v1"
	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
)

// ParticipantsList returns an inline keyboard for listing participants.
func ParticipantsList(participants []*participantsv1.Participant, nextCursor string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, p := range participants {
		label := p.Name
		if p.NotesCount > 0 {
			label = p.Name + " 📝"
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, "participant:view:"+p.Id),
		))
	}
	if nextCursor != "" {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➡️ Далее", "page:participants:"+nextCursor),
		))
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// ParticipantView returns the action keyboard for a single participant.
func ParticipantView(participantID string, role usersv1.Role) tgbotapi.InlineKeyboardMarkup {
	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Заметки", "notes:participant:"+participantID),
			tgbotapi.NewInlineKeyboardButtonData("📝 Написать", "participant:note:"+participantID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🖼 Фото", "participant:photo:"+participantID),
		),
	}
	if role == usersv1.Role_ROLE_ADMIN || role == usersv1.Role_ROLE_ROOT {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Редактировать", "participant:edit:"+participantID),
			tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить", "participant:delete:"+participantID),
		))
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// SelectParticipantForNote returns keyboard for selecting participant when creating a note.
func SelectParticipantForNote(participants []*participantsv1.Participant, nextCursor string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📄 Без участника", "participant:select:none"),
	))
	for _, p := range participants {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(p.Name, "participant:select:"+p.Id),
		))
	}
	if nextCursor != "" {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➡️ Далее", "page:select_participant:"+nextCursor),
		))
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
