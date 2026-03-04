package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Participant represents an event participant.
type Participant struct {
	ID               string
	Name             string
	TelegramID       *int64
	CustomIdentifier *string
	GroupID          *string
	GroupName        *string
	PhotoMediaID     *string
	NotesCount       int32
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ParticipantsRepo handles participant persistence.
type ParticipantsRepo struct {
	pool *pgxpool.Pool
}

// NewParticipantsRepo creates a new ParticipantsRepo.
func NewParticipantsRepo(pool *pgxpool.Pool) *ParticipantsRepo {
	return &ParticipantsRepo{pool: pool}
}

const participantSelect = `
	SELECT p.id, p.name, p.telegram_id, p.custom_identifier, p.group_id,
	       g.name as group_name, p.photo_media_id,
	       (SELECT COUNT(*) FROM notes WHERE participant_id = p.id)::int as notes_count,
	       p.created_at, p.updated_at
	FROM participants p
	LEFT JOIN groups g ON g.id = p.group_id`

// Create inserts a new participant.
func (r *ParticipantsRepo) Create(ctx context.Context, name string, telegramID *int64, customID *string, groupID *string) (*Participant, error) {
	const q = `INSERT INTO participants (name, telegram_id, custom_identifier, group_id)
	           VALUES ($1, $2, $3, $4)
	           RETURNING id, name, telegram_id, custom_identifier, group_id, photo_media_id, created_at, updated_at`
	row := &struct {
		ID               string
		Name             string
		TelegramID       *int64
		CustomIdentifier *string
		GroupID          *string
		PhotoMediaID     *string
		CreatedAt        time.Time
		UpdatedAt        time.Time
	}{}
	err := r.pool.QueryRow(ctx, q, name, telegramID, customID, groupID).Scan(
		&row.ID, &row.Name, &row.TelegramID, &row.CustomIdentifier,
		&row.GroupID, &row.PhotoMediaID, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create participant: %w", err)
	}
	return &Participant{
		ID:               row.ID,
		Name:             row.Name,
		TelegramID:       row.TelegramID,
		CustomIdentifier: row.CustomIdentifier,
		GroupID:          row.GroupID,
		PhotoMediaID:     row.PhotoMediaID,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}, nil
}

// GetByID finds a participant by ID.
func (r *ParticipantsRepo) GetByID(ctx context.Context, id string) (*Participant, error) {
	q := participantSelect + ` WHERE p.id = $1`
	p := &Participant{}
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&p.ID, &p.Name, &p.TelegramID, &p.CustomIdentifier,
		&p.GroupID, &p.GroupName, &p.PhotoMediaID, &p.NotesCount,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get participant: %w", err)
	}
	return p, nil
}

// List returns paginated participants with optional group filter and search.
func (r *ParticipantsRepo) List(ctx context.Context, groupID, search string, limit int, cursor string) ([]*Participant, error) {
	q := participantSelect + ` WHERE 1=1`
	var args []any
	argIdx := 1

	if groupID != "" {
		q += fmt.Sprintf(` AND p.group_id = $%d`, argIdx)
		args = append(args, groupID)
		argIdx++
	}
	if search != "" {
		q += fmt.Sprintf(` AND p.name ILIKE $%d`, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if cursor != "" {
		q += fmt.Sprintf(` AND p.id > $%d`, argIdx)
		args = append(args, cursor)
		argIdx++
	}
	q += fmt.Sprintf(` ORDER BY p.name LIMIT %d`, limit)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list participants: %w", err)
	}
	defer rows.Close()

	var participants []*Participant
	for rows.Next() {
		p := &Participant{}
		if err := rows.Scan(
			&p.ID, &p.Name, &p.TelegramID, &p.CustomIdentifier,
			&p.GroupID, &p.GroupName, &p.PhotoMediaID, &p.NotesCount,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan participant: %w", err)
		}
		participants = append(participants, p)
	}
	return participants, rows.Err()
}

// Update modifies an existing participant.
func (r *ParticipantsRepo) Update(ctx context.Context, id, name string, telegramID *int64, customID *string, groupID *string) (*Participant, error) {
	const q = `UPDATE participants
	           SET name = $1, telegram_id = $2, custom_identifier = $3, group_id = $4, updated_at = now()
	           WHERE id = $5`
	tag, err := r.pool.Exec(ctx, q, name, telegramID, customID, groupID, id)
	if err != nil {
		return nil, fmt.Errorf("update participant: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

// Delete removes a participant.
func (r *ParticipantsRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM participants WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete participant: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetPhoto sets the photo media ID for a participant.
func (r *ParticipantsRepo) SetPhoto(ctx context.Context, participantID, mediaID string) (*Participant, error) {
	const q = `UPDATE participants SET photo_media_id = $1, updated_at = now() WHERE id = $2`
	tag, err := r.pool.Exec(ctx, q, mediaID, participantID)
	if err != nil {
		return nil, fmt.Errorf("set photo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, participantID)
}
