package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// Note represents a note about a participant.
type Note struct {
	bun.BaseModel `bun:"table:notes,alias:n"`

	ID            string    `bun:"id,pk"`
	AuthorID      string    `bun:"author_id"`
	ParticipantID *string   `bun:"participant_id"`
	Text          string    `bun:"text"`
	CreatedAt     time.Time `bun:"created_at"`
	UpdatedAt     time.Time `bun:"updated_at"`

	// Scan-only fields populated by list/get queries.
	AuthorName      string  `bun:"author_name,scanonly"`
	ParticipantName *string `bun:"participant_name,scanonly"`
}

// NotesRepo handles note persistence.
type NotesRepo struct {
	db *bun.DB
}

// NewNotesRepo creates a new NotesRepo.
func NewNotesRepo(db *bun.DB) *NotesRepo {
	return &NotesRepo{db: db}
}

func (r *NotesRepo) selectWithJoins() *bun.SelectQuery {
	return r.db.NewSelect().
		TableExpr("notes AS n").
		ColumnExpr("n.*, u.name AS author_name, p.name AS participant_name").
		Join("JOIN users u ON u.id = n.author_id").
		Join("LEFT JOIN participants p ON p.id = n.participant_id")
}

// Create inserts a new note.
func (r *NotesRepo) Create(ctx context.Context, authorID string, participantID *string, text string) (*Note, error) {
	n := &Note{AuthorID: authorID, ParticipantID: participantID, Text: text}
	_, err := r.db.NewInsert().Model(n).ExcludeColumn("id").Returning("id").Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("create note: %w", err)
	}
	return r.GetByID(ctx, n.ID)
}

// GetByID finds a note by ID.
func (r *NotesRepo) GetByID(ctx context.Context, id string) (*Note, error) {
	n := &Note{}
	err := r.selectWithJoins().Where("n.id = ?", id).Scan(ctx, n)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get note: %w", err)
	}
	return n, nil
}

// ListFilter contains filtering options for listing notes.
type ListFilter struct {
	AuthorID       string
	ParticipantID  string
	UnassignedOnly bool
	AllNotes       bool
	Limit          int
	Offset         int
}

// List returns paginated notes based on filter.
func (r *NotesRepo) List(ctx context.Context, f ListFilter) ([]*Note, error) {
	var notes []*Note
	q := r.selectWithJoins()
	if !f.AllNotes && f.AuthorID != "" {
		q = q.Where("n.author_id = ?", f.AuthorID)
	}
	if f.ParticipantID != "" {
		q = q.Where("n.participant_id = ?", f.ParticipantID)
	}
	if f.UnassignedOnly {
		q = q.Where("n.participant_id IS NULL")
	}
	q = q.OrderExpr("n.created_at DESC").Limit(f.Limit).Offset(f.Offset)
	if err := q.Scan(ctx, &notes); err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	return notes, nil
}

// Update modifies an existing note.
func (r *NotesRepo) Update(ctx context.Context, id, text string) (*Note, error) {
	res, err := r.db.NewUpdate().TableExpr("notes").
		Set("text = ?", text).
		Set("updated_at = now()").
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update note: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

// Delete removes a note.
func (r *NotesRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.NewDelete().TableExpr("notes").Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Count returns the total number of notes matching the filter (ignores Limit/Cursor).
func (r *NotesRepo) Count(ctx context.Context, f ListFilter) (int32, error) {
	q := r.db.NewSelect().TableExpr("notes AS n")
	if !f.AllNotes && f.AuthorID != "" {
		q = q.Where("n.author_id = ?", f.AuthorID)
	}
	if f.ParticipantID != "" {
		q = q.Where("n.participant_id = ?", f.ParticipantID)
	}
	if f.UnassignedOnly {
		q = q.Where("n.participant_id IS NULL")
	}
	n, err := q.Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count notes: %w", err)
	}
	return int32(n), nil
}

// AssignToParticipant sets the participant for a note.
func (r *NotesRepo) AssignToParticipant(ctx context.Context, noteID, participantID string) (*Note, error) {
	res, err := r.db.NewUpdate().TableExpr("notes").
		Set("participant_id = ?", participantID).
		Set("updated_at = now()").
		Where("id = ?", noteID).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("assign note: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, noteID)
}
