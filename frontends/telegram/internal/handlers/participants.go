package handlers

import (
	"bytes"
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/keyboards"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/state"
	commonv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/common/v1"
	groupsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/groups/v1"
	mediav1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/media/v1"
	participantsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/participants/v1"
	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
)

// ParticipantsHandler handles participant-related interactions.
type ParticipantsHandler struct {
	*Base
}

// NewParticipantsHandler creates a new ParticipantsHandler.
func NewParticipantsHandler(base *Base) *ParticipantsHandler {
	return &ParticipantsHandler{Base: base}
}

const participantsPageSize = 10

// HandleParticipantsList shows the participants list by editing the current message.
func (h *ParticipantsHandler) HandleParticipantsList(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, groupID, backTo string) error {
	// Save pagination context.
	h.States.Set(cb.From.ID, &state.UserContext{
		ParticipantsGroupID: groupID,
		ParticipantsBackTo:  backTo,
	})
	return h.renderParticipantsPage(ctx, cb, user, groupID, backTo, 0)
}

// HandleParticipantsPage handles offset-based pagination for participants.
func (h *ParticipantsHandler) HandleParticipantsPage(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, offset int32) error {
	h.AnswerCallback(cb.ID, "")
	userCtx := h.States.Get(cb.From.ID)
	return h.renderParticipantsPage(ctx, cb, user, userCtx.ParticipantsGroupID, userCtx.ParticipantsBackTo, offset)
}

func (h *ParticipantsHandler) renderParticipantsPage(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, groupID, backTo string, offset int32) error {
	resp, err := h.Clients.Participants.ListParticipants(ctx, &participantsv1.ListParticipantsRequest{
		GroupId:    groupID,
		Pagination: &commonv1.Pagination{Limit: participantsPageSize, Offset: offset},
	})
	if err != nil {
		return h.SendError(cb.Message.Chat.ID, "Не удалось загрузить участников.")
	}

	title := "👥 *Участники*"
	if groupID != "" {
		title = "👥 *Мой отряд*"
	}
	if len(resp.Participants) == 0 && offset == 0 {
		if groupID != "" {
			title = "👥 В вашем отряде нет участников\\."
		} else {
			title = "👥 Участников пока нет\\."
		}
	}

	hasNext := resp.PageInfo != nil && resp.PageInfo.HasNext

	kb := keyboards.ParticipantsList(keyboards.ParticipantsListOpts{
		Participants: resp.Participants,
		Role:         user.Role,
		BackTo:       backTo,
		Offset:       offset,
		HasNext:      hasNext,
		PageSize:     participantsPageSize,
	})
	return h.EditMD(cb.Message.Chat.ID, cb.Message.MessageID, title, &kb)
}

// HandleParticipantView shows a single participant.
func (h *ParticipantsHandler) HandleParticipantView(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, participantID string) error {
	h.AnswerCallback(cb.ID, "")

	p, err := h.Clients.Participants.GetParticipant(ctx, &participantsv1.GetParticipantRequest{Id: participantID})
	if err != nil {
		return h.SendError(cb.Message.Chat.ID, "Участник не найден.")
	}

	text := fmt.Sprintf("👤 *%s*\n", EscapeMarkdown(p.Name))
	if p.TelegramUsername != "" {
		text += fmt.Sprintf("Telegram: @%s\n", EscapeMarkdown(p.TelegramUsername))
	}
	if p.GroupName != "" {
		text += fmt.Sprintf("Отряд: %s\n", EscapeMarkdown(p.GroupName))
	}
	text += fmt.Sprintf("Заметок: %d", p.NotesCount)

	kb := keyboards.ParticipantView(participantID, user.Role)
	return h.EditMD(cb.Message.Chat.ID, cb.Message.MessageID, text, &kb)
}

// HandleParticipantPhoto shows the photo if it exists, or asks to upload one.
func (h *ParticipantsHandler) HandleParticipantPhoto(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, participantID string) error {
	h.AnswerCallback(cb.ID, "")

	p, err := h.Clients.Participants.GetParticipant(ctx, &participantsv1.GetParticipantRequest{Id: participantID})
	if err != nil {
		return h.SendError(cb.Message.Chat.ID, "Участник не найден.")
	}

	if p.PhotoMediaId != "" {
		mediaResp, err := h.Clients.Media.GetMedia(ctx, &mediav1.GetMediaRequest{Id: p.PhotoMediaId})
		if err == nil {
			photo := tgbotapi.NewPhoto(cb.Message.Chat.ID, tgbotapi.FileBytes{
				Name:  "photo.jpg",
				Bytes: mediaResp.Data,
			})
			photo.Caption = "📸 Фото: " + p.Name
			_, _ = h.Bot.Send(photo)
		}
		// Show options: update photo or go back.
		kb := keyboards.ParticipantPhotoView(participantID)
		return h.SendPlain(cb.Message.Chat.ID, "Выберите действие:", kb)
	}

	// No photo — ask to upload (edit the existing message).
	h.States.Set(cb.From.ID, &state.UserContext{
		State:       state.StateUploadingPhoto,
		PendingData: participantID,
	})
	kb := keyboards.CancelInline()
	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, "Отправьте фото для участника:")
	edit.ReplyMarkup = &kb
	_, err = h.Bot.Send(edit)
	return err
}

// HandleParticipantUpdatePhoto starts the photo upload flow (from the "Обновить фото" button).
func (h *ParticipantsHandler) HandleParticipantUpdatePhoto(ctx context.Context, cb *tgbotapi.CallbackQuery, _ *usersv1.User, participantID string) error {
	h.AnswerCallback(cb.ID, "")
	h.States.Set(cb.From.ID, &state.UserContext{
		State:       state.StateUploadingPhoto,
		PendingData: participantID,
	})
	kb := keyboards.CancelInline()
	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, "Отправьте фото для участника:")
	edit.ReplyMarkup = &kb
	_, err := h.Bot.Send(edit)
	return err
}

// HandlePhotoUpload processes an uploaded photo.
func (h *ParticipantsHandler) HandlePhotoUpload(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) error {
	userCtx := h.States.Get(msg.From.ID)
	participantID := userCtx.PendingData
	h.States.Reset(msg.From.ID)

	if len(msg.Photo) == 0 {
		return h.SendPlain(msg.Chat.ID, "Пожалуйста, отправьте фото.", keyboards.MainMenu(user.Role))
	}

	photo := msg.Photo[len(msg.Photo)-1]
	fileURL, err := h.Bot.GetFileDirectURL(photo.FileID)
	if err != nil {
		return h.SendError(msg.Chat.ID, "Не удалось получить файл.")
	}

	resp, err := downloadFile(fileURL)
	if err != nil {
		return h.SendError(msg.Chat.ID, "Не удалось скачать фото.")
	}

	mediaResp, err := h.Clients.Media.UploadMedia(ctx, &mediav1.UploadMediaRequest{
		Data:         resp,
		MimeType:     "image/jpeg",
		OriginalName: photo.FileID + ".jpg",
	})
	if err != nil {
		return h.SendError(msg.Chat.ID, "Не удалось сохранить фото.")
	}

	_, err = h.Clients.Participants.SetParticipantPhoto(ctx, &participantsv1.SetParticipantPhotoRequest{
		ParticipantId: participantID,
		MediaId:       mediaResp.Id,
	})
	if err != nil {
		return h.SendError(msg.Chat.ID, "Не удалось обновить участника.")
	}

	return h.SendPlain(msg.Chat.ID, "✅ Фото сохранено!", keyboards.MainMenu(user.Role))
}

// HandleAddParticipantName starts the add-participant flow by asking for name.
func (h *ParticipantsHandler) HandleAddParticipantName(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) error {
	h.States.SetState(msg.From.ID, state.StateAddingParticipantName)
	return h.SendPlain(msg.Chat.ID, "Введите имя участника:", keyboards.CancelInline())
}

// HandleParticipantNameInput handles the name input during participant creation.
func (h *ParticipantsHandler) HandleParticipantNameInput(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) error {
	name := msg.Text
	h.States.Set(msg.From.ID, &state.UserContext{
		State:       state.StateAddingParticipantGroup,
		PendingData: name,
	})

	resp, err := h.Clients.Groups.ListGroups(ctx, &groupsv1.ListGroupsRequest{})
	if err != nil || len(resp.Groups) == 0 {
		return h.createParticipant(ctx, msg, user, name, "")
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Без отряда", "participant:group:none"),
	))
	for _, g := range resp.Groups {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(g.Name, "participant:group:"+g.Id),
		))
	}

	return h.SendPlain(msg.Chat.ID, "Выберите отряд:", tgbotapi.NewInlineKeyboardMarkup(rows...))
}

// HandleParticipantGroupSelect handles group selection during participant creation.
func (h *ParticipantsHandler) HandleParticipantGroupSelect(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, groupID string) error {
	h.AnswerCallback(cb.ID, "")
	userCtx := h.States.Get(cb.From.ID)
	name := userCtx.PendingData
	h.States.Reset(cb.From.ID)

	if groupID == "none" {
		groupID = ""
	}

	p, err := h.Clients.Participants.CreateParticipant(ctx, &participantsv1.CreateParticipantRequest{
		Name:    name,
		GroupId: groupID,
	})
	if err != nil {
		return h.SendError(cb.Message.Chat.ID, "Не удалось создать участника.")
	}

	text := fmt.Sprintf("✅ Участник *%s* создан\\!", EscapeMarkdown(p.Name))
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👤 Профиль участника", "participant:view:"+p.Id),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👥 Участники", "menu:participants"),
		),
	)
	return h.EditMD(cb.Message.Chat.ID, cb.Message.MessageID, text, &kb)
}

// HandleAddParticipantStart initiates participant creation from a callback.
func (h *ParticipantsHandler) HandleAddParticipantStart(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User) error {
	h.AnswerCallback(cb.ID, "")
	h.States.SetState(cb.From.ID, state.StateAddingParticipantName)
	kb := keyboards.CancelInline()
	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, "Введите имя участника:")
	edit.ReplyMarkup = &kb
	_, err := h.Bot.Send(edit)
	return err
}

func (h *ParticipantsHandler) createParticipant(ctx context.Context, msg *tgbotapi.Message, _ *usersv1.User, name, groupID string) error {
	h.States.Reset(msg.From.ID)
	p, err := h.Clients.Participants.CreateParticipant(ctx, &participantsv1.CreateParticipantRequest{
		Name:    name,
		GroupId: groupID,
	})
	if err != nil {
		return h.SendError(msg.Chat.ID, "Не удалось создать участника.")
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👤 Профиль участника", "participant:view:"+p.Id),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👥 Участники", "menu:participants"),
		),
	)
	text := fmt.Sprintf("✅ Участник *%s* создан\\!", EscapeMarkdown(p.Name))
	return h.SendMD(msg.Chat.ID, text, kb)
}

func downloadFile(url string) ([]byte, error) {
	//nolint:noctx
	resp, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	buf := &bytes.Buffer{}
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
