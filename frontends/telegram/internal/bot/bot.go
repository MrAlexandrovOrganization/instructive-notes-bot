package bot

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/client"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/handlers"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/keyboards"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/middleware"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/state"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/whisper"
	commonv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/common/v1"
	groupsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/groups/v1"
	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
)

// msgHandler is the signature shared by all state-driven message handlers.
type msgHandler func(context.Context, *tgbotapi.Message, *usersv1.User) error

// Bot is the main bot dispatcher.
type Bot struct {
	api                      *tgbotapi.BotAPI
	clients                  *client.Clients
	auth                     *middleware.AuthMiddleware
	states                   *state.Manager
	startHandler             *handlers.StartHandler
	notesHandler             *handlers.NotesHandler
	partHandler              *handlers.ParticipantsHandler
	adminHandler             *handlers.AdminHandler
	stateHandlers            map[state.UserState]msgHandler
	createNoteOnVoiceMessage bool
}

// New creates a new Bot instance.
func New(token string, clients *client.Clients, rootTelegramID int64, wc *whisper.Client, createNoteOnVoice bool) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	states := state.NewManager()
	base := handlers.NewBase(api, clients, states, wc)

	nh := handlers.NewNotesHandler(base)
	ph := handlers.NewParticipantsHandler(base)
	ah := handlers.NewAdminHandler(base)

	b := &Bot{
		api:                      api,
		clients:                  clients,
		auth:                     middleware.NewAuthMiddleware(clients, rootTelegramID),
		states:                   states,
		startHandler:             handlers.NewStartHandler(base),
		notesHandler:             nh,
		partHandler:              ph,
		adminHandler:             ah,
		createNoteOnVoiceMessage: createNoteOnVoice,
	}
	b.stateHandlers = map[state.UserState]msgHandler{
		state.StateWritingNoteText:       nh.HandleNoteText,
		state.StateUploadingPhoto:        ph.HandlePhotoUpload,
		state.StateAddingParticipantName: ph.HandleParticipantNameInput,
		state.StateAddingUserName:        ah.HandleUserNameInput,
		state.StateAddingUserTelegramID:  ah.HandleUserTelegramIDInput,
		state.StateAddingGroupName:       ah.HandleGroupNameInput,
	}
	return b, nil
}

// Run starts processing updates.
func (b *Bot) Run(ctx context.Context) {
	slog.Info("bot started", "username", b.api.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			b.api.StopReceivingUpdates()
			return
		case update := <-updates:
			go b.handleUpdate(ctx, update)
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic in update handler", "panic", r)
		}
	}()

	var from *tgbotapi.User
	var chatID int64

	if update.Message != nil {
		from = update.Message.From
		chatID = update.Message.Chat.ID
	} else if update.CallbackQuery != nil {
		from = update.CallbackQuery.From
		chatID = update.CallbackQuery.Message.Chat.ID
	} else {
		return
	}

	// Resolve user.
	user, err := b.auth.ResolveUser(ctx, from)
	if err != nil {
		slog.Error("resolve user", "error", err)
		return
	}
	if user == nil {
		handlers.HandleNoAccess(b.api, chatID)
		return
	}

	if update.Message != nil {
		b.handleMessage(ctx, update.Message, user)
	} else if update.CallbackQuery != nil {
		b.handleCallback(ctx, update.CallbackQuery, user)
	}
}

func (b *Bot) handleMessage(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) {
	userCtx := b.states.Get(msg.From.ID)

	// Handle state-driven input first.
	if handler, ok := b.stateHandlers[userCtx.State]; ok {
		if err := handler(ctx, msg, user); err != nil {
			slog.Error("handle state input", "state", userCtx.State, "error", err)
		}
		return
	}

	// Handle commands.
	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			if err := b.startHandler.HandleStart(ctx, msg, user); err != nil {
				slog.Error("handle start", "error", err)
			}
		case "help":
			if err := b.startHandler.HandleHelp(ctx, msg, user); err != nil {
				slog.Error("handle help", "error", err)
			}
		default:
			if err := b.startHandler.HandleUnknownState(ctx, msg, user); err != nil {
				slog.Error("handle unknown command", "error", err)
			}
		}
		return
	}

	// Voice messages and video notes → always transcribe, optionally save as note.
	if msg.Voice != nil || msg.VideoNote != nil {
		if err := b.notesHandler.HandleVoiceNote(ctx, msg, user, b.createNoteOnVoiceMessage); err != nil {
			slog.Error("handle voice note", "error", err)
		}
		return
	}

	// Any free text without an active state is saved as a quick note.
	if msg.Text != "" {
		if err := b.notesHandler.HandleQuickNote(ctx, msg, user); err != nil {
			slog.Error("handle quick note", "error", err)
		}
	}
}

func (b *Bot) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User) {
	data := cb.Data
	parts := strings.SplitN(data, ":", 4)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "cancel":
		b.states.Reset(cb.From.ID)
		b.AnswerCallback(cb.ID, "Отменено")
		kb := keyboards.MainMenu(user.Role)
		_ = b.startHandler.Base.EditMD(cb.Message.Chat.ID, cb.Message.MessageID, "📋 *Главное меню*", &kb)
	case "menu":
		b.handleMenuCallback(ctx, cb, user, parts)
	case "back":
		b.handleBackCallback(ctx, cb, user, parts)
	case "participant":
		b.handleParticipantCallback(ctx, cb, user, parts)
	case "note":
		b.handleNoteCallback(ctx, cb, user, parts)
	case "notes":
		b.handleNotesCallback(ctx, cb, user, parts)
	case "group":
		b.handleGroupCallback(ctx, cb, user, parts)
	case "admin":
		b.handleAdminCallback(ctx, cb, user, parts)
	case "user":
		b.handleUserCallback(ctx, cb, user, parts)
	case "page":
		b.handlePageCallback(ctx, cb, user, parts)
	}
}

func (b *Bot) handleMenuCallback(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, parts []string) {
	if len(parts) < 2 {
		return
	}
	b.AnswerCallback(cb.ID, "")
	switch parts[1] {
	case "notes":
		if err := b.notesHandler.HandleMyNotes(ctx, cb, user); err != nil {
			slog.Error("handle my notes", "error", err)
		}
	case "all_notes":
		if isAdminOrRoot(user) {
			if err := b.notesHandler.HandleAllNotes(ctx, cb, user); err != nil {
				slog.Error("handle all notes", "error", err)
			}
		}
	case "participants":
		if err := b.partHandler.HandleParticipantsList(ctx, cb, user, "", "back:menu"); err != nil {
			slog.Error("handle participants list", "error", err)
		}
	case "my_group":
		if err := b.partHandler.HandleParticipantsList(ctx, cb, user, user.GroupId, "back:menu"); err != nil {
			slog.Error("handle my group", "error", err)
		}
	case "groups":
		b.handleBrowseGroups(ctx, cb, user)
	case "admin":
		if isAdminOrRoot(user) {
			if err := b.adminHandler.HandleAdminPanel(ctx, cb, user); err != nil {
				slog.Error("handle admin panel", "error", err)
			}
		}
	}
}

func (b *Bot) handleBackCallback(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, parts []string) {
	if len(parts) < 2 {
		return
	}
	b.AnswerCallback(cb.ID, "")
	switch parts[1] {
	case "menu":
		kb := keyboards.MainMenu(user.Role)
		_ = b.startHandler.Base.EditMD(cb.Message.Chat.ID, cb.Message.MessageID, "📋 *Главное меню*", &kb)
	case "notes":
		if isAdminOrRoot(user) {
			if err := b.notesHandler.HandleAllNotes(ctx, cb, user); err != nil {
				slog.Error("back all notes", "error", err)
			}
		} else {
			if err := b.notesHandler.HandleMyNotes(ctx, cb, user); err != nil {
				slog.Error("back my notes", "error", err)
			}
		}
	case "notes_list":
		// Return to the current notes list based on saved context.
		if err := b.notesHandler.HandleNotesPage(ctx, cb, user, 0); err != nil {
			slog.Error("back notes list", "error", err)
		}
	case "participants":
		if err := b.partHandler.HandleParticipantsList(ctx, cb, user, "", "back:menu"); err != nil {
			slog.Error("back participants list", "error", err)
		}
	case "participants_list":
		// Return to the current participants list based on saved context (page 0).
		if err := b.partHandler.HandleParticipantsPage(ctx, cb, user, 0); err != nil {
			slog.Error("back participants list page", "error", err)
		}
	case "participant":
		if len(parts) >= 3 {
			if err := b.partHandler.HandleParticipantView(ctx, cb, user, parts[2]); err != nil {
				slog.Error("back participant view", "error", err)
			}
		}
	case "admin":
		if isAdminOrRoot(user) {
			if err := b.adminHandler.HandleAdminPanel(ctx, cb, user); err != nil {
				slog.Error("back admin panel", "error", err)
			}
		}
	case "groups":
		b.handleBrowseGroups(ctx, cb, user)
	case "group_view":
		// back:group_view:{groupID} — return to participants of a specific group
		if len(parts) >= 3 {
			if err := b.partHandler.HandleParticipantsList(ctx, cb, user, parts[2], "back:groups"); err != nil {
				slog.Error("back group view", "error", err)
			}
		}
	}
}

func (b *Bot) handleParticipantCallback(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, parts []string) {
	if len(parts) < 2 {
		return
	}
	action := parts[1]

	if action == "add" {
		if err := b.partHandler.HandleAddParticipantStart(ctx, cb, user); err != nil {
			slog.Error("add participant start", "error", err)
		}
		return
	}

	if len(parts) < 3 {
		return
	}
	id := parts[2]

	switch action {
	case "view":
		if err := b.partHandler.HandleParticipantView(ctx, cb, user, id); err != nil {
			slog.Error("participant view", "error", err)
		}
	case "note":
		// Start note creation pre-selected for this participant.
		b.states.Set(cb.From.ID, &state.UserContext{
			State:       state.StateWritingNoteText,
			PendingData: id,
		})
		b.AnswerCallback(cb.ID, "")
		kb := keyboards.CancelInline()
		edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, "✍️ Напишите текст заметки:")
		edit.ReplyMarkup = &kb
		_, _ = b.api.Send(edit)
	case "photo":
		if err := b.partHandler.HandleParticipantPhoto(ctx, cb, user, id); err != nil {
			slog.Error("participant photo", "error", err)
		}
	case "update_photo":
		if err := b.partHandler.HandleParticipantUpdatePhoto(ctx, cb, user, id); err != nil {
			slog.Error("participant update photo", "error", err)
		}
	case "select":
		userCtx := b.states.Get(cb.From.ID)
		if userCtx.State == state.StateAssigningNoteToParticipant {
			if err := b.notesHandler.HandleNoteAssignParticipant(ctx, cb, user, id); err != nil {
				slog.Error("assign note participant", "error", err)
			}
		}
	case "group":
		if err := b.partHandler.HandleParticipantGroupSelect(ctx, cb, user, id); err != nil {
			slog.Error("participant group select", "error", err)
		}
	}
}

func (b *Bot) handleNoteCallback(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, parts []string) {
	if len(parts) < 3 {
		return
	}
	action := parts[1]
	id := parts[2]

	switch action {
	case "view":
		if err := b.notesHandler.HandleNoteView(ctx, cb, user, id); err != nil {
			slog.Error("note view", "error", err)
		}
	case "delete":
		if err := b.notesHandler.HandleNoteDelete(ctx, cb, user, id); err != nil {
			slog.Error("note delete", "error", err)
		}
	case "assign":
		if err := b.notesHandler.HandleNoteAssignStart(ctx, cb, user, id); err != nil {
			slog.Error("note assign start", "error", err)
		}
	}
}

func (b *Bot) handleNotesCallback(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, parts []string) {
	if len(parts) < 2 {
		return
	}
	switch parts[1] {
	case "unassigned":
		if err := b.notesHandler.HandleUnassignedNotes(ctx, cb, user); err != nil {
			slog.Error("unassigned notes", "error", err)
		}
	case "participant":
		if len(parts) >= 3 {
			if err := b.notesHandler.HandleNotesByParticipant(ctx, cb, user, parts[2]); err != nil {
				slog.Error("notes by participant", "error", err)
			}
		}
	}
}

// handleGroupCallback handles group:for_note:{groupID} and group:view:{groupID} callbacks.
func (b *Bot) handleGroupCallback(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, parts []string) {
	if len(parts) < 3 {
		return
	}
	switch parts[1] {
	case "for_note":
		if err := b.notesHandler.HandleGroupForNote(ctx, cb, user, parts[2]); err != nil {
			slog.Error("group for note", "error", err)
		}
	case "view":
		// Show participants of a group.
		backTo := "back:groups"
		if err := b.partHandler.HandleParticipantsList(ctx, cb, user, parts[2], backTo); err != nil {
			slog.Error("group view participants", "error", err)
		}
	}
}

func (b *Bot) handleAdminCallback(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, parts []string) {
	if !isAdminOrRoot(user) {
		if _, err := b.api.Request(tgbotapi.NewCallback(cb.ID, "Нет доступа")); err != nil {
			slog.Error("answer callback", "error", err)
		}
		return
	}
	if len(parts) < 2 {
		return
	}
	if err := b.adminHandler.HandleAdminCallback(ctx, cb, user, parts[1]); err != nil {
		slog.Error("admin callback", "error", err)
	}
}

func (b *Bot) handleUserCallback(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, parts []string) {
	if !isAdminOrRoot(user) {
		return
	}
	switch parts[1] {
	case "role":
		// user:role:{id}:{role}
		if len(parts) >= 4 {
			if err := b.adminHandler.HandleUserRoleUpdate(ctx, cb, user, parts[2], parts[3]); err != nil {
				slog.Error("user role update", "error", err)
			}
		}
	case "create_role":
		// user:create_role:{role}
		if len(parts) >= 3 {
			if err := b.adminHandler.HandleUserCreateRole(ctx, cb, user, parts[2]); err != nil {
				slog.Error("user create role", "error", err)
			}
		}
	}
}

// handlePageCallback handles pagination callbacks.
// Format: page:notes:{offset}
func (b *Bot) handlePageCallback(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, parts []string) {
	if len(parts) < 3 {
		b.AnswerCallback(cb.ID, "")
		return
	}

	offset, err := strconv.Atoi(parts[2])
	if err != nil {
		b.AnswerCallback(cb.ID, "")
		return
	}

	switch parts[1] {
	case "notes":
		if err := b.notesHandler.HandleNotesPage(ctx, cb, user, int32(offset)); err != nil {
			slog.Error("notes page", "error", err)
		}
	case "participants":
		if err := b.partHandler.HandleParticipantsPage(ctx, cb, user, int32(offset)); err != nil {
			slog.Error("participants page", "error", err)
		}
	case "assign_participant":
		if err := b.notesHandler.HandleAssignParticipantPage(ctx, cb, user, int32(offset)); err != nil {
			slog.Error("assign participant page", "error", err)
		}
	case "users":
		if isAdminOrRoot(user) {
			if err := b.adminHandler.HandleUsersPage(ctx, cb, int32(offset)); err != nil {
				slog.Error("users page", "error", err)
			}
		}
	default:
		b.AnswerCallback(cb.ID, "")
	}
}

// handleBrowseGroups shows groups for browsing participants.
func (b *Bot) handleBrowseGroups(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User) {
	resp, err := b.clients.Groups.ListGroups(ctx, &groupsv1.ListGroupsRequest{
		Pagination: &commonv1.Pagination{Limit: 50},
	})
	if err != nil {
		slog.Error("browse groups", "error", err)
		return
	}

	kb := keyboards.GroupsListForBrowse(resp.Groups, user.Role)
	_ = b.startHandler.Base.EditMD(cb.Message.Chat.ID, cb.Message.MessageID, "🏷 *Отряды*", &kb)
}

func isAdminOrRoot(user *usersv1.User) bool {
	return user.Role == usersv1.Role_ROLE_ADMIN || user.Role == usersv1.Role_ROLE_ROOT
}

// AnswerCallback answers a callback query with an optional toast message.
func (b *Bot) AnswerCallback(id, text string) {
	_, _ = b.api.Request(tgbotapi.NewCallback(id, text))
}
