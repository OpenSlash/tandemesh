package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// UpsertTaskAttachment stores a provider attachment projection.
func (s *Store) UpsertTaskAttachment(ctx context.Context, attachment *domain.TaskAttachment) error {
	if attachment.CreatedAt.IsZero() {
		attachment.CreatedAt = time.Now().UTC()
	}
	if attachment.Source == "" {
		attachment.Source = "issue"
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO task_attachments
		(id, task_id, provider, external_id, name, url, mime, size, author, source, created_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(task_id, provider, external_id) DO UPDATE SET
			name = excluded.name, url = excluded.url, mime = excluded.mime,
			size = excluded.size, author = excluded.author, source = excluded.source`,
		attachment.ID, attachment.TaskID, attachment.Provider, attachment.ExternalID,
		attachment.Name, attachment.URL, attachment.Mime, attachment.Size,
		attachment.Author, attachment.Source, ms(attachment.CreatedAt))
	return err
}

// ListTaskAttachments returns attachments for a task.
func (s *Store) ListTaskAttachments(ctx context.Context, taskID string) ([]*domain.TaskAttachment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, task_id, provider, external_id, name, url,
		mime, size, author, source, created_at_ms FROM task_attachments WHERE task_id = ? ORDER BY created_at_ms ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.TaskAttachment
	for rows.Next() {
		var (
			item    domain.TaskAttachment
			created int64
		)
		if err := rows.Scan(&item.ID, &item.TaskID, &item.Provider, &item.ExternalID,
			&item.Name, &item.URL, &item.Mime, &item.Size, &item.Author, &item.Source, &created); err != nil {
			return nil, err
		}
		item.CreatedAt = fromMS(created)
		out = append(out, &item)
	}
	return out, rows.Err()
}

// ReplaceTaskAttachments replaces the provider attachment set for a task.
func (s *Store) ReplaceTaskAttachments(ctx context.Context, taskID, provider, source string, items []*domain.TaskAttachment) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM task_attachments WHERE task_id = ? AND provider = ? AND source = ?`, taskID, provider, source); err != nil {
		return err
	}
	for _, item := range items {
		if item.CreatedAt.IsZero() {
			item.CreatedAt = time.Now().UTC()
		}
		if item.Source == "" {
			item.Source = source
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_attachments
			(id, task_id, provider, external_id, name, url, mime, size, author, source, created_at_ms)
			VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			item.ID, item.TaskID, item.Provider, item.ExternalID,
			item.Name, item.URL, item.Mime, item.Size, item.Author, item.Source, ms(item.CreatedAt)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetTaskAttachment returns one attachment or nil.
func (s *Store) GetTaskAttachment(ctx context.Context, id string) (*domain.TaskAttachment, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, task_id, provider, external_id, name, url,
		mime, size, author, source, created_at_ms FROM task_attachments WHERE id = ?`, id)
	var (
		item    domain.TaskAttachment
		created int64
	)
	err := row.Scan(&item.ID, &item.TaskID, &item.Provider, &item.ExternalID,
		&item.Name, &item.URL, &item.Mime, &item.Size, &item.Author, &item.Source, &created)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item.CreatedAt = fromMS(created)
	return &item, nil
}
