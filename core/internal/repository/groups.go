package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Group represents a participant group.
type Group struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
}

// GroupsRepo handles group persistence.
type GroupsRepo struct {
	pool *pgxpool.Pool
}

// NewGroupsRepo creates a new GroupsRepo.
func NewGroupsRepo(pool *pgxpool.Pool) *GroupsRepo {
	return &GroupsRepo{pool: pool}
}

// Create inserts a new group.
func (r *GroupsRepo) Create(ctx context.Context, name, description string) (*Group, error) {
	const q = `INSERT INTO groups (name, description) VALUES ($1, $2)
	           RETURNING id, name, description, created_at`
	g := &Group{}
	err := r.pool.QueryRow(ctx, q, name, description).Scan(&g.ID, &g.Name, &g.Description, &g.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}
	return g, nil
}

// GetByID finds a group by ID.
func (r *GroupsRepo) GetByID(ctx context.Context, id string) (*Group, error) {
	const q = `SELECT id, name, description, created_at FROM groups WHERE id = $1`
	g := &Group{}
	err := r.pool.QueryRow(ctx, q, id).Scan(&g.ID, &g.Name, &g.Description, &g.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get group: %w", err)
	}
	return g, nil
}

// List returns paginated groups.
func (r *GroupsRepo) List(ctx context.Context, limit int, cursor string) ([]*Group, error) {
	q := `SELECT id, name, description, created_at FROM groups`
	var args []any
	if cursor != "" {
		q += ` WHERE id > $1`
		args = append(args, cursor)
	}
	q += fmt.Sprintf(` ORDER BY name LIMIT %d`, limit)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer rows.Close()

	var groups []*Group
	for rows.Next() {
		g := &Group{}
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// Update modifies an existing group.
func (r *GroupsRepo) Update(ctx context.Context, id, name, description string) (*Group, error) {
	const q = `UPDATE groups SET name = $1, description = $2 WHERE id = $3
	           RETURNING id, name, description, created_at`
	g := &Group{}
	err := r.pool.QueryRow(ctx, q, name, description, id).Scan(&g.ID, &g.Name, &g.Description, &g.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update group: %w", err)
	}
	return g, nil
}

// Delete removes a group.
func (r *GroupsRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM groups WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
