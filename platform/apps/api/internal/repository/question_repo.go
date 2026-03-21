package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"platform/api/internal/model"
)

type QuestionRepository struct{ db *sqlx.DB }

func NewQuestionRepository(db *sqlx.DB) *QuestionRepository { return &QuestionRepository{db: db} }

func (r *QuestionRepository) Create(ctx context.Context, tenantID, eventID, userID, subject string, isPublic bool) (*model.Question, error) {
	var q model.Question
	err := r.db.GetContext(ctx, &q,
		`INSERT INTO questions (tenant_id, event_id, user_id, subject, is_public)
		 VALUES ($1, $2, $3, $4, $5) RETURNING *`,
		tenantID, eventID, userID, subject, isPublic,
	)
	if err != nil {
		return nil, fmt.Errorf("question Create: %w", err)
	}
	return &q, nil
}

func (r *QuestionRepository) FindByID(ctx context.Context, tenantID, id string) (*model.Question, error) {
	var q model.Question
	err := r.db.GetContext(ctx, &q,
		`SELECT * FROM questions WHERE id = $1 AND tenant_id = $2 LIMIT 1`, id, tenantID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("question FindByID: %w", err)
	}
	return &q, nil
}

func (r *QuestionRepository) ListByEvent(ctx context.Context, tenantID, eventID string, publicOnly bool) ([]model.Question, error) {
	query := `SELECT * FROM questions WHERE event_id = $1 AND tenant_id = $2`
	args := []interface{}{eventID, tenantID}
	if publicOnly {
		query += " AND is_public = true"
	}
	query += " ORDER BY created_at DESC"
	var questions []model.Question
	if err := r.db.SelectContext(ctx, &questions, query, args...); err != nil {
		return nil, fmt.Errorf("question ListByEvent: %w", err)
	}
	return questions, nil
}

func (r *QuestionRepository) UpdateStatus(ctx context.Context, tenantID, id, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE questions SET status = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`,
		status, id, tenantID)
	if err != nil {
		return fmt.Errorf("question UpdateStatus: %w", err)
	}
	return nil
}

// ── Question Messages ──────────────────────────────────────────────────────────

type QuestionMessageRepository struct{ db *sqlx.DB }

func NewQuestionMessageRepository(db *sqlx.DB) *QuestionMessageRepository {
	return &QuestionMessageRepository{db: db}
}

func (r *QuestionMessageRepository) Create(ctx context.Context, questionID, senderID, senderRole, body string) (*model.QuestionMessage, error) {
	var m model.QuestionMessage
	err := r.db.GetContext(ctx, &m,
		`INSERT INTO question_messages (question_id, sender_id, sender_role, body)
		 VALUES ($1, $2, $3, $4) RETURNING *`,
		questionID, senderID, senderRole, body,
	)
	if err != nil {
		return nil, fmt.Errorf("message Create: %w", err)
	}
	return &m, nil
}

func (r *QuestionMessageRepository) ListByQuestion(ctx context.Context, questionID string) ([]model.QuestionMessage, error) {
	var messages []model.QuestionMessage
	err := r.db.SelectContext(ctx, &messages,
		`SELECT * FROM question_messages WHERE question_id = $1 ORDER BY created_at ASC`,
		questionID,
	)
	if err != nil {
		return nil, fmt.Errorf("message ListByQuestion: %w", err)
	}
	return messages, nil
}
