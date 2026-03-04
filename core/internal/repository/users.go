package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User represents a bot user.
type User struct {
	ID         string
	TelegramID int64
	Name       string
	Username   string
	Role       string
	GroupID    *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// UsersRepo handles user persistence.
type UsersRepo struct {
	pool *pgxpool.Pool
}

// NewUsersRepo creates a new UsersRepo.
func NewUsersRepo(pool *pgxpool.Pool) *UsersRepo {
	return &UsersRepo{pool: pool}
}

// GetByTelegramID finds a user by Telegram ID.
func (r *UsersRepo) GetByTelegramID(ctx context.Context, telegramID int64) (*User, error) {
	const q = `SELECT id, telegram_id, name, username, role, group_id, created_at, updated_at
	           FROM users WHERE telegram_id = $1`
	u := &User{}
	err := r.pool.QueryRow(ctx, q, telegramID).Scan(
		&u.ID, &u.TelegramID, &u.Name, &u.Username, &u.Role, &u.GroupID, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by telegram_id: %w", err)
	}
	return u, nil
}

// GetByID finds a user by internal ID.
func (r *UsersRepo) GetByID(ctx context.Context, id string) (*User, error) {
	const q = `SELECT id, telegram_id, name, username, role, group_id, created_at, updated_at
	           FROM users WHERE id = $1`
	u := &User{}
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.TelegramID, &u.Name, &u.Username, &u.Role, &u.GroupID, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

// CreateUser inserts a new user.
func (r *UsersRepo) CreateUser(ctx context.Context, telegramID int64, name, username, role string) (*User, error) {
	const q = `INSERT INTO users (telegram_id, name, username, role)
	           VALUES ($1, $2, $3, $4)
	           RETURNING id, telegram_id, name, username, role, group_id, created_at, updated_at`
	u := &User{}
	err := r.pool.QueryRow(ctx, q, telegramID, name, username, role).Scan(
		&u.ID, &u.TelegramID, &u.Name, &u.Username, &u.Role, &u.GroupID, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// ListUsers returns paginated users with optional role filter.
func (r *UsersRepo) ListUsers(ctx context.Context, roleFilter string, limit int, cursor string) ([]*User, error) {
	q := `SELECT id, telegram_id, name, username, role, group_id, created_at, updated_at
	      FROM users WHERE ($1 = '' OR role = $1)`
	args := []any{roleFilter}
	if cursor != "" {
		q += ` AND id > $2`
		args = append(args, cursor)
	}
	q += fmt.Sprintf(` ORDER BY id LIMIT %d`, limit)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.TelegramID, &u.Name, &u.Username, &u.Role, &u.GroupID, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// UpdateRole changes the role of a user.
func (r *UsersRepo) UpdateRole(ctx context.Context, id, role string) (*User, error) {
	const q = `UPDATE users SET role = $1, updated_at = now()
	           WHERE id = $2
	           RETURNING id, telegram_id, name, username, role, group_id, created_at, updated_at`
	u := &User{}
	err := r.pool.QueryRow(ctx, q, role, id).Scan(
		&u.ID, &u.TelegramID, &u.Name, &u.Username, &u.Role, &u.GroupID, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update role: %w", err)
	}
	return u, nil
}

// AssignGroup assigns a group to a user.
func (r *UsersRepo) AssignGroup(ctx context.Context, userID, groupID string) (*User, error) {
	const q = `UPDATE users SET group_id = $1, updated_at = now()
	           WHERE id = $2
	           RETURNING id, telegram_id, name, username, role, group_id, created_at, updated_at`
	u := &User{}
	err := r.pool.QueryRow(ctx, q, groupID, userID).Scan(
		&u.ID, &u.TelegramID, &u.Name, &u.Username, &u.Role, &u.GroupID, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("assign group: %w", err)
	}
	return u, nil
}

// DeleteUser removes a user.
func (r *UsersRepo) DeleteUser(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
