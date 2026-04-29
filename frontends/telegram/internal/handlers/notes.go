package handlers

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/keyboards"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/state"
	commonv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/common/v1"
	groupsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/groups/v1"
	notesv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/notes/v1"
	participantsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/participants/v1"
	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
)

// NotesHandler handles note-related interactions.
type NotesHandler struct {
	*Base
}

// NewNotesHandler creates a new NotesHandler.
func NewNotesHandler(base *Base) *NotesHandler {
	return &NotesHandler{Base: base}
}

// HandleQuickNote saves any incoming text as a note immediately (no participant).
func (h *NotesHandler) HandleQuickNote(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) error {
	_, err := h.Clients.Notes.CreateNote(ctx, &notesv1.CreateNoteRequest{
		AuthorId: user.Id,
		Text:     msg.Text,
	})
	if err != nil {
		return h.SendError(msg.Chat.ID, "Не удалось сохранить заметку.")
	}
	return h.SendPlain(msg.Chat.ID, "✅ Заметка сохранена!", keyboards.MainMenu(user.Role))
}

// HandleVoiceNote transcribes a voice message or video note.
// If saveNote is true, the transcription is also saved as an unassigned note.
func (h *NotesHandler) HandleVoiceNote(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User, saveNote bool) error {
	kind := "voice"
	if msg.VideoNote != nil {
		kind = "video_note"
	}
	slog.Info("voice note received", "kind", kind, "user_id", msg.From.ID, "whisper_configured", h.Whisper != nil)

	if h.Whisper == nil {
		if !saveNote {
			return nil // no whisper, nothing to do
		}
		slog.Warn("whisper not configured, saving placeholder", "user_id", msg.From.ID)
		_, err := h.Clients.Notes.CreateNote(ctx, &notesv1.CreateNoteRequest{
			AuthorId: user.Id,
			Text:     "[голосовое сообщение]",
		})
		if err != nil {
			return h.SendError(msg.Chat.ID, "Не удалось сохранить заметку.")
		}
		return h.SendPlain(msg.Chat.ID, "✅ Заметка сохранена!", keyboards.MainMenu(user.Role))
	}

	statusMsg, err := h.Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Расшифровываю..."))
	if err != nil {
		return err
	}

	go h.transcribeAndSave(context.Background(), msg, user, statusMsg.MessageID, saveNote)

	kb := keyboards.MainMenu(user.Role)
	m := tgbotapi.NewMessage(msg.Chat.ID, "📋 Главное меню")
	m.ReplyMarkup = kb
	_, err = h.Bot.Send(m)
	return err

	return nil
}

func (h *NotesHandler) transcribeAndSave(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User, statusMsgID int, saveNote bool) {
	editStatus := func(text string) {
		edit := tgbotapi.NewEditMessageText(msg.Chat.ID, statusMsgID, text)
		_, _ = h.Bot.Send(edit)
	}

	fileID, format := voiceFileID(msg)
	fileURL, err := h.Bot.GetFileDirectURL(fileID)
	if err != nil {
		slog.Error("get voice file url", "error", err)
		editStatus("❌ Не удалось получить файл.")
		return
	}

	resp, err := http.Get(fileURL) //nolint:noctx
	if err != nil {
		slog.Error("download voice file", "error", err)
		editStatus("❌ Не удалось скачать файл.")
		return
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	if _, err = buf.ReadFrom(resp.Body); err != nil {
		slog.Error("read voice file", "error", err)
		editStatus("❌ Не удалось прочитать файл.")
		return
	}

	text, err := h.Whisper.Transcribe(ctx, &buf, format)
	if err != nil {
		slog.Error("transcribe voice", "error", err)
		editStatus("❌ Не удалось расшифровать сообщение.")
		return
	}
	if text == "" {
		text = "(тишина)"
	}

	if saveNote {
		_, err = h.Clients.Notes.CreateNote(ctx, &notesv1.CreateNoteRequest{
			AuthorId: user.Id,
			Text:     text,
		})
		if err != nil {
			slog.Error("save transcribed note", "error", err)
			editStatus("❌ Не удалось сохранить заметку.")
			return
		}
	}

	prefix := "✅ "
	if !saveNote {
		prefix = "🎤 "
	}
	edit := tgbotapi.NewEditMessageText(msg.Chat.ID, statusMsgID,
		fmt.Sprintf("%s%s", prefix, EscapeMarkdown(text)))
	edit.ParseMode = "MarkdownV2"
	_, _ = h.Bot.Send(edit)
}

func voiceFileID(msg *tgbotapi.Message) (fileID, format string) {
	if msg.Voice != nil {
		return msg.Voice.FileID, "ogg"
	}
	return msg.VideoNote.FileID, "mp4"
}

// HandleNoteText handles text input when user is writing a note in StateWritingNoteText.
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
		return h.SendError(msg.Chat.ID, "Не удалось сохранить заметку.")
	}

	h.States.Reset(msg.From.ID)
	return h.SendPlain(msg.Chat.ID, "✅ Заметка сохранена!", keyboards.MainMenu(user.Role))
}

const notesPageSize = 8

// handleNotesList is a shared helper for all notes list views.
func (h *NotesHandler) handleNotesList(
	ctx context.Context,
	chatID int64, msgID int,
	user *usersv1.User, userID int64,
	notesCtx state.NotesContext,
	participantID string,
	title string,
	backTo string,
) error {
	req := h.buildListRequest(user, notesCtx, participantID, 0)

	resp, err := h.Clients.Notes.ListNotes(ctx, req)
	if err != nil {
		return h.SendError(chatID, "Не удалось загрузить заметки.")
	}

	if len(resp.Notes) == 0 {
		emptyText := title + "\n\nЗаметок пока нет\\. Просто отправьте текст — он сохранится как заметка\\."
		var rows [][]tgbotapi.InlineKeyboardButton
		if participantID != "" {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✍️ Написать заметку", "participant:note:"+participantID),
			))
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("↩️ Вернуться", backTo),
		))
		kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
		return h.EditMD(chatID, msgID, emptyText, &kb)
	}

	total, hasNext := pageInfoFields(resp.PageInfo)

	// Save pagination context for subsequent page requests.
	h.States.Set(userID, &state.UserContext{
		NotesCtx:    notesCtx,
		PendingData: participantID,
	})

	kb := keyboards.NotesList(keyboards.NotesListOpts{
		Notes:         resp.Notes,
		Total:         total,
		Offset:        0,
		HasNext:       hasNext,
		BackTo:        backTo,
		ParticipantID: participantID,
		PageSize:      notesPageSize,
	})
	return h.EditMD(chatID, msgID,
		fmt.Sprintf("%s \\(%d\\)", title, total), &kb)
}

// HandleMyNotes shows the current user's notes.
func (h *NotesHandler) HandleMyNotes(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User) error {
	return h.handleNotesList(ctx, cb.Message.Chat.ID, cb.Message.MessageID,
		user, cb.From.ID, state.NotesCtxMy, "", "📋 *Мои заметки*", "back:menu")
}

// HandleAllNotes shows all notes (admin/root only).
func (h *NotesHandler) HandleAllNotes(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User) error {
	return h.handleNotesList(ctx, cb.Message.Chat.ID, cb.Message.MessageID,
		user, cb.From.ID, state.NotesCtxAll, "", "📊 *Все заметки*", "back:menu")
}

// HandleUnassignedNotes shows unassigned notes.
func (h *NotesHandler) HandleUnassignedNotes(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User) error {
	h.AnswerCallback(cb.ID, "")
	return h.handleNotesList(ctx, cb.Message.Chat.ID, cb.Message.MessageID,
		user, cb.From.ID, state.NotesCtxUnassigned, "", "📄 *Заметки без участника*", "back:notes")
}

// HandleNotesByParticipant shows notes for a participant.
func (h *NotesHandler) HandleNotesByParticipant(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, participantID string) error {
	h.AnswerCallback(cb.ID, "")
	backTo := "back:participant:" + participantID
	return h.handleNotesList(ctx, cb.Message.Chat.ID, cb.Message.MessageID,
		user, cb.From.ID, state.NotesCtxParticipant, participantID,
		"📋 *Заметки по участнику*", backTo)
}

// HandleNotesPage handles pagination by offset.
func (h *NotesHandler) HandleNotesPage(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, offset int32) error {
	h.AnswerCallback(cb.ID, "")

	userCtx := h.States.Get(cb.From.ID)
	req := h.buildListRequest(user, userCtx.NotesCtx, userCtx.PendingData, offset)

	resp, err := h.Clients.Notes.ListNotes(ctx, req)
	if err != nil {
		return h.SendError(cb.Message.Chat.ID, "Не удалось загрузить заметки.")
	}

	total, hasNext := pageInfoFields(resp.PageInfo)

	backTo := h.notesBackTo(userCtx)
	participantID := ""
	if userCtx.NotesCtx == state.NotesCtxParticipant {
		participantID = userCtx.PendingData
	}

	kb := keyboards.NotesList(keyboards.NotesListOpts{
		Notes:         resp.Notes,
		Total:         total,
		Offset:        offset,
		HasNext:       hasNext,
		BackTo:        backTo,
		ParticipantID: participantID,
		PageSize:      notesPageSize,
	})
	title := h.notesTitle(userCtx)
	return h.EditMD(cb.Message.Chat.ID, cb.Message.MessageID,
		fmt.Sprintf("%s \\(%d\\)", title, total), &kb)
}

func (h *NotesHandler) buildListRequest(user *usersv1.User, notesCtx state.NotesContext, participantID string, offset int32) *notesv1.ListNotesRequest {
	req := &notesv1.ListNotesRequest{
		Pagination: &commonv1.Pagination{Limit: notesPageSize, Offset: offset},
	}
	admin := user.Role == usersv1.Role_ROLE_ADMIN || user.Role == usersv1.Role_ROLE_ROOT
	switch notesCtx {
	case state.NotesCtxMy:
		req.AuthorId = user.Id
	case state.NotesCtxAll:
		req.AllNotes = true
	case state.NotesCtxUnassigned:
		if !admin {
			req.AuthorId = user.Id
		} else {
			req.AllNotes = true
		}
		req.UnassignedOnly = true
	case state.NotesCtxParticipant:
		if !admin {
			req.AuthorId = user.Id
		} else {
			req.AllNotes = true
		}
		req.ParticipantId = participantID
	}
	return req
}

func (h *NotesHandler) notesTitle(userCtx *state.UserContext) string {
	switch userCtx.NotesCtx {
	case state.NotesCtxAll:
		return "📊 *Все заметки*"
	case state.NotesCtxUnassigned:
		return "📄 *Заметки без участника*"
	case state.NotesCtxParticipant:
		return "📋 *Заметки по участнику*"
	default:
		return "📋 *Мои заметки*"
	}
}

func (h *NotesHandler) notesBackTo(userCtx *state.UserContext) string {
	switch userCtx.NotesCtx {
	case state.NotesCtxParticipant:
		return "back:participant:" + userCtx.PendingData
	case state.NotesCtxUnassigned:
		return "back:notes"
	default:
		return "back:menu"
	}
}

// HandleNoteView shows a single note.
func (h *NotesHandler) HandleNoteView(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, noteID string) error {
	h.AnswerCallback(cb.ID, "")

	n, err := h.Clients.Notes.GetNote(ctx, &notesv1.GetNoteRequest{Id: noteID})
	if err != nil {
		return h.SendError(cb.Message.Chat.ID, "Заметка не найдена.")
	}

	text := fmt.Sprintf("📝 *Заметка*\n\n%s", EscapeMarkdown(n.Text))
	if n.ParticipantName != "" {
		text = fmt.Sprintf("📝 *Заметка о %s*\n\n%s", EscapeMarkdown(n.ParticipantName), EscapeMarkdown(n.Text))
	}
	text += fmt.Sprintf("\n\n_Автор: %s_", EscapeMarkdown(n.AuthorName))

	kb := keyboards.NoteActions(noteID, n.ParticipantId != "", "back:notes_list")
	return h.EditMD(cb.Message.Chat.ID, cb.Message.MessageID, text, &kb)
}

// HandleNoteDelete deletes a note.
func (h *NotesHandler) HandleNoteDelete(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, noteID string) error {
	_, err := h.Clients.Notes.DeleteNote(ctx, &notesv1.DeleteNoteRequest{Id: noteID})
	if err != nil {
		h.AnswerCallback(cb.ID, "Ошибка удаления")
		return err
	}
	h.AnswerCallback(cb.ID, "✅ Заметка удалена")
	return h.EditMD(cb.Message.Chat.ID, cb.Message.MessageID,
		"🗑 Заметка удалена\\.", menuKeyboard(user))
}

// HandleNoteAssignStart starts assigning a note to a participant by showing groups.
func (h *NotesHandler) HandleNoteAssignStart(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, noteID string) error {
	h.AnswerCallback(cb.ID, "")
	h.States.Set(cb.From.ID, &state.UserContext{
		State:         state.StateAssigningNoteToParticipant,
		PendingNoteID: noteID,
	})

	resp, err := h.Clients.Groups.ListGroups(ctx, &groupsv1.ListGroupsRequest{
		Pagination: &commonv1.Pagination{Limit: 50},
	})
	if err != nil {
		return h.SendError(cb.Message.Chat.ID, "Не удалось загрузить отряды.")
	}

	if len(resp.Groups) == 0 {
		return h.showParticipantsForAssign(ctx, cb, groupID(""), noteID)
	}

	kb := keyboards.GroupsListForAssign(resp.Groups, user.GroupId, noteID)
	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, "Выберите отряд:")
	edit.ReplyMarkup = &kb
	_, err = h.Bot.Send(edit)
	return err
}

// HandleGroupForNote shows participants from a group for note assignment.
func (h *NotesHandler) HandleGroupForNote(ctx context.Context, cb *tgbotapi.CallbackQuery, _ *usersv1.User, gID string) error {
	h.AnswerCallback(cb.ID, "")
	userCtx := h.States.Get(cb.From.ID)
	if gID == "all" {
		gID = ""
	}
	return h.showParticipantsForAssign(ctx, cb, groupID(gID), userCtx.PendingNoteID)
}

func (h *NotesHandler) showParticipantsForAssign(ctx context.Context, cb *tgbotapi.CallbackQuery, gID groupID, noteID string) error {
	resp, err := h.Clients.Participants.ListParticipants(ctx, &participantsv1.ListParticipantsRequest{
		GroupId:    string(gID),
		Pagination: &commonv1.Pagination{Limit: 20},
	})
	if err != nil {
		return h.SendError(cb.Message.Chat.ID, "Не удалось загрузить участников.")
	}

	kb := keyboards.SelectParticipantForNote(resp.Participants, "", noteID)
	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, "Выберите участника:")
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
		h.AnswerCallback(cb.ID, "Ошибка назначения")
		return err
	}
	h.AnswerCallback(cb.ID, "✅ Заметка назначена")
	return h.EditMD(cb.Message.Chat.ID, cb.Message.MessageID,
		"✅ Заметка назначена участнику\\.", menuKeyboard(user))
}

// groupID is a typed string to avoid confusion between group ID and empty string.
type groupID string

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func menuKeyboard(user *usersv1.User) *tgbotapi.InlineKeyboardMarkup {
	kb := keyboards.MainMenu(user.Role)
	return &kb
}

func pageInfoFields(pi *commonv1.PageInfo) (total int32, hasNext bool) {
	if pi != nil {
		total = pi.Total
		hasNext = pi.HasNext
	}
	return
}
