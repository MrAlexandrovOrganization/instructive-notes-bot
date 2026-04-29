package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/mrralexandrov/instructive-notes-bot/core/internal/repository"
)

// UsersService handles user business logic.
type UsersService struct {
	repo *repository.UsersRepo
}

// NewUsersService creates a new UsersService.
func NewUsersService(repo *repository.UsersRepo) *UsersService {
	return &UsersService{repo: repo}
}

// GetOrCreate returns an existing user or creates one with the given role.
func (s *UsersService) GetOrCreate(ctx context.Context, telegramID int64, name, username, defaultRole string) (*repository.User, bool, error) {
	u, err := s.repo.GetByTelegramID(ctx, telegramID)
	if err == nil {
		return u, false, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, false, fmt.Errorf("get user: %w", err)
	}
	created, err := s.repo.CreateUser(ctx, telegramID, name, username, defaultRole)
	if err != nil {
		return nil, false, fmt.Errorf("create user: %w", err)
	}
	slog.Info("user created", "user_id", created.ID, "telegram_id", telegramID, "name", name, "role", defaultRole)
	return created, true, nil
}

// GetByID returns a user by internal ID.
func (s *UsersService) GetByID(ctx context.Context, id string) (*repository.User, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByTelegramID returns a user by Telegram ID.
func (s *UsersService) GetByTelegramID(ctx context.Context, telegramID int64) (*repository.User, error) {
	return s.repo.GetByTelegramID(ctx, telegramID)
}

// List returns paginated users.
func (s *UsersService) List(ctx context.Context, roleFilter string, limit int, cursor string) ([]*repository.User, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.ListUsers(ctx, roleFilter, limit, cursor)
}

// UpdateRole changes the role of a user.
func (s *UsersService) UpdateRole(ctx context.Context, id, role string) (*repository.User, error) {
	validRoles := map[string]bool{"organizer": true, "curator": true, "admin": true, "root": true}
	if !validRoles[role] {
		return nil, fmt.Errorf("invalid role: %s", role)
	}
	return s.repo.UpdateRole(ctx, id, role)
}

// AssignGroup assigns a group to a user.
func (s *UsersService) AssignGroup(ctx context.Context, userID, groupID string) (*repository.User, error) {
	return s.repo.AssignGroup(ctx, userID, groupID)
}

// Delete removes a user.
func (s *UsersService) Delete(ctx context.Context, id string) error {
	return s.repo.DeleteUser(ctx, id)
}
