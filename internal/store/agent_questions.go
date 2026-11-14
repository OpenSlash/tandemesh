package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// CreateAgentQuestion persists a question that needs a human answer.
func (s *Store) CreateAgentQuestion(ctx context.Context, question *domain.AgentQuestion) (*domain.AgentQuestion, error) {
	if question.ID == "" {
		question.ID = "q_" + ulid.Make().String()
	}
	if question.Status == "" {
		question.Status = "pending"
	}
	if question.CreatedAt.IsZero() {
		question.CreatedAt = time.Now().UTC()
	}
	if question.ExpiresAt == nil {
		expires := question.CreatedAt.Add(15 * time.Minute)
		question.ExpiresAt = &expires
	}
	var expires any
	if question.ExpiresAt != nil {
		expires = ms(*question.ExpiresAt)
	}
	options, _ := json.Marshal(question.Options)
	_, err := s.db.ExecContext(ctx, `INSERT INTO agent_questions
		(id, session_id, run_id, question, options_json, status, answer, created_at_ms, answered_at_ms, expires_at_ms)
		VALUES (?,?,?,?,?,?,?,?,NULL,?)`,
		question.ID, question.SessionID, question.RunID, question.Question, string(options),
		question.Status, question.Answer, ms(question.CreatedAt), expires)
	if err != nil {
		return nil, err
	}
	return question, nil
}

// ListAgentQuestions returns questions for a session, optionally by status.
func (s *Store) ListAgentQuestions(ctx context.Context, sessionID, status string) ([]*domain.AgentQuestion, error) {
	_ = s.ExpireAgentQuestions(ctx, time.Now().UTC())
	q := `SELECT id, session_id, run_id, question, options_json, status, answer, created_at_ms, answered_at_ms, expires_at_ms
		FROM agent_questions WHERE session_id = ?`
	args := []any{sessionID}
	if status != "" {
		q += " AND status = ?"
		args = append(args, status)
	}
	q += " ORDER BY created_at_ms ASC, id ASC"
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.AgentQuestion{}
	for rows.Next() {
		question, err := scanAgentQuestion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, question)
	}
	return out, rows.Err()
}

// GetAgentQuestion returns one question or nil.
func (s *Store) GetAgentQuestion(ctx context.Context, id string) (*domain.AgentQuestion, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, session_id, run_id, question, options_json, status, answer, created_at_ms, answered_at_ms, expires_at_ms
		FROM agent_questions WHERE id = ?`, id)
	return scanAgentQuestion(row)
}

// AnswerAgentQuestion records an answer for a pending question.
func (s *Store) AnswerAgentQuestion(ctx context.Context, id, answer string) (*domain.AgentQuestion, error) {
	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx, `UPDATE agent_questions SET status = 'answered', answer = ?, answered_at_ms = ?
		WHERE id = ? AND status = 'pending' AND (expires_at_ms IS NULL OR expires_at_ms > ?)`, answer, ms(now), id, ms(now))
	if err != nil {
		return nil, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, nil
	}
	return s.GetAgentQuestion(ctx, id)
}

// ExpireAgentQuestions marks pending questions past their deadline.
func (s *Store) ExpireAgentQuestions(ctx context.Context, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE agent_questions SET status = 'expired'
		WHERE status = 'pending' AND expires_at_ms IS NOT NULL AND expires_at_ms <= ?`, ms(now))
	return err
}
func scanAgentQuestion(sc interface{ Scan(...any) error }) (*domain.AgentQuestion, error) {
	var (
		question domain.AgentQuestion
		options  string
		created  int64
		answered sql.NullInt64
		expires  sql.NullInt64
	)
	if err := sc.Scan(&question.ID, &question.SessionID, &question.RunID, &question.Question, &options,
		&question.Status, &question.Answer, &created, &answered, &expires); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	_ = json.Unmarshal([]byte(options), &question.Options)
	question.CreatedAt = fromMS(created)
	if answered.Valid {
		ts := fromMS(answered.Int64)
		question.AnsweredAt = &ts
	}
	if expires.Valid {
		ts := fromMS(expires.Int64)
		question.ExpiresAt = &ts
	}
	return &question, nil
}
