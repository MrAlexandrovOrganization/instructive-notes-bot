package keyboards

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	groupsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/groups/v1"
	participantsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/participants/v1"
	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
)

// ParticipantsListOpts configures the participants list keyboard.
type ParticipantsListOpts struct {
	Participants []*participantsv1.Participant
	Role         usersv1.Role
	BackTo       string
	Offset       int32
	HasNext      bool
	PageSize     int32
}

// ParticipantsList returns an inline keyboard for listing participants.
func ParticipantsList(opts ParticipantsListOpts) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, p := range opts.Participants {
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

	// Pagination row.
	var navRow []tgbotapi.InlineKeyboardButton
	if opts.Offset > 0 {
		prevOffset := opts.Offset - opts.PageSize
		if prevOffset < 0 {
			prevOffset = 0
		}
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("page:participants:%d", prevOffset)))
	}
	if opts.HasNext {
		nextOffset := opts.Offset + int32(len(opts.Participants))
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("➡️ Далее", fmt.Sprintf("page:participants:%d", nextOffset)))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	role := opts.Role
	if role == usersv1.Role_ROLE_ADMIN || role == usersv1.Role_ROLE_ROOT || role == usersv1.Role_ROLE_ORGANIZER || role == usersv1.Role_ROLE_CURATOR {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить участника", "participant:add"),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("↩️ Вернуться", opts.BackTo),
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
		tgbotapi.NewInlineKeyboardButtonData("↩️ Вернуться", "back:participants_list"),
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
func GroupsListForAssign(groups []*groupsv1.Group, userGroupID, noteID string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	// "All participants" option — no group filter.
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("👥 Все участники", "group:for_note:all"),
	))

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

// SelectParticipantOpts configures the participant selection keyboard for note assignment.
type SelectParticipantOpts struct {
	Participants []*participantsv1.Participant
	NoteID       string
	Offset       int32
	HasNext      bool
	PageSize     int32
}

// SelectParticipantForNote returns keyboard for selecting participant during note assignment.
func SelectParticipantForNote(opts SelectParticipantOpts) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, p := range opts.Participants {
		label := p.Name
		if p.GroupName != "" {
			label += fmt.Sprintf(" (%s)", p.GroupName)
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, "participant:select:"+p.Id),
		))
	}

	var navRow []tgbotapi.InlineKeyboardButton
	if opts.Offset > 0 {
		prevOffset := opts.Offset - opts.PageSize
		if prevOffset < 0 {
			prevOffset = 0
		}
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("page:assign_participant:%d", prevOffset)))
	}
	if opts.HasNext {
		nextOffset := opts.Offset + int32(len(opts.Participants))
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("➡️ Далее", fmt.Sprintf("page:assign_participant:%d", nextOffset)))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("↩️ Вернуться", "note:assign:"+opts.NoteID),
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
