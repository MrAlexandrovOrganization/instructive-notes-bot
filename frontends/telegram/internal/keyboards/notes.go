package keyboards

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	notesv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/notes/v1"
)

// NotesListOpts configures the notes list keyboard.
type NotesListOpts struct {
	Notes         []*notesv1.Note
	Total         int32
	Offset        int32
	HasNext       bool
	BackTo        string // callback for "Вернуться"
	ParticipantID string // if set, show "Написать заметку" button
	PageSize      int32
}

// NotesList returns an inline keyboard for a list of notes with stable numbering.
func NotesList(opts NotesListOpts) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, n := range opts.Notes {
		icon := "📌" // unassigned
		if n.ParticipantId != "" {
			icon = "👤"
		}
		label := fmt.Sprintf("%s %s", icon, truncate(n.Text, 33))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, "note:view:"+n.Id),
		))
	}

	// Pagination row: [⬅️ Назад] [➡️ Далее]
	var navRow []tgbotapi.InlineKeyboardButton
	if opts.Offset > 0 {
		prevOffset := opts.Offset - opts.PageSize
		if prevOffset < 0 {
			prevOffset = 0
		}
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("page:notes:%d", prevOffset)))
	}
	if opts.HasNext {
		nextOffset := opts.Offset + int32(len(opts.Notes))
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("➡️ Далее", fmt.Sprintf("page:notes:%d", nextOffset)))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	if opts.ParticipantID != "" {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✍️ Написать заметку", "participant:note:"+opts.ParticipantID),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("↩️ Вернуться", opts.BackTo),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// NoteActions returns action buttons for a single note.
func NoteActions(noteID string, hasParticipant bool, backTo string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	if !hasParticipant {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👤 Назначить участника", "note:assign:"+noteID),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить", "note:delete:"+noteID),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("↩️ Вернуться", backTo),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// MyNotesMenu returns the inline keyboard for the "My notes" section.
func MyNotesMenu(unassignedCount int) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
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
