package service

import (
	"context"
	"fmt"

	"github.com/mrralexandrov/instructive-notes-bot/core/internal/repository"
)

// GroupsService handles group business logic.
type GroupsService struct {
	repo *repository.GroupsRepo
}

// NewGroupsService creates a new GroupsService.
func NewGroupsService(repo *repository.GroupsRepo) *GroupsService {
	return &GroupsService{repo: repo}
}

// Create creates a new group.
func (s *GroupsService) Create(ctx context.Context, name, description string) (*repository.Group, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	return s.repo.Create(ctx, name, description)
}

// GetByID returns a group by ID.
func (s *GroupsService) GetByID(ctx context.Context, id string) (*repository.Group, error) {
	return s.repo.GetByID(ctx, id)
}

// List returns paginated groups.
func (s *GroupsService) List(ctx context.Context, limit, offset int) ([]*repository.Group, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.List(ctx, limit, offset)
}

// Update modifies an existing group.
func (s *GroupsService) Update(ctx context.Context, id, name, description string) (*repository.Group, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	return s.repo.Update(ctx, id, name, description)
}

// Delete removes a group.
func (s *GroupsService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
