package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// Participant represents an event participant.
type Participant struct {
	bun.BaseModel `bun:"table:participants,alias:p"`

	ID               string    `bun:"id,pk"`
	Name             string    `bun:"name"`
	TelegramID       *int64    `bun:"telegram_id"`
	CustomIdentifier *string   `bun:"custom_identifier"`
	GroupID          *string   `bun:"group_id"`
	PhotoMediaID     *string   `bun:"photo_media_id"`
	CreatedAt        time.Time `bun:"created_at"`
	UpdatedAt        time.Time `bun:"updated_at"`

	// Scan-only fields populated by list/get queries.
	GroupName  *string `bun:"group_name,scanonly"`
	NotesCount int32   `bun:"notes_count,scanonly"`
}

// ParticipantsRepo handles participant persistence.
type ParticipantsRepo struct {
	db *bun.DB
}

// NewParticipantsRepo creates a new ParticipantsRepo.
func NewParticipantsRepo(db *bun.DB) *ParticipantsRepo {
	return &ParticipantsRepo{db: db}
}

func (r *ParticipantsRepo) selectWithJoins() *bun.SelectQuery {
	return r.db.NewSelect().
		TableExpr("participants AS p").
		ColumnExpr("p.*, g.name AS group_name, (SELECT COUNT(*) FROM notes WHERE participant_id = p.id)::int AS notes_count").
		Join("LEFT JOIN groups g ON g.id = p.group_id")
}

// Create inserts a new participant.
func (r *ParticipantsRepo) Create(ctx context.Context, name string, telegramID *int64, customID *string, groupID *string) (*Participant, error) {
	p := &Participant{
		Name:             name,
		TelegramID:       telegramID,
		CustomIdentifier: customID,
		GroupID:          groupID,
	}
	_, err := r.db.NewInsert().Model(p).Returning("*").Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("create participant: %w", err)
	}
	return r.GetByID(ctx, p.ID)
}

// GetByID finds a participant by ID.
func (r *ParticipantsRepo) GetByID(ctx context.Context, id string) (*Participant, error) {
	p := &Participant{}
	err := r.selectWithJoins().Where("p.id = ?", id).Scan(ctx, p)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get participant: %w", err)
	}
	return p, nil
}

// List returns paginated participants with optional group filter and search.
func (r *ParticipantsRepo) List(ctx context.Context, groupID, search string, limit int, cursor string) ([]*Participant, error) {
	var participants []*Participant
	q := r.selectWithJoins()
	if groupID != "" {
		q = q.Where("p.group_id = ?", groupID)
	}
	if search != "" {
		q = q.Where("p.name ILIKE ?", "%"+search+"%")
	}
	if cursor != "" {
		q = q.Where("p.id > ?", cursor)
	}
	q = q.OrderExpr("p.name").Limit(limit)
	if err := q.Scan(ctx, &participants); err != nil {
		return nil, fmt.Errorf("list participants: %w", err)
	}
	return participants, nil
}

// Update modifies an existing participant.
func (r *ParticipantsRepo) Update(ctx context.Context, id, name string, telegramID *int64, customID *string, groupID *string) (*Participant, error) {
	res, err := r.db.NewUpdate().TableExpr("participants").
		Set("name = ?", name).
		Set("telegram_id = ?", telegramID).
		Set("custom_identifier = ?", customID).
		Set("group_id = ?", groupID).
		Set("updated_at = now()").
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update participant: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

// Delete removes a participant.
func (r *ParticipantsRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.NewDelete().TableExpr("participants").Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete participant: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetPhoto sets the photo media ID for a participant.
func (r *ParticipantsRepo) SetPhoto(ctx context.Context, participantID, mediaID string) (*Participant, error) {
	res, err := r.db.NewUpdate().TableExpr("participants").
		Set("photo_media_id = ?", mediaID).
		Set("updated_at = now()").
		Where("id = ?", participantID).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("set photo: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, participantID)
}
