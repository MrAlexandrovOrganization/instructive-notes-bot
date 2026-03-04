package keyboards

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	notesv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/notes/v1"
)

// NotesList returns an inline keyboard for a list of notes.
func NotesList(notes []*notesv1.Note, nextCursor string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, n := range notes {
		label := truncate(n.Text, 40)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, "note:view:"+n.Id),
		))
	}
	if nextCursor != "" {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➡️ Далее", "page:notes:"+nextCursor),
		))
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// NoteActions returns action buttons for a single note.
func NoteActions(noteID string, hasParticipant bool) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	if !hasParticipant {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👤 Назначить участника", "note:assign:"+noteID),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить", "note:delete:"+noteID),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// MyNotesMenu returns the inline keyboard for the "My notes" section.
func MyNotesMenu(unassignedCount int) tgbotapi.InlineKeyboardMarkup {
	rows := [][]tgbotapi.InlineKeyboardButton{}
	if unassignedCount > 0 {
		label := fmt.Sprintf("📄 Без участника (%d)", unassignedCount)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, "notes:unassigned"),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("👥 По участникам", "notes:by_participant"),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "…"
}
