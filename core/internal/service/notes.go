package service

import (
	"context"
	"fmt"

	"github.com/mrralexandrov/instructive-notes-bot/core/internal/repository"
)

// NotesService handles note business logic.
type NotesService struct {
	repo *repository.NotesRepo
}

// NewNotesService creates a new NotesService.
func NewNotesService(repo *repository.NotesRepo) *NotesService {
	return &NotesService{repo: repo}
}

// Create creates a new note.
func (s *NotesService) Create(ctx context.Context, authorID string, participantID *string, text string) (*repository.Note, error) {
	if authorID == "" {
		return nil, fmt.Errorf("author_id is required")
	}
	if text == "" {
		return nil, fmt.Errorf("text is required")
	}
	return s.repo.Create(ctx, authorID, participantID, text)
}

// GetByID returns a note by ID.
func (s *NotesService) GetByID(ctx context.Context, id string) (*repository.Note, error) {
	return s.repo.GetByID(ctx, id)
}

// List returns paginated notes with filtering.
func (s *NotesService) List(ctx context.Context, f repository.ListFilter) ([]*repository.Note, error) {
	if f.Limit <= 0 {
		f.Limit = 20
	}
	return s.repo.List(ctx, f)
}

// Update modifies an existing note.
func (s *NotesService) Update(ctx context.Context, id, text string) (*repository.Note, error) {
	if text == "" {
		return nil, fmt.Errorf("text is required")
	}
	return s.repo.Update(ctx, id, text)
}

// Delete removes a note.
func (s *NotesService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// AssignToParticipant assigns a note to a participant.
func (s *NotesService) AssignToParticipant(ctx context.Context, noteID, participantID string) (*repository.Note, error) {
	return s.repo.AssignToParticipant(ctx, noteID, participantID)
}
