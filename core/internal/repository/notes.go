package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Note represents a note about a participant.
type Note struct {
	ID              string
	AuthorID        string
	AuthorName      string
	ParticipantID   *string
	ParticipantName *string
	Text            string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NotesRepo handles note persistence.
type NotesRepo struct {
	pool *pgxpool.Pool
}

// NewNotesRepo creates a new NotesRepo.
func NewNotesRepo(pool *pgxpool.Pool) *NotesRepo {
	return &NotesRepo{pool: pool}
}

const noteSelect = `
	SELECT n.id, n.author_id, u.name as author_name, n.participant_id,
	       p.name as participant_name, n.text, n.created_at, n.updated_at
	FROM notes n
	JOIN users u ON u.id = n.author_id
	LEFT JOIN participants p ON p.id = n.participant_id`

// Create inserts a new note.
func (r *NotesRepo) Create(ctx context.Context, authorID string, participantID *string, text string) (*Note, error) {
	const q = `INSERT INTO notes (author_id, participant_id, text) VALUES ($1, $2, $3)
	           RETURNING id`
	var id string
	err := r.pool.QueryRow(ctx, q, authorID, participantID, text).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("create note: %w", err)
	}
	return r.GetByID(ctx, id)
}

// GetByID finds a note by ID.
func (r *NotesRepo) GetByID(ctx context.Context, id string) (*Note, error) {
	q := noteSelect + ` WHERE n.id = $1`
	n := &Note{}
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&n.ID, &n.AuthorID, &n.AuthorName, &n.ParticipantID,
		&n.ParticipantName, &n.Text, &n.CreatedAt, &n.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get note: %w", err)
	}
	return n, nil
}

// ListFilter contains filtering options for listing notes.
type ListFilter struct {
	AuthorID        string
	ParticipantID   string
	UnassignedOnly  bool
	AllNotes        bool
	Limit           int
	Cursor          string
}

// List returns paginated notes based on filter.
func (r *NotesRepo) List(ctx context.Context, f ListFilter) ([]*Note, error) {
	q := noteSelect + ` WHERE 1=1`
	var args []any
	argIdx := 1

	if !f.AllNotes && f.AuthorID != "" {
		q += fmt.Sprintf(` AND n.author_id = $%d`, argIdx)
		args = append(args, f.AuthorID)
		argIdx++
	}
	if f.ParticipantID != "" {
		q += fmt.Sprintf(` AND n.participant_id = $%d`, argIdx)
		args = append(args, f.ParticipantID)
		argIdx++
	}
	if f.UnassignedOnly {
		q += ` AND n.participant_id IS NULL`
	}
	if f.Cursor != "" {
		q += fmt.Sprintf(` AND n.id > $%d`, argIdx)
		args = append(args, f.Cursor)
		argIdx++
	}
	q += fmt.Sprintf(` ORDER BY n.created_at DESC LIMIT %d`, f.Limit)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()

	var notes []*Note
	for rows.Next() {
		n := &Note{}
		if err := rows.Scan(
			&n.ID, &n.AuthorID, &n.AuthorName, &n.ParticipantID,
			&n.ParticipantName, &n.Text, &n.CreatedAt, &n.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// Update modifies an existing note.
func (r *NotesRepo) Update(ctx context.Context, id, text string) (*Note, error) {
	const q = `UPDATE notes SET text = $1, updated_at = now() WHERE id = $2`
	tag, err := r.pool.Exec(ctx, q, text, id)
	if err != nil {
		return nil, fmt.Errorf("update note: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

// Delete removes a note.
func (r *NotesRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM notes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AssignToParticipant sets the participant for a note.
func (r *NotesRepo) AssignToParticipant(ctx context.Context, noteID, participantID string) (*Note, error) {
	const q = `UPDATE notes SET participant_id = $1, updated_at = now() WHERE id = $2`
	tag, err := r.pool.Exec(ctx, q, participantID, noteID)
	if err != nil {
		return nil, fmt.Errorf("assign note: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, noteID)
}
