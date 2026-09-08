package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Attachment — метаданные очищенного вложения (без имени исходного файла).
type Attachment struct {
	ID          uuid.UUID `json:"id"`
	AppealID    uuid.UUID `json:"appeal_id"`
	StorageName string    `json:"-"`
	ContentType string    `json:"content_type"`
	Size        int       `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
}

func (st *Store) CreateAttachment(ctx context.Context, appealID uuid.UUID,
	storageName, contentType string, size int) (Attachment, error) {
	var a Attachment
	err := st.DB.QueryRowContext(ctx, `
		INSERT INTO attachments (appeal_id, storage_name, content_type, size)
		VALUES ($1, $2, $3, $4)
		RETURNING id, appeal_id, storage_name, content_type, size, created_at`,
		appealID, storageName, contentType, size).
		Scan(&a.ID, &a.AppealID, &a.StorageName, &a.ContentType, &a.Size, &a.CreatedAt)
	return a, err
}

// GetAttachment проверяет принадлежность вложения обращению (anti-IDOR).
func (st *Store) GetAttachment(ctx context.Context, appealID, attachmentID uuid.UUID) (Attachment, error) {
	row := st.DB.QueryRowContext(ctx, `
		SELECT id, appeal_id, storage_name, content_type, size, created_at
		FROM attachments WHERE id = $1 AND appeal_id = $2`, attachmentID, appealID)
	var a Attachment
	err := row.Scan(&a.ID, &a.AppealID, &a.StorageName, &a.ContentType, &a.Size, &a.CreatedAt)
	return a, err
}

func (st *Store) ListAttachments(ctx context.Context, appealID uuid.UUID) ([]Attachment, error) {
	rows, err := st.DB.QueryContext(ctx, `
		SELECT id, appeal_id, storage_name, content_type, size, created_at
		FROM attachments WHERE appeal_id = $1 ORDER BY created_at`, appealID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Attachment
	for rows.Next() {
		var a Attachment
		if err := rows.Scan(&a.ID, &a.AppealID, &a.StorageName, &a.ContentType, &a.Size, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (st *Store) CountAttachments(ctx context.Context, appealID uuid.UUID) (int, error) {
	var n int
	err := st.DB.QueryRowContext(ctx,
		`SELECT count(*) FROM attachments WHERE appeal_id = $1`, appealID).Scan(&n)
	return n, err
}
