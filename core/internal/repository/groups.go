package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// Group represents a participant group.
type Group struct {
	bun.BaseModel `bun:"table:groups"`

	ID          string    `bun:"id,pk"`
	Name        string    `bun:"name"`
	Description string    `bun:"description"`
	CreatedAt   time.Time `bun:"created_at"`
}

// GroupsRepo handles group persistence.
type GroupsRepo struct {
	db *bun.DB
}

// NewGroupsRepo creates a new GroupsRepo.
func NewGroupsRepo(db *bun.DB) *GroupsRepo {
	return &GroupsRepo{db: db}
}

// Create inserts a new group.
func (r *GroupsRepo) Create(ctx context.Context, name, description string) (*Group, error) {
	g := &Group{Name: name, Description: description}
	_, err := r.db.NewInsert().Model(g).ExcludeColumn("id").Returning("*").Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}
	return g, nil
}

// GetByID finds a group by ID.
func (r *GroupsRepo) GetByID(ctx context.Context, id string) (*Group, error) {
	g := &Group{}
	err := r.db.NewSelect().Model(g).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get group: %w", err)
	}
	return g, nil
}

// List returns paginated groups.
func (r *GroupsRepo) List(ctx context.Context, limit int, cursor string) ([]*Group, error) {
	var groups []*Group
	q := r.db.NewSelect().Model(&groups)
	if cursor != "" {
		q = q.Where("id > ?", cursor)
	}
	q = q.OrderExpr("name").Limit(limit)
	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	return groups, nil
}

// Update modifies an existing group.
func (r *GroupsRepo) Update(ctx context.Context, id, name, description string) (*Group, error) {
	g := &Group{}
	_, err := r.db.NewUpdate().Model(g).
		Set("name = ?", name).
		Set("description = ?", description).
		Where("id = ?", id).
		Returning("*").
		Exec(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update group: %w", err)
	}
	if g.ID == "" {
		return nil, ErrNotFound
	}
	return g, nil
}

// Delete removes a group.
func (r *GroupsRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.NewDelete().Model((*Group)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
