package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MediaRecord holds metadata about stored media.
type MediaRecord struct {
	ID           string
	FilePath     string
	MimeType     string
	OriginalName string
	SizeBytes    int64
	CreatedAt    time.Time
}

// MediaService handles file storage operations.
type MediaService struct {
	pool     *pgxpool.Pool
	mediaDir string
}

// NewMediaService creates a new MediaService.
func NewMediaService(pool *pgxpool.Pool, mediaDir string) *MediaService {
	return &MediaService{pool: pool, mediaDir: mediaDir}
}

// Upload stores file bytes and records metadata.
func (s *MediaService) Upload(ctx context.Context, data []byte, mimeType, originalName string) (*MediaRecord, error) {
	if err := os.MkdirAll(s.mediaDir, 0o755); err != nil {
		return nil, fmt.Errorf("create media dir: %w", err)
	}

	// Insert metadata first to get the UUID.
	const q = `INSERT INTO media (file_path, mime_type, original_name, size_bytes)
	           VALUES ($1, $2, $3, $4)
	           RETURNING id, file_path, mime_type, original_name, size_bytes, created_at`

	// We'll update file_path after we know the ID.
	// Use a placeholder path first.
	m := &MediaRecord{}
	err := s.pool.QueryRow(ctx, q, "pending", mimeType, originalName, int64(len(data))).Scan(
		&m.ID, &m.FilePath, &m.MimeType, &m.OriginalName, &m.SizeBytes, &m.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert media record: %w", err)
	}

	// Determine extension from mime type.
	ext := extFromMime(mimeType)
	fileName := m.ID + ext
	filePath := filepath.Join(s.mediaDir, fileName)

	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		// Clean up DB record on file write failure.
		_, _ = s.pool.Exec(ctx, `DELETE FROM media WHERE id = $1`, m.ID)
		return nil, fmt.Errorf("write file: %w", err)
	}

	// Update path in DB.
	if _, err := s.pool.Exec(ctx, `UPDATE media SET file_path = $1 WHERE id = $2`, filePath, m.ID); err != nil {
		return nil, fmt.Errorf("update file path: %w", err)
	}
	m.FilePath = filePath
	return m, nil
}

// Get retrieves media metadata and file bytes.
func (s *MediaService) Get(ctx context.Context, id string) (*MediaRecord, []byte, error) {
	const q = `SELECT id, file_path, mime_type, original_name, size_bytes, created_at FROM media WHERE id = $1`
	m := &MediaRecord{}
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&m.ID, &m.FilePath, &m.MimeType, &m.OriginalName, &m.SizeBytes, &m.CreatedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("get media: %w", err)
	}
	data, err := os.ReadFile(m.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read file: %w", err)
	}
	return m, data, nil
}

// Delete removes media file and its record.
func (s *MediaService) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM media WHERE id = $1 RETURNING file_path`
	var filePath string
	err := s.pool.QueryRow(ctx, q, id).Scan(&filePath)
	if err != nil {
		return fmt.Errorf("delete media record: %w", err)
	}
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file: %w", err)
	}
	return nil
}

func extFromMime(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}
