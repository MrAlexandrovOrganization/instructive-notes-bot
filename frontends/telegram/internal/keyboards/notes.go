package keyboards

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	notesv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/notes/v1"
)

// NotesListOpts configures the notes list keyboard.
type NotesListOpts struct {
	Notes         []*notesv1.Note
	NextCursor    string
	Total         int32
	Offset        int32
	BackTo        string // callback for "Вернуться", e.g. "back:menu"
	HasPrevPage   bool   // show "⬅️ Назад" pagination button
	ParticipantID string // if set, show "Написать заметку" button
}

// NotesList returns an inline keyboard for a list of notes with stable numbering.
func NotesList(opts NotesListOpts) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for i, n := range opts.Notes {
		num := opts.Total - opts.Offset - int32(i)
		label := fmt.Sprintf("#%d %s", num, truncate(n.Text, 35))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, "note:view:"+n.Id),
		))
	}

	// Pagination row: [⬅️ Назад] [➡️ Далее]
	var navRow []tgbotapi.InlineKeyboardButton
	if opts.HasPrevPage {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "page:notes:b"))
	}
	if opts.NextCursor != "" {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("➡️ Далее", "page:notes:f:"+opts.NextCursor))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	// "Написать заметку" for participant notes view.
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
