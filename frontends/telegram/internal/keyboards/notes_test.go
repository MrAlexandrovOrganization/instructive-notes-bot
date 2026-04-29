package keyboards_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notesv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/notes/v1"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/keyboards"
)

func TestNotesList_EmptyNotes(t *testing.T) {
	t.Parallel()
	kb := keyboards.NotesList(keyboards.NotesListOpts{
		Total:    0,
		BackTo:   "back:menu",
		PageSize: 8,
	})
	// Only "Вернуться" button.
	require.Len(t, kb.InlineKeyboard, 1)
	assert.Equal(t, "back:menu", *kb.InlineKeyboard[0][0].CallbackData)
}

func TestNotesList_SinglePage(t *testing.T) {
	t.Parallel()
	notes := []*notesv1.Note{
		{Id: "n1", Text: "First note"},
		{Id: "n2", Text: "Second note"},
	}
	kb := keyboards.NotesList(keyboards.NotesListOpts{
		Notes:    notes,
		Total:    2,
		Offset:   0,
		HasNext:  false,
		BackTo:   "back:menu",
		PageSize: 8,
	})
	// 2 notes + "Вернуться" = 3 rows, no pagination row.
	require.Len(t, kb.InlineKeyboard, 3)
	assert.Contains(t, *kb.InlineKeyboard[0][0].CallbackData, "note:view:n1")
	assert.Contains(t, *kb.InlineKeyboard[1][0].CallbackData, "note:view:n2")
	assert.Equal(t, "back:menu", *kb.InlineKeyboard[2][0].CallbackData)
}

func TestNotesList_HasNextPage(t *testing.T) {
	t.Parallel()
	notes := make([]*notesv1.Note, 8)
	for i := range notes {
		notes[i] = &notesv1.Note{Id: "n", Text: "text"}
	}
	kb := keyboards.NotesList(keyboards.NotesListOpts{
		Notes:    notes,
		Total:    20,
		Offset:   0,
		HasNext:  true,
		BackTo:   "back:menu",
		PageSize: 8,
	})
	// 8 notes + 1 nav row + 1 back row = 10 rows.
	require.Len(t, kb.InlineKeyboard, 10)
	navRow := kb.InlineKeyboard[8]
	// Only "Далее" (no prev on first page).
	require.Len(t, navRow, 1)
	assert.Equal(t, "page:notes:8", *navRow[0].CallbackData)
}

func TestNotesList_MiddlePage_BothButtons(t *testing.T) {
	t.Parallel()
	notes := make([]*notesv1.Note, 8)
	for i := range notes {
		notes[i] = &notesv1.Note{Id: "n", Text: "text"}
	}
	kb := keyboards.NotesList(keyboards.NotesListOpts{
		Notes:    notes,
		Total:    30,
		Offset:   8,
		HasNext:  true,
		BackTo:   "back:menu",
		PageSize: 8,
	})
	navRow := kb.InlineKeyboard[8]
	require.Len(t, navRow, 2)
	assert.Equal(t, "page:notes:0", *navRow[0].CallbackData)   // Назад → offset 0
	assert.Equal(t, "page:notes:16", *navRow[1].CallbackData)  // Далее → offset 16
}

func TestNotesList_LastPage_OnlyPrevButton(t *testing.T) {
	t.Parallel()
	notes := []*notesv1.Note{{Id: "n1", Text: "last"}}
	kb := keyboards.NotesList(keyboards.NotesListOpts{
		Notes:    notes,
		Total:    9,
		Offset:   8,
		HasNext:  false,
		BackTo:   "back:menu",
		PageSize: 8,
	})
	// 1 note + 1 nav row (only prev) + 1 back row = 3 rows.
	require.Len(t, kb.InlineKeyboard, 3)
	navRow := kb.InlineKeyboard[1]
	require.Len(t, navRow, 1)
	assert.Equal(t, "page:notes:0", *navRow[0].CallbackData)
}

func TestNotesList_WithParticipantID(t *testing.T) {
	t.Parallel()
	notes := []*notesv1.Note{{Id: "n1", Text: "note"}}
	kb := keyboards.NotesList(keyboards.NotesListOpts{
		Notes:         notes,
		Total:         1,
		BackTo:        "back:participant:p1",
		ParticipantID: "p1",
		PageSize:      8,
	})
	// 1 note + "Написать заметку" + "Вернуться" = 3 rows.
	require.Len(t, kb.InlineKeyboard, 3)
	assert.Equal(t, "participant:note:p1", *kb.InlineKeyboard[1][0].CallbackData)
	assert.Equal(t, "back:participant:p1", *kb.InlineKeyboard[2][0].CallbackData)
}

func TestNoteActions_Unassigned(t *testing.T) {
	t.Parallel()
	kb := keyboards.NoteActions("note-1", false, "back:notes_list")
	// "Назначить участника" + "Удалить" + "Вернуться" = 3 rows.
	require.Len(t, kb.InlineKeyboard, 3)
	assert.Contains(t, *kb.InlineKeyboard[0][0].CallbackData, "note:assign:note-1")
	assert.Contains(t, *kb.InlineKeyboard[1][0].CallbackData, "note:delete:note-1")
	assert.Equal(t, "back:notes_list", *kb.InlineKeyboard[2][0].CallbackData)
}

func TestNoteActions_Assigned(t *testing.T) {
	t.Parallel()
	kb := keyboards.NoteActions("note-1", true, "back:notes_list")
	// No "Назначить" → "Удалить" + "Вернуться" = 2 rows.
	require.Len(t, kb.InlineKeyboard, 2)
	assert.Contains(t, *kb.InlineKeyboard[0][0].CallbackData, "note:delete:note-1")
}
