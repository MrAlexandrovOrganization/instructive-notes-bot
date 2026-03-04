package handlers

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	commonv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/common/v1"
	notesv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/notes/v1"
	participantsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/participants/v1"
	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/keyboards"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/state"
)

// NotesHandler handles note-related interactions.
type NotesHandler struct {
	*Base
}

// NewNotesHandler creates a new NotesHandler.
func NewNotesHandler(base *Base) *NotesHandler {
	return &NotesHandler{Base: base}
}

// HandleNewNote starts the note creation flow.
func (h *NotesHandler) HandleNewNote(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) error {
	// Show participant selection.
	resp, err := h.Clients.Participants.ListParticipants(ctx, &participantsv1.ListParticipantsRequest{
		Pagination: &commonv1.Pagination{Limit: 10},
	})
	if err != nil {
		return h.sendError(msg.Chat.ID, "Не удалось загрузить участников.")
	}

	h.States.Set(msg.From.ID, &state.UserContext{State: state.StateSelectingParticipantForNote})

	nextCursor := ""
	if resp.PageInfo != nil && resp.PageInfo.HasNext {
		nextCursor = resp.PageInfo.NextCursor
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, "Выберите участника для заметки или создайте без привязки:")
	reply.ReplyMarkup = keyboards.SelectParticipantForNote(resp.Participants, nextCursor)
	_, err = h.Bot.Send(reply)
	return err
}

// HandleParticipantSelectedForNote handles participant selection callback during note creation.
func (h *NotesHandler) HandleParticipantSelectedForNote(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, participantID string) error {
	userCtx := h.States.Get(cb.From.ID)
	userCtx.State = state.StateWritingNoteText
	if participantID == "none" {
		userCtx.PendingData = ""
	} else {
		userCtx.PendingData = participantID
	}
	h.States.Set(cb.From.ID, userCtx)

	h.answerCallback(cb.ID, "")

	reply := tgbotapi.NewMessage(cb.Message.Chat.ID, "✍️ Напишите текст заметки:")
	reply.ReplyMarkup = keyboards.CancelKeyboard()
	_, err := h.Bot.Send(reply)
	return err
}

// HandleNoteText handles text input when user is writing a note.
func (h *NotesHandler) HandleNoteText(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) error {
	userCtx := h.States.Get(msg.From.ID)

	var participantID *string
	if userCtx.PendingData != "" {
		id := userCtx.PendingData
		participantID = &id
	}

	_, err := h.Clients.Notes.CreateNote(ctx, &notesv1.CreateNoteRequest{
		AuthorId:      user.Id,
		ParticipantId: derefString(participantID),
		Text:          msg.Text,
	})
	if err != nil {
		return h.sendError(msg.Chat.ID, "Не удалось сохранить заметку.")
	}

	h.States.Reset(msg.From.ID)

	reply := tgbotapi.NewMessage(msg.Chat.ID, "✅ Заметка сохранена!")
	reply.ReplyMarkup = keyboards.MainMenu(user.Role)
	_, err = h.Bot.Send(reply)
	return err
}

// HandleMyNotes shows the current user's notes.
func (h *NotesHandler) HandleMyNotes(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) error {
	// Get unassigned count.
	unassignedResp, err := h.Clients.Notes.ListNotes(ctx, &notesv1.ListNotesRequest{
		AuthorId:       user.Id,
		UnassignedOnly: true,
		Pagination:     &commonv1.Pagination{Limit: 100},
	})
	if err != nil {
		return h.sendError(msg.Chat.ID, "Не удалось загрузить заметки.")
	}

	text := "📋 *Мои заметки*\n\n"
	if len(unassignedResp.Notes) > 0 {
		text += fmt.Sprintf("📄 *Без участника:* %d заметок\n", len(unassignedResp.Notes))
	}

	// Get all notes for listing.
	allResp, err := h.Clients.Notes.ListNotes(ctx, &notesv1.ListNotesRequest{
		AuthorId:   user.Id,
		Pagination: &commonv1.Pagination{Limit: 20},
	})
	if err != nil {
		return h.sendError(msg.Chat.ID, "Не удалось загрузить заметки.")
	}

	nextCursor := ""
	if allResp.PageInfo != nil && allResp.PageInfo.HasNext {
		nextCursor = allResp.PageInfo.NextCursor
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = keyboards.NotesList(allResp.Notes, nextCursor)
	_, err = h.Bot.Send(reply)
	return err
}

// HandleAllNotes shows all notes (admin/root only).
func (h *NotesHandler) HandleAllNotes(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) error {
	resp, err := h.Clients.Notes.ListNotes(ctx, &notesv1.ListNotesRequest{
		AllNotes:   true,
		Pagination: &commonv1.Pagination{Limit: 20},
	})
	if err != nil {
		return h.sendError(msg.Chat.ID, "Не удалось загрузить заметки.")
	}

	nextCursor := ""
	if resp.PageInfo != nil && resp.PageInfo.HasNext {
		nextCursor = resp.PageInfo.NextCursor
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, "📊 *Все заметки*")
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = keyboards.NotesList(resp.Notes, nextCursor)
	_, err = h.Bot.Send(reply)
	return err
}

// HandleNoteView shows a single note.
func (h *NotesHandler) HandleNoteView(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, noteID string) error {
	h.answerCallback(cb.ID, "")

	n, err := h.Clients.Notes.GetNote(ctx, &notesv1.GetNoteRequest{Id: noteID})
	if err != nil {
		return h.sendError(cb.Message.Chat.ID, "Заметка не найдена.")
	}

	text := fmt.Sprintf("📝 *Заметка*\n\n%s", escapeMarkdown(n.Text))
	if n.ParticipantName != "" {
		text = fmt.Sprintf("📝 *Заметка о %s*\n\n%s", escapeMarkdown(n.ParticipantName), escapeMarkdown(n.Text))
	}
	text += fmt.Sprintf("\n\n_Автор: %s_", escapeMarkdown(n.AuthorName))

	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, text)
	edit.ParseMode = "MarkdownV2"
	edit.ReplyMarkup = &[]tgbotapi.InlineKeyboardMarkup{keyboards.NoteActions(noteID, n.ParticipantId != "")}[0]
	_, err = h.Bot.Send(edit)
	return err
}

// HandleNoteDelete deletes a note.
func (h *NotesHandler) HandleNoteDelete(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, noteID string) error {
	_, err := h.Clients.Notes.DeleteNote(ctx, &notesv1.DeleteNoteRequest{Id: noteID})
	if err != nil {
		h.answerCallback(cb.ID, "Ошибка удаления")
		return err
	}
	h.answerCallback(cb.ID, "✅ Заметка удалена")
	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, "🗑 Заметка удалена.")
	_, err = h.Bot.Send(edit)
	return err
}

// HandleNoteAssignStart starts assigning a note to a participant.
func (h *NotesHandler) HandleNoteAssignStart(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, noteID string) error {
	h.answerCallback(cb.ID, "")
	h.States.Set(cb.From.ID, &state.UserContext{
		State:         state.StateAssigningNoteToParticipant,
		PendingNoteID: noteID,
	})

	resp, err := h.Clients.Participants.ListParticipants(ctx, &participantsv1.ListParticipantsRequest{
		Pagination: &commonv1.Pagination{Limit: 10},
	})
	if err != nil {
		return h.sendError(cb.Message.Chat.ID, "Не удалось загрузить участников.")
	}

	nextCursor := ""
	if resp.PageInfo != nil && resp.PageInfo.HasNext {
		nextCursor = resp.PageInfo.NextCursor
	}

	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, "Выберите участника для заметки:")
	kb := keyboards.SelectParticipantForNote(resp.Participants, nextCursor)
	edit.ReplyMarkup = &kb
	_, err = h.Bot.Send(edit)
	return err
}

// HandleNoteAssignParticipant completes assigning a note to a participant.
func (h *NotesHandler) HandleNoteAssignParticipant(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, participantID string) error {
	userCtx := h.States.Get(cb.From.ID)
	noteID := userCtx.PendingNoteID
	h.States.Reset(cb.From.ID)

	_, err := h.Clients.Notes.AssignNoteToParticipant(ctx, &notesv1.AssignNoteToParticipantRequest{
		NoteId:        noteID,
		ParticipantId: participantID,
	})
	if err != nil {
		h.answerCallback(cb.ID, "Ошибка назначения")
		return err
	}
	h.answerCallback(cb.ID, "✅ Заметка назначена")
	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, "✅ Заметка назначена участнику.")
	_, err = h.Bot.Send(edit)
	return err
}

// HandleUnassignedNotes shows unassigned notes.
func (h *NotesHandler) HandleUnassignedNotes(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User) error {
	h.answerCallback(cb.ID, "")
	resp, err := h.Clients.Notes.ListNotes(ctx, &notesv1.ListNotesRequest{
		AuthorId:       user.Id,
		UnassignedOnly: true,
		Pagination:     &commonv1.Pagination{Limit: 20},
	})
	if err != nil {
		return h.sendError(cb.Message.Chat.ID, "Не удалось загрузить заметки.")
	}

	nextCursor := ""
	if resp.PageInfo != nil && resp.PageInfo.HasNext {
		nextCursor = resp.PageInfo.NextCursor
	}

	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, "📄 *Заметки без участника*")
	edit.ParseMode = "Markdown"
	kb := keyboards.NotesList(resp.Notes, nextCursor)
	edit.ReplyMarkup = &kb
	_, err = h.Bot.Send(edit)
	return err
}

// HandleNotesByParticipant shows notes for a participant.
func (h *NotesHandler) HandleNotesByParticipant(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, participantID string) error {
	h.answerCallback(cb.ID, "")
	resp, err := h.Clients.Notes.ListNotes(ctx, &notesv1.ListNotesRequest{
		AuthorId:      user.Id,
		ParticipantId: participantID,
		Pagination:    &commonv1.Pagination{Limit: 20},
	})
	if err != nil {
		return h.sendError(cb.Message.Chat.ID, "Не удалось загрузить заметки.")
	}

	nextCursor := ""
	if resp.PageInfo != nil && resp.PageInfo.HasNext {
		nextCursor = resp.PageInfo.NextCursor
	}

	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, "📋 *Заметки по участнику*")
	edit.ParseMode = "Markdown"
	kb := keyboards.NotesList(resp.Notes, nextCursor)
	edit.ReplyMarkup = &kb
	_, err = h.Bot.Send(edit)
	return err
}

func (h *NotesHandler) sendError(chatID int64, text string) error {
	_, err := h.Bot.Send(tgbotapi.NewMessage(chatID, "❌ "+text))
	return err
}

func (h *NotesHandler) answerCallback(callbackID, text string) {
	cb := tgbotapi.NewCallback(callbackID, text)
	_, _ = h.Bot.Request(cb)
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func escapeMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]",
		"(", "\\(", ")", "\\)", "~", "\\~", "`", "\\`",
		">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
		"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}",
		".", "\\.", "!", "\\!",
	)
	return replacer.Replace(s)
}
