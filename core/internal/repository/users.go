package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// User represents a bot user.
type User struct {
	bun.BaseModel `bun:"table:users"`

	ID         string    `bun:"id,pk"`
	TelegramID int64     `bun:"telegram_id"`
	Name       string    `bun:"name"`
	Username   string    `bun:"username"`
	Role       string    `bun:"role"`
	GroupID    *string   `bun:"group_id"`
	CreatedAt  time.Time `bun:"created_at"`
	UpdatedAt  time.Time `bun:"updated_at"`
}

// UsersRepo handles user persistence.
type UsersRepo struct {
	db *bun.DB
}

// NewUsersRepo creates a new UsersRepo.
func NewUsersRepo(db *bun.DB) *UsersRepo {
	return &UsersRepo{db: db}
}

// GetByTelegramID finds a user by Telegram ID.
func (r *UsersRepo) GetByTelegramID(ctx context.Context, telegramID int64) (*User, error) {
	u := &User{}
	err := r.db.NewSelect().Model(u).Where("telegram_id = ?", telegramID).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by telegram_id: %w", err)
	}
	return u, nil
}

// GetByID finds a user by internal ID.
func (r *UsersRepo) GetByID(ctx context.Context, id string) (*User, error) {
	u := &User{}
	err := r.db.NewSelect().Model(u).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

// CreateUser inserts a new user.
func (r *UsersRepo) CreateUser(ctx context.Context, telegramID int64, name, username, role string) (*User, error) {
	u := &User{
		TelegramID: telegramID,
		Name:       name,
		Username:   username,
		Role:       role,
	}
	_, err := r.db.NewInsert().Model(u).Returning("*").Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// ListUsers returns paginated users with optional role filter.
func (r *UsersRepo) ListUsers(ctx context.Context, roleFilter string, limit int, cursor string) ([]*User, error) {
	var users []*User
	q := r.db.NewSelect().Model(&users)
	if roleFilter != "" {
		q = q.Where("role = ?", roleFilter)
	}
	if cursor != "" {
		q = q.Where("id > ?", cursor)
	}
	q = q.OrderExpr("id").Limit(limit)
	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

// UpdateRole changes the role of a user.
func (r *UsersRepo) UpdateRole(ctx context.Context, id, role string) (*User, error) {
	u := &User{}
	_, err := r.db.NewUpdate().Model(u).
		Set("role = ?", role).
		Set("updated_at = now()").
		Where("id = ?", id).
		Returning("*").
		Exec(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update role: %w", err)
	}
	if u.ID == "" {
		return nil, ErrNotFound
	}
	return u, nil
}

// AssignGroup assigns a group to a user.
func (r *UsersRepo) AssignGroup(ctx context.Context, userID, groupID string) (*User, error) {
	u := &User{}
	_, err := r.db.NewUpdate().Model(u).
		Set("group_id = ?", groupID).
		Set("updated_at = now()").
		Where("id = ?", userID).
		Returning("*").
		Exec(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("assign group: %w", err)
	}
	if u.ID == "" {
		return nil, ErrNotFound
	}
	return u, nil
}

// DeleteUser removes a user.
func (r *UsersRepo) DeleteUser(ctx context.Context, id string) error {
	res, err := r.db.NewDelete().Model((*User)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
