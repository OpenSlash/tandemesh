package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

const skillColumns = `id, workspace_id, name, description, version, source, source_url, tags_json, enabled, checksum, body,
	created_at_ms, updated_at_ms`

// UpsertSkill creates or updates a workspace skill by name.
func (s *Store) UpsertSkill(ctx context.Context, skill *domain.Skill) error {
	now := time.Now().UTC()
	if skill.CreatedAt.IsZero() {
		skill.CreatedAt = now
	}
	skill.UpdatedAt = now
	if skill.Source == "" {
		skill.Source = "manual"
	}
	tags, err := json.Marshal(skill.Tags)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO skills
		(id, workspace_id, name, description, version, source, source_url, tags_json, enabled, checksum, body, created_at_ms, updated_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(workspace_id, name) DO UPDATE SET
			description = excluded.description, version = excluded.version, source = excluded.source,
			source_url = excluded.source_url, tags_json = excluded.tags_json, enabled = excluded.enabled,
			checksum = excluded.checksum, body = excluded.body, updated_at_ms = excluded.updated_at_ms`,
		skill.ID, skill.WorkspaceID, skill.Name, skill.Description, skill.Version, skill.Source, skill.SourceURL,
		string(tags), boolInt(skill.Enabled), skill.Checksum, skill.Body, ms(skill.CreatedAt), ms(skill.UpdatedAt))
	return err
}

// ListSkills returns workspace skills ordered by name.
func (s *Store) ListSkills(ctx context.Context, workspaceID string) ([]*domain.Skill, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+skillColumns+` FROM skills WHERE workspace_id = ? ORDER BY name`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Skill
	for rows.Next() {
		sk, err := scanSkill(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sk)
	}
	return out, rows.Err()
}

// GetSkill returns one skill by id or nil.
func (s *Store) GetSkill(ctx context.Context, id string) (*domain.Skill, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+skillColumns+` FROM skills WHERE id = ?`, id)
	return scanSkill(row)
}

// GetSkillByName returns a workspace skill by name or nil.
func (s *Store) GetSkillByName(ctx context.Context, workspaceID, name string) (*domain.Skill, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+skillColumns+` FROM skills WHERE workspace_id = ? AND name = ?`, workspaceID, name)
	return scanSkill(row)
}

// SetSkillEnabled toggles whether a skill is injected into prompts.
func (s *Store) SetSkillEnabled(ctx context.Context, id string, enabled bool) (*domain.Skill, error) {
	if _, err := s.db.ExecContext(ctx, `UPDATE skills SET enabled = ?, updated_at_ms = ? WHERE id = ?`, boolInt(enabled), ms(time.Now()), id); err != nil {
		return nil, err
	}
	return s.GetSkill(ctx, id)
}

// DeleteSkill removes a skill and reports whether a row was deleted.
func (s *Store) DeleteSkill(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM skills WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func scanSkill(sc interface{ Scan(...any) error }) (*domain.Skill, error) {
	var (
		sk               domain.Skill
		tags             string
		enabled          int
		created, updated int64
	)
	err := sc.Scan(&sk.ID, &sk.WorkspaceID, &sk.Name, &sk.Description, &sk.Version, &sk.Source, &sk.SourceURL,
		&tags, &enabled, &sk.Checksum, &sk.Body, &created, &updated)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(tags), &sk.Tags)
	sk.Enabled = enabled != 0
	sk.CreatedAt = fromMS(created)
	sk.UpdatedAt = fromMS(updated)
	return &sk, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
