package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Store) SaveWorkflowArtifact(ctx context.Context, runID, nodeID string, artifact *domain.WorkflowArtifact) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO workflow_artifacts
		(id, run_id, node_id, name, path, content, size, content_type, created_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?)
		ON CONFLICT(run_id, node_id, path) DO UPDATE SET
			name = excluded.name, content = excluded.content, size = excluded.size,
			content_type = excluded.content_type, created_at_ms = excluded.created_at_ms`,
		"wfa_"+ulid.Make().String(), runID, nodeID, artifact.Name, artifact.Path, []byte(artifact.Content),
		artifact.Size, artifact.ContentType, time.Now().UnixMilli())
	return err
}

func (s *Store) GetWorkflowArtifact(ctx context.Context, runID, nodeID, path string) (*domain.WorkflowArtifact, error) {
	var artifact domain.WorkflowArtifact
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT name, path, content, size, content_type FROM workflow_artifacts WHERE run_id = ? AND node_id = ? AND path = ?`,
		runID, nodeID, path).Scan(&artifact.Name, &artifact.Path, &raw, &artifact.Size, &artifact.ContentType)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	artifact.Content = string(raw)
	return &artifact, nil
}
func (s *Store) DeleteWorkflowArtifactsBefore(ctx context.Context, before time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM workflow_artifacts WHERE created_at_ms < ?`, ms(before))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
