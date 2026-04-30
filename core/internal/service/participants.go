package service

import (
	"context"
	"fmt"

	"github.com/mrralexandrov/instructive-notes-bot/core/internal/repository"
)

// ParticipantsService handles participant business logic.
type ParticipantsService struct {
	repo *repository.ParticipantsRepo
}

// NewParticipantsService creates a new ParticipantsService.
func NewParticipantsService(repo *repository.ParticipantsRepo) *ParticipantsService {
	return &ParticipantsService{repo: repo}
}

// Create creates a new participant.
func (s *ParticipantsService) Create(ctx context.Context, name string, telegramID *int64, telegramUsername string, customID *string, groupID *string) (*repository.Participant, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	return s.repo.Create(ctx, name, telegramID, telegramUsername, customID, groupID)
}

// GetByID returns a participant by ID.
func (s *ParticipantsService) GetByID(ctx context.Context, id string) (*repository.Participant, error) {
	return s.repo.GetByID(ctx, id)
}

// List returns paginated participants.
func (s *ParticipantsService) List(ctx context.Context, groupID, search string, limit, offset int) ([]*repository.Participant, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.List(ctx, groupID, search, limit, offset)
}

// Update modifies an existing participant.
func (s *ParticipantsService) Update(ctx context.Context, id, name string, telegramID *int64, telegramUsername string, customID *string, groupID *string) (*repository.Participant, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	return s.repo.Update(ctx, id, name, telegramID, telegramUsername, customID, groupID)
}

// Delete removes a participant.
func (s *ParticipantsService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// SetPhoto sets the photo for a participant.
func (s *ParticipantsService) SetPhoto(ctx context.Context, participantID, mediaID string) (*repository.Participant, error) {
	return s.repo.SetPhoto(ctx, participantID, mediaID)
}
