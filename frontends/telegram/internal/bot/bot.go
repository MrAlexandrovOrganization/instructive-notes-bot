package bot

import (
	"context"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/client"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/handlers"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/middleware"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/state"
)

// Bot is the main bot dispatcher.
type Bot struct {
	api          *tgbotapi.BotAPI
	auth         *middleware.AuthMiddleware
	states       *state.Manager
	startHandler *handlers.StartHandler
	notesHandler *handlers.NotesHandler
	partHandler  *handlers.ParticipantsHandler
	adminHandler *handlers.AdminHandler
}

// New creates a new Bot instance.
func New(token string, clients *client.Clients, rootTelegramID int64) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	states := state.NewManager()
	base := handlers.NewBase(api, clients, states)

	return &Bot{
		api:          api,
		auth:         middleware.NewAuthMiddleware(clients, rootTelegramID),
		states:       states,
		startHandler: handlers.NewStartHandler(base),
		notesHandler: handlers.NewNotesHandler(base),
		partHandler:  handlers.NewParticipantsHandler(base),
		adminHandler: handlers.NewAdminHandler(base),
	}, nil
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

	// Handle cancel.
	if msg.Text == "❌ Отмена" {
		if err := b.startHandler.HandleCancel(ctx, msg, user); err != nil {
			slog.Error("handle cancel", "error", err)
		}
		return
	}

	// Handle state-driven input first.
	switch userCtx.State {
	case state.StateWritingNoteText:
		if err := b.notesHandler.HandleNoteText(ctx, msg, user); err != nil {
			slog.Error("handle note text", "error", err)
		}
		return
	case state.StateUploadingPhoto:
		if err := b.partHandler.HandlePhotoUpload(ctx, msg, user); err != nil {
			slog.Error("handle photo upload", "error", err)
		}
		return
	case state.StateAddingParticipantName:
		if err := b.partHandler.HandleParticipantNameInput(ctx, msg, user); err != nil {
			slog.Error("handle participant name", "error", err)
		}
		return
	case state.StateAddingUserName:
		if err := b.adminHandler.HandleUserNameInput(ctx, msg, user); err != nil {
			slog.Error("handle user name", "error", err)
		}
		return
	case state.StateAddingUserTelegramID:
		if err := b.adminHandler.HandleUserTelegramIDInput(ctx, msg, user); err != nil {
			slog.Error("handle user telegram id", "error", err)
		}
		return
	case state.StateAddingGroupName:
		if err := b.adminHandler.HandleGroupNameInput(ctx, msg, user); err != nil {
			slog.Error("handle group name", "error", err)
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

	// Handle keyboard buttons.
	b.handleKeyboardText(ctx, msg, user)
}

func (b *Bot) handleKeyboardText(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) {
	switch msg.Text {
	case "📝 Новая заметка":
		if err := b.notesHandler.HandleNewNote(ctx, msg, user); err != nil {
			slog.Error("handle new note", "error", err)
		}
	case "📋 Мои заметки":
		if err := b.notesHandler.HandleMyNotes(ctx, msg, user); err != nil {
			slog.Error("handle my notes", "error", err)
		}
	case "📋 Все заметки":
		if isAdminOrRoot(user) {
			if err := b.notesHandler.HandleAllNotes(ctx, msg, user); err != nil {
				slog.Error("handle all notes", "error", err)
			}
		}
	case "👥 Участники", "🔍 Все участники":
		if err := b.partHandler.HandleParticipantsList(ctx, msg, user, ""); err != nil {
			slog.Error("handle participants list", "error", err)
		}
	case "👥 Моя группа":
		groupID := ""
		if user.GroupId != "" {
			groupID = user.GroupId
		}
		if err := b.partHandler.HandleParticipantsList(ctx, msg, user, groupID); err != nil {
			slog.Error("handle my group", "error", err)
		}
	case "⚙️ Управление":
		if isAdminOrRoot(user) {
			if err := b.adminHandler.HandleAdminPanel(ctx, msg, user); err != nil {
				slog.Error("handle admin panel", "error", err)
			}
		}
	default:
		if err := b.startHandler.HandleUnknownState(ctx, msg, user); err != nil {
			slog.Error("handle unknown text", "error", err)
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
	case "participant":
		b.handleParticipantCallback(ctx, cb, user, parts)
	case "note":
		b.handleNoteCallback(ctx, cb, user, parts)
	case "notes":
		b.handleNotesCallback(ctx, cb, user, parts)
	case "admin":
		b.handleAdminCallback(ctx, cb, user, parts)
	case "user":
		b.handleUserCallback(ctx, cb, user, parts)
	case "page":
		b.handlePageCallback(ctx, cb, user, parts)
	}
}

func (b *Bot) handleParticipantCallback(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, parts []string) {
	if len(parts) < 3 {
		return
	}
	action := parts[1]
	id := parts[2]

	switch action {
	case "view":
		if err := b.partHandler.HandleParticipantView(ctx, cb, user, id); err != nil {
			slog.Error("participant view", "error", err)
		}
	case "note":
		// Start note creation pre-selected for this participant.
		userCtx := b.states.Get(cb.From.ID)
		userCtx.State = state.StateWritingNoteText
		userCtx.PendingData = id
		b.states.Set(cb.From.ID, userCtx)
		if _, err := b.api.Request(tgbotapi.NewCallback(cb.ID, "")); err != nil {
			slog.Error("answer callback", "error", err)
		}
		_, _ = b.api.Send(tgbotapi.NewMessage(cb.Message.Chat.ID, "✍️ Напишите текст заметки:"))
	case "photo":
		if err := b.partHandler.HandleParticipantPhoto(ctx, cb, user, id); err != nil {
			slog.Error("participant photo", "error", err)
		}
	case "select":
		userCtx := b.states.Get(cb.From.ID)
		if userCtx.State == state.StateAssigningNoteToParticipant {
			if err := b.notesHandler.HandleNoteAssignParticipant(ctx, cb, user, id); err != nil {
				slog.Error("assign note participant", "error", err)
			}
		} else {
			if err := b.notesHandler.HandleParticipantSelectedForNote(ctx, cb, user, id); err != nil {
				slog.Error("select participant for note", "error", err)
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
	// user:role:{id}:{role}
	if len(parts) >= 4 && parts[1] == "role" {
		if err := b.adminHandler.HandleUserRoleUpdate(ctx, cb, user, parts[2], parts[3]); err != nil {
			slog.Error("user role update", "error", err)
		}
	}
}

func (b *Bot) handlePageCallback(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, parts []string) {
	// Pagination: page:{section}:{cursor}
	// For now, just acknowledge.
	if _, err := b.api.Request(tgbotapi.NewCallback(cb.ID, "")); err != nil {
		slog.Error("answer callback", "error", err)
	}
}

func isAdminOrRoot(user *usersv1.User) bool {
	return user.Role == usersv1.Role_ROLE_ADMIN || user.Role == usersv1.Role_ROLE_ROOT
}
