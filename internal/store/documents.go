package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Store) CreateDocument(ctx context.Context, d *domain.Document) error {
	now := time.Now().UTC()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	d.UpdatedAt = now
	tags, _ := json.Marshal(d.Tags)
	_, err := s.db.ExecContext(ctx, `INSERT INTO documents(id, workspace_id, title, body, tags, created_by, created_at_ms, updated_at_ms)
		VALUES(?,?,?,?,?,?,?,?)`,
		d.ID, d.WorkspaceID, d.Title, d.Body, string(tags), d.CreatedBy, ms(d.CreatedAt), ms(d.UpdatedAt))
	return err
}

func (s *Store) UpdateDocument(ctx context.Context, d *domain.Document) error {
	d.UpdatedAt = time.Now().UTC()
	tags, _ := json.Marshal(d.Tags)
	_, err := s.db.ExecContext(ctx, `UPDATE documents SET title = ?, body = ?, tags = ?, updated_at_ms = ? WHERE id = ?`,
		d.Title, d.Body, string(tags), ms(d.UpdatedAt), d.ID)
	return err
}

func (s *Store) GetDocument(ctx context.Context, id string) (*domain.Document, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, workspace_id, title, body, tags, created_by, created_at_ms, updated_at_ms FROM documents WHERE id = ?`, id)
	return scanDocument(row)
}

func (s *Store) ListDocuments(ctx context.Context, workspaceID string, limit int) ([]*domain.Document, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, workspace_id, title, body, tags, created_by, created_at_ms, updated_at_ms
		FROM documents WHERE workspace_id = ? ORDER BY updated_at_ms DESC LIMIT ?`, workspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Document
	for rows.Next() {
		d, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) DeleteDocument(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM documents WHERE id = ?`, id)
	return err
}

type documentScanner interface {
	Scan(dest ...any) error
}

func scanDocument(row documentScanner) (*domain.Document, error) {
	var (
		d                domain.Document
		tags             string
		created, updated int64
	)
	if err := row.Scan(&d.ID, &d.WorkspaceID, &d.Title, &d.Body, &tags, &d.CreatedBy, &created, &updated); err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(tags), &d.Tags)
	d.CreatedAt = fromMS(created)
	d.UpdatedAt = fromMS(updated)
	return &d, nil
}
