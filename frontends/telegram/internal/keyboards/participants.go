package keyboards

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	groupsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/groups/v1"
	participantsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/participants/v1"
	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
)

// ParticipantsList returns an inline keyboard for listing participants.
// backTo is the callback for the back button (e.g. "back:menu", "back:group_view:{id}").
func ParticipantsList(participants []*participantsv1.Participant, nextCursor string, role usersv1.Role, backTo string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, p := range participants {
		label := p.Name
		if p.GroupName != "" {
			label += fmt.Sprintf(" (%s)", p.GroupName)
		}
		if p.NotesCount > 0 {
			label += " 📝"
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
	if role == usersv1.Role_ROLE_ADMIN || role == usersv1.Role_ROLE_ROOT || role == usersv1.Role_ROLE_ORGANIZER || role == usersv1.Role_ROLE_CURATOR {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить участника", "participant:add"),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("↩️ Вернуться", backTo),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// ParticipantView returns the action keyboard for a single participant.
func ParticipantView(participantID string, _ usersv1.Role) tgbotapi.InlineKeyboardMarkup {
	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Заметки", "notes:participant:"+participantID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🖼 Фото", "participant:photo:"+participantID),
		),
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("↩️ Вернуться", "back:participants"),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// ParticipantPhotoView returns buttons after viewing an existing photo.
func ParticipantPhotoView(participantID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📷 Обновить фото", "participant:update_photo:"+participantID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("↩️ Вернуться", "back:participant:"+participantID),
		),
	)
}

// GroupsListForAssign returns an inline keyboard of groups for note assignment.
// noteID is used for the back button to return to the note.
func GroupsListForAssign(groups []*groupsv1.Group, userGroupID, noteID string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	// Put user's own group first.
	for _, g := range groups {
		if g.Id == userGroupID {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⭐ "+g.Name, "group:for_note:"+g.Id),
			))
			break
		}
	}
	for _, g := range groups {
		if g.Id == userGroupID {
			continue
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(g.Name, "group:for_note:"+g.Id),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("↩️ Вернуться", "note:view:"+noteID),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// SelectParticipantForNote returns keyboard for selecting participant during note assignment.
// noteID is used for the back button to return to group selection.
func SelectParticipantForNote(participants []*participantsv1.Participant, nextCursor, noteID string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, p := range participants {
		label := p.Name
		if p.GroupName != "" {
			label += fmt.Sprintf(" (%s)", p.GroupName)
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, "participant:select:"+p.Id),
		))
	}
	if nextCursor != "" {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➡️ Далее", "page:select_participant:"+nextCursor),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "note:assign:"+noteID),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// GroupsListForBrowse returns an inline keyboard of groups for browsing participants.
func GroupsListForBrowse(groups []*groupsv1.Group) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, g := range groups {
		label := g.Name
		if g.Description != "" {
			label += " — " + g.Description
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, "group:view:"+g.Id),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("➕ Добавить отряд", "admin:add_group"),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("↩️ Вернуться", "back:admin"),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
