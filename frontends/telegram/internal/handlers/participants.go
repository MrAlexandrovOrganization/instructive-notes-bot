package handlers

import (
	"bytes"
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	commonv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/common/v1"
	groupsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/groups/v1"
	mediav1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/media/v1"
	participantsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/participants/v1"
	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/keyboards"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/state"
)

// ParticipantsHandler handles participant-related interactions.
type ParticipantsHandler struct {
	*Base
}

// NewParticipantsHandler creates a new ParticipantsHandler.
func NewParticipantsHandler(base *Base) *ParticipantsHandler {
	return &ParticipantsHandler{Base: base}
}

// HandleParticipantsList shows the participants list.
func (h *ParticipantsHandler) HandleParticipantsList(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User, groupID string) error {
	resp, err := h.Clients.Participants.ListParticipants(ctx, &participantsv1.ListParticipantsRequest{
		GroupId:    groupID,
		Pagination: &commonv1.Pagination{Limit: 10},
	})
	if err != nil {
		return h.sendError(msg.Chat.ID, "Не удалось загрузить участников.")
	}

	title := "👥 *Участники*"
	if groupID != "" {
		title = "👥 *Моя группа*"
	}

	nextCursor := ""
	if resp.PageInfo != nil && resp.PageInfo.HasNext {
		nextCursor = resp.PageInfo.NextCursor
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, title)
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = keyboards.ParticipantsList(resp.Participants, nextCursor)
	_, err = h.Bot.Send(reply)
	return err
}

// HandleParticipantView shows a single participant.
func (h *ParticipantsHandler) HandleParticipantView(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, participantID string) error {
	h.answerCallback(cb.ID, "")

	p, err := h.Clients.Participants.GetParticipant(ctx, &participantsv1.GetParticipantRequest{Id: participantID})
	if err != nil {
		return h.sendError(cb.Message.Chat.ID, "Участник не найден.")
	}

	text := fmt.Sprintf("👤 *%s*\n", escapeMarkdown(p.Name))
	if p.GroupName != "" {
		text += fmt.Sprintf("Группа: %s\n", escapeMarkdown(p.GroupName))
	}
	text += fmt.Sprintf("Заметок: %d", p.NotesCount)

	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, text)
	edit.ParseMode = "MarkdownV2"
	kb := keyboards.ParticipantView(participantID, user.Role)
	edit.ReplyMarkup = &kb
	_, err = h.Bot.Send(edit)
	return err
}

// HandleParticipantPhoto shows or updates the photo for a participant.
func (h *ParticipantsHandler) HandleParticipantPhoto(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, participantID string) error {
	h.answerCallback(cb.ID, "")

	p, err := h.Clients.Participants.GetParticipant(ctx, &participantsv1.GetParticipantRequest{Id: participantID})
	if err != nil {
		return h.sendError(cb.Message.Chat.ID, "Участник не найден.")
	}

	// If participant has a photo, show it.
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
	}

	// Ask to upload a new photo.
	h.States.Set(cb.From.ID, &state.UserContext{
		State:       state.StateUploadingPhoto,
		PendingData: participantID,
	})
	reply := tgbotapi.NewMessage(cb.Message.Chat.ID, "Отправьте фото для участника:")
	reply.ReplyMarkup = keyboards.CancelKeyboard()
	_, err = h.Bot.Send(reply)
	return err
}

// HandlePhotoUpload processes an uploaded photo.
func (h *ParticipantsHandler) HandlePhotoUpload(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) error {
	userCtx := h.States.Get(msg.From.ID)
	participantID := userCtx.PendingData
	h.States.Reset(msg.From.ID)

	if len(msg.Photo) == 0 {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Пожалуйста, отправьте фото.")
		reply.ReplyMarkup = keyboards.MainMenu(user.Role)
		_, err := h.Bot.Send(reply)
		return err
	}

	// Use the largest photo size.
	photo := msg.Photo[len(msg.Photo)-1]
	fileURL, err := h.Bot.GetFileDirectURL(photo.FileID)
	if err != nil {
		return h.sendError(msg.Chat.ID, "Не удалось получить файл.")
	}

	// Download the photo bytes.
	resp, err := downloadFile(fileURL)
	if err != nil {
		return h.sendError(msg.Chat.ID, "Не удалось скачать фото.")
	}

	// Upload to media service.
	mediaResp, err := h.Clients.Media.UploadMedia(ctx, &mediav1.UploadMediaRequest{
		Data:         resp,
		MimeType:     "image/jpeg",
		OriginalName: photo.FileID + ".jpg",
	})
	if err != nil {
		return h.sendError(msg.Chat.ID, "Не удалось сохранить фото.")
	}

	// Set photo on participant.
	_, err = h.Clients.Participants.SetParticipantPhoto(ctx, &participantsv1.SetParticipantPhotoRequest{
		ParticipantId: participantID,
		MediaId:       mediaResp.Id,
	})
	if err != nil {
		return h.sendError(msg.Chat.ID, "Не удалось обновить участника.")
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, "✅ Фото сохранено!")
	reply.ReplyMarkup = keyboards.MainMenu(user.Role)
	_, err = h.Bot.Send(reply)
	return err
}

// HandleAddParticipantName starts the add-participant flow by asking for name.
func (h *ParticipantsHandler) HandleAddParticipantName(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) error {
	h.States.SetState(msg.From.ID, state.StateAddingParticipantName)
	reply := tgbotapi.NewMessage(msg.Chat.ID, "Введите имя участника:")
	reply.ReplyMarkup = keyboards.CancelKeyboard()
	_, err := h.Bot.Send(reply)
	return err
}

// HandleParticipantNameInput handles the name input during participant creation.
func (h *ParticipantsHandler) HandleParticipantNameInput(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User) error {
	name := msg.Text
	h.States.Set(msg.From.ID, &state.UserContext{
		State:       state.StateAddingParticipantGroup,
		PendingData: name,
	})

	// Show groups to choose from.
	resp, err := h.Clients.Groups.ListGroups(ctx, &groupsv1.ListGroupsRequest{})
	if err != nil || len(resp.Groups) == 0 {
		// Create without group.
		return h.createParticipant(ctx, msg, user, name, "")
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Без группы", "participant:group:none"),
	))
	for _, g := range resp.Groups {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(g.Name, "participant:group:"+g.Id),
		))
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, "Выберите группу:")
	reply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	_, err = h.Bot.Send(reply)
	return err
}

// HandleParticipantGroupSelect handles group selection during participant creation.
func (h *ParticipantsHandler) HandleParticipantGroupSelect(ctx context.Context, cb *tgbotapi.CallbackQuery, user *usersv1.User, groupID string) error {
	h.answerCallback(cb.ID, "")
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
		return h.sendError(cb.Message.Chat.ID, "Не удалось создать участника.")
	}

	text := fmt.Sprintf("✅ Участник *%s* создан!", escapeMarkdown(p.Name))
	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, text)
	edit.ParseMode = "MarkdownV2"
	_, err = h.Bot.Send(edit)
	return err
}

func (h *ParticipantsHandler) createParticipant(ctx context.Context, msg *tgbotapi.Message, user *usersv1.User, name, groupID string) error {
	h.States.Reset(msg.From.ID)
	p, err := h.Clients.Participants.CreateParticipant(ctx, &participantsv1.CreateParticipantRequest{
		Name:    name,
		GroupId: groupID,
	})
	if err != nil {
		return h.sendError(msg.Chat.ID, "Не удалось создать участника.")
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ Участник *%s* создан!", escapeMarkdown(p.Name)))
	reply.ParseMode = "MarkdownV2"
	reply.ReplyMarkup = keyboards.MainMenu(user.Role)
	_, err = h.Bot.Send(reply)
	return err
}

func (h *ParticipantsHandler) sendError(chatID int64, text string) error {
	_, err := h.Bot.Send(tgbotapi.NewMessage(chatID, "❌ "+text))
	return err
}

func (h *ParticipantsHandler) answerCallback(callbackID, text string) {
	_, _ = h.Bot.Request(tgbotapi.NewCallback(callbackID, text))
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
