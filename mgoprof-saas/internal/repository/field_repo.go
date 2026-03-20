package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"

	"mgoprof-saas/internal/model"
)

// FieldRepository manages custom event registration fields and answers.
type FieldRepository struct {
	db *sqlx.DB
}

func NewFieldRepository(db *sqlx.DB) *FieldRepository {
	return &FieldRepository{db: db}
}

// --- Event Fields ---

// ListByEvent returns all fields for an event ordered by sort_order.
func (r *FieldRepository) ListByEvent(ctx context.Context, eventID int) ([]model.EventField, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, event_id, label, field_type, COALESCE(options,'[]'::jsonb), placeholder, is_required, sort_order, created_at
		FROM event_fields
		WHERE event_id = $1
		ORDER BY sort_order, id`, eventID)
	if err != nil {
		return nil, fmt.Errorf("list fields: %w", err)
	}
	defer rows.Close()
	return scanFields(rows)
}

// GetByID returns one field or nil.
func (r *FieldRepository) GetByID(ctx context.Context, id int) (*model.EventField, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, event_id, label, field_type, COALESCE(options,'[]'::jsonb), placeholder, is_required, sort_order, created_at
		FROM event_fields
		WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("get field: %w", err)
	}
	defer rows.Close()
	fields, err := scanFields(rows)
	if err != nil || len(fields) == 0 {
		return nil, err
	}
	return &fields[0], nil
}

// Create inserts a new EventField and returns it with the generated ID.
func (r *FieldRepository) Create(ctx context.Context, f model.EventField) (*model.EventField, error) {
	optJSON, err := json.Marshal(f.Options)
	if err != nil {
		return nil, fmt.Errorf("marshal options: %w", err)
	}
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO event_fields (event_id, label, field_type, options, placeholder, is_required, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`,
		f.EventID, f.Label, f.FieldType, optJSON, f.Placeholder, f.IsRequired, f.SortOrder,
	).Scan(&f.ID, &f.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create field: %w", err)
	}
	return &f, nil
}

// Update modifies an existing EventField.
func (r *FieldRepository) Update(ctx context.Context, f model.EventField) (*model.EventField, error) {
	optJSON, err := json.Marshal(f.Options)
	if err != nil {
		return nil, fmt.Errorf("marshal options: %w", err)
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE event_fields
		SET label=$1, field_type=$2, options=$3, placeholder=$4, is_required=$5, sort_order=$6
		WHERE id=$7 AND event_id=$8`,
		f.Label, f.FieldType, optJSON, f.Placeholder, f.IsRequired, f.SortOrder, f.ID, f.EventID,
	)
	if err != nil {
		return nil, fmt.Errorf("update field: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, sql.ErrNoRows
	}
	return &f, nil
}

// Delete removes a field by ID (answers cascade via FK).
func (r *FieldRepository) Delete(ctx context.Context, id, eventID int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM event_fields WHERE id=$1 AND event_id=$2`, id, eventID)
	if err != nil {
		return fmt.Errorf("delete field: %w", err)
	}
	return nil
}

// --- Answers ---

// SaveAnswersTx stores all answers for one registration inside a TX.
func (r *FieldRepository) SaveAnswersTx(ctx context.Context, tx *sqlx.Tx, regID int, answers []model.FieldAnswer) error {
	for _, a := range answers {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO reg_answers (registration_id, field_id, value)
			VALUES ($1, $2, $3)
			ON CONFLICT (registration_id, field_id) DO UPDATE SET value = EXCLUDED.value`,
			regID, a.FieldID, a.Value,
		)
		if err != nil {
			return fmt.Errorf("save answer field %d: %w", a.FieldID, err)
		}
	}
	return nil
}

// ListAnswersByRegistration returns all answers for a registration.
func (r *FieldRepository) ListAnswersByRegistration(ctx context.Context, regID int) ([]model.FieldAnswer, error) {
	var rows []model.FieldAnswer
	err := r.db.SelectContext(ctx, &rows, `
		SELECT id, registration_id, field_id, value
		FROM reg_answers
		WHERE registration_id = $1
		ORDER BY field_id`, regID)
	if err != nil {
		return nil, fmt.Errorf("list answers: %w", err)
	}
	return rows, nil
}

// --- helpers ---

// scanFields reads rows from event_fields, handling the JSONB options column.
func scanFields(rows *sql.Rows) ([]model.EventField, error) {
	var result []model.EventField
	for rows.Next() {
		var f model.EventField
		var optRaw []byte
		if err := rows.Scan(&f.ID, &f.EventID, &f.Label, &f.FieldType, &optRaw,
			&f.Placeholder, &f.IsRequired, &f.SortOrder, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan field: %w", err)
		}
		if len(optRaw) > 0 {
			_ = json.Unmarshal(optRaw, &f.Options)
		}
		result = append(result, f)
	}
	return result, rows.Err()
}
