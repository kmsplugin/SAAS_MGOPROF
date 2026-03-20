package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"mgoprof-saas/internal/model"
)

// QuestionRepository handles all DB operations for the Q&A module.
type QuestionRepository struct {
	db *sqlx.DB
}

func NewQuestionRepository(db *sqlx.DB) *QuestionRepository {
	return &QuestionRepository{db: db}
}

// Create inserts a new question and its opening message in a single TX.
func (r *QuestionRepository) Create(
	ctx context.Context,
	userID, eventID int,
	subject, body string,
) (*model.UserQuestion, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("question Create begin tx: %w", err)
	}

	var q model.UserQuestion
	err = tx.QueryRowContext(ctx,
		`INSERT INTO user_questions (user_id, event_id, subject)
		 VALUES ($1, $2, $3)
		 RETURNING *`,
		userID, eventID, subject,
	).Scan(
		&q.ID, &q.UserID, &q.EventID, &q.Subject, &q.Status, &q.Priority,
		&q.AssignedTo, &q.FirstReplyAt, &q.CreatedAt, &q.UpdatedAt, &q.ClosedAt,
	)
	if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("question Create insert: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO question_messages (question_id, sender_id, sender_role, body)
		 VALUES ($1, $2, 'user', $3)`,
		q.ID, userID, body,
	)
	if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("question Create first message: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("question Create commit: %w", err)
	}
	return &q, nil
}

// ListByUser returns all questions for a given user, newest first.
func (r *QuestionRepository) ListByUser(ctx context.Context, userID int) ([]model.UserQuestionWithMeta, error) {
	var rows []model.UserQuestionWithMeta
	err := r.db.SelectContext(ctx, &rows, `
		SELECT
			q.*,
			e.title                                                  AS event_title,
			u.email                                                  AS user_email,
			u.last_name || ' ' || u.first_name                      AS user_name,
			COUNT(m.id) FILTER (WHERE m.is_read = FALSE
			                    AND  m.sender_role = 'admin')        AS unread_count
		FROM   user_questions q
		JOIN   reg_events e ON e.id = q.event_id
		JOIN   reg_users  u ON u.id = q.user_id
		LEFT   JOIN question_messages m ON m.question_id = q.id
		WHERE  q.user_id = $1
		GROUP  BY q.id, e.title, u.email, u.last_name, u.first_name
		ORDER  BY q.created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("question ListByUser: %w", err)
	}
	return rows, nil
}

// ListAll returns all questions (admin view) with optional status filter.
func (r *QuestionRepository) ListAll(ctx context.Context, status, eventID string) ([]model.UserQuestionWithMeta, error) {
	query := `
		SELECT
			q.*,
			e.title                                             AS event_title,
			u.email                                             AS user_email,
			u.last_name || ' ' || u.first_name                 AS user_name,
			COUNT(m.id) FILTER (WHERE m.is_read = FALSE
			                    AND  m.sender_role = 'user')   AS unread_count
		FROM   user_questions q
		JOIN   reg_events e ON e.id = q.event_id
		JOIN   reg_users  u ON u.id = q.user_id
		LEFT   JOIN question_messages m ON m.question_id = q.id`

	args := []interface{}{}
	where := []string{}
	idx := 1

	if status != "" && status != "all" {
		where = append(where, fmt.Sprintf("q.status = $%d", idx))
		args = append(args, status)
		idx++
	}
	if eventID != "" && eventID != "0" {
		where = append(where, fmt.Sprintf("q.event_id = $%d", idx))
		args = append(args, eventID)
		idx++
	}

	if len(where) > 0 {
		query += " WHERE " + where[0]
		for _, w := range where[1:] {
			query += " AND " + w
		}
	}

	query += `
		GROUP  BY q.id, e.title, u.email, u.last_name, u.first_name
		ORDER  BY q.priority DESC, q.created_at DESC`

	var rows []model.UserQuestionWithMeta
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("question ListAll: %w", err)
	}
	return rows, nil
}

// GetByID returns a single question (checks ownership if userID > 0).
func (r *QuestionRepository) GetByID(ctx context.Context, id, userID int) (*model.UserQuestion, error) {
	var q model.UserQuestion
	query := `SELECT * FROM user_questions WHERE id = $1`
	args := []interface{}{id}
	if userID > 0 {
		query += ` AND user_id = $2`
		args = append(args, userID)
	}
	err := r.db.GetContext(ctx, &q, query, args...)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("question GetByID: %w", err)
	}
	return &q, nil
}

// ListMessages returns all messages for a question, ordered chronologically.
func (r *QuestionRepository) ListMessages(ctx context.Context, questionID int) ([]model.QuestionMessage, error) {
	var msgs []model.QuestionMessage
	err := r.db.SelectContext(ctx, &msgs,
		`SELECT * FROM question_messages WHERE question_id = $1 ORDER BY created_at ASC`,
		questionID)
	if err != nil {
		return nil, fmt.Errorf("question ListMessages: %w", err)
	}
	return msgs, nil
}

// AddMessage appends a message and updates the question's updated_at.
// If it is the first admin reply, sets first_reply_at.
func (r *QuestionRepository) AddMessage(
	ctx context.Context,
	questionID, senderID int,
	senderRole, body string,
) (*model.QuestionMessage, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("question AddMessage begin tx: %w", err)
	}

	var msg model.QuestionMessage
	err = tx.QueryRowContext(ctx,
		`INSERT INTO question_messages (question_id, sender_id, sender_role, body)
		 VALUES ($1, $2, $3, $4)
		 RETURNING *`,
		questionID, senderID, senderRole, body,
	).Scan(&msg.ID, &msg.QuestionID, &msg.SenderID, &msg.SenderRole, &msg.Body, &msg.IsRead, &msg.CreatedAt)
	if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("question AddMessage insert: %w", err)
	}

	// Auto-advance status and record first reply time for admin messages.
	if senderRole == "admin" {
		_, err = tx.ExecContext(ctx,
			`UPDATE user_questions
			 SET updated_at     = NOW(),
			     first_reply_at = COALESCE(first_reply_at, NOW()),
			     status         = CASE WHEN status = 'new' THEN 'in_progress'
			                          ELSE status END
			 WHERE id = $1`, questionID)
	} else {
		_, err = tx.ExecContext(ctx,
			`UPDATE user_questions SET updated_at = NOW() WHERE id = $1`, questionID)
	}
	if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("question AddMessage update: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("question AddMessage commit: %w", err)
	}
	return &msg, nil
}

// MarkRead marks all unread messages from the given sender_role as read.
func (r *QuestionRepository) MarkRead(ctx context.Context, questionID int, readerRole string) error {
	// User reads → mark admin messages as read; Admin reads → mark user messages as read.
	senderRole := "admin"
	if readerRole == "admin" {
		senderRole = "user"
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE question_messages
		 SET is_read = TRUE
		 WHERE question_id = $1 AND sender_role = $2 AND is_read = FALSE`,
		questionID, senderRole)
	if err != nil {
		return fmt.Errorf("question MarkRead: %w", err)
	}
	return nil
}

// UpdateStatus changes question status and writes to the status log.
func (r *QuestionRepository) UpdateStatus(
	ctx context.Context,
	questionID, adminID int,
	newStatus, comment string,
) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("question UpdateStatus begin tx: %w", err)
	}

	var oldStatus string
	if err := tx.QueryRowContext(ctx,
		`SELECT status FROM user_questions WHERE id = $1`, questionID,
	).Scan(&oldStatus); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("question UpdateStatus fetch: %w", err)
	}

	closedAt := "NULL"
	if newStatus == "closed" {
		closedAt = "NOW()"
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE user_questions
		 SET status = $1, updated_at = NOW(), closed_at = `+closedAt+`
		 WHERE id = $2`,
		newStatus, questionID)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("question UpdateStatus update: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO question_status_log (question_id, old_status, new_status, changed_by, comment)
		 VALUES ($1, $2, $3, $4, $5)`,
		questionID, oldStatus, newStatus, adminID, comment)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("question UpdateStatus log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("question UpdateStatus commit: %w", err)
	}
	return nil
}

// Assign assigns a question to an admin operator.
func (r *QuestionRepository) Assign(ctx context.Context, questionID, adminID int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE user_questions SET assigned_to = $1, updated_at = NOW() WHERE id = $2`,
		adminID, questionID)
	if err != nil {
		return fmt.Errorf("question Assign: %w", err)
	}
	return nil
}

// Stats returns aggregate Q&A metrics.
func (r *QuestionRepository) Stats(ctx context.Context, eventID int) (*model.QuestionStats, error) {
	query := `
		SELECT
			COUNT(*)                                                      AS total,
			COUNT(*) FILTER (WHERE status = 'new')                       AS new_count,
			COUNT(*) FILTER (WHERE status = 'in_progress')               AS in_progress,
			COUNT(*) FILTER (WHERE status = 'answered')                  AS answered,
			COUNT(*) FILTER (WHERE status = 'closed')                    AS closed,
			COALESCE(
				AVG(EXTRACT(EPOCH FROM (first_reply_at - created_at))/60)
				FILTER (WHERE first_reply_at IS NOT NULL),
				0
			)                                                             AS avg_reply_min
		FROM user_questions`

	var args []interface{}
	if eventID > 0 {
		query += ` WHERE event_id = $1`
		args = append(args, eventID)
	}

	var s model.QuestionStats
	if err := r.db.GetContext(ctx, &s, query, args...); err != nil {
		return nil, fmt.Errorf("question Stats: %w", err)
	}
	return &s, nil
}

// QueueNotification inserts a pending email notification.
func (r *QuestionRepository) QueueNotification(ctx context.Context, questionID, userID int, notifType string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO question_notifications (question_id, user_id, type)
		 VALUES ($1, $2, $3)`,
		questionID, userID, notifType)
	if err != nil {
		return fmt.Errorf("question QueueNotification: %w", err)
	}
	return nil
}

// PendingNotifications returns unsent notifications (for the background mailer worker).
func (r *QuestionRepository) PendingNotifications(ctx context.Context, limit int) ([]struct {
	ID         int
	QuestionID int
	UserID     int
	Type       string
	Subject    string
	UserEmail  string
	UserName   string
}, error) {
	type row struct {
		ID         int    `db:"id"`
		QuestionID int    `db:"question_id"`
		UserID     int    `db:"user_id"`
		Type       string `db:"type"`
		Subject    string `db:"subject"`
		UserEmail  string `db:"user_email"`
		UserName   string `db:"user_name"`
	}
	var rows []row
	err := r.db.SelectContext(ctx, &rows, `
		SELECT n.id, n.question_id, n.user_id, n.type,
		       q.subject,
		       u.email AS user_email,
		       u.first_name || ' ' || u.last_name AS user_name
		FROM   question_notifications n
		JOIN   user_questions q ON q.id = n.question_id
		JOIN   reg_users      u ON u.id = n.user_id
		WHERE  n.sent_at IS NULL
		ORDER  BY n.created_at ASC
		LIMIT  $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("question PendingNotifications: %w", err)
	}

	result := make([]struct {
		ID         int
		QuestionID int
		UserID     int
		Type       string
		Subject    string
		UserEmail  string
		UserName   string
	}, len(rows))
	for i, r := range rows {
		result[i].ID = r.ID
		result[i].QuestionID = r.QuestionID
		result[i].UserID = r.UserID
		result[i].Type = r.Type
		result[i].Subject = r.Subject
		result[i].UserEmail = r.UserEmail
		result[i].UserName = r.UserName
	}
	return result, nil
}

// MarkNotificationSent marks a notification as delivered.
func (r *QuestionRepository) MarkNotificationSent(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE question_notifications SET sent_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("question MarkNotificationSent: %w", err)
	}
	return nil
}

// ForExport returns all questions with metadata for Excel/CSV export.
func (r *QuestionRepository) ForExport(ctx context.Context, eventID int) ([]struct {
	ID              int
	EventTitle      string
	UserEmail       string
	UserName        string
	Subject         string
	Status          string
	CreatedAt       time.Time
	FirstReplyAt    *time.Time
	ClosedAt        *time.Time
	ReplyMinutes    float64
	AssignedName    string
}, error) {
	type row struct {
		ID           int        `db:"id"`
		EventTitle   string     `db:"event_title"`
		UserEmail    string     `db:"user_email"`
		UserName     string     `db:"user_name"`
		Subject      string     `db:"subject"`
		Status       string     `db:"status"`
		CreatedAt    time.Time  `db:"created_at"`
		FirstReplyAt *time.Time `db:"first_reply_at"`
		ClosedAt     *time.Time `db:"closed_at"`
		ReplyMinutes float64    `db:"reply_minutes"`
		AssignedName string     `db:"assigned_name"`
	}
	query := `
		SELECT q.id, e.title AS event_title,
		       u.email AS user_email,
		       u.last_name || ' ' || u.first_name AS user_name,
		       q.subject, q.status, q.created_at,
		       q.first_reply_at, q.closed_at,
		       COALESCE(
		           EXTRACT(EPOCH FROM (q.first_reply_at - q.created_at))/60, 0
		       ) AS reply_minutes,
		       COALESCE(a.name, '') AS assigned_name
		FROM   user_questions q
		JOIN   reg_events  e ON e.id = q.event_id
		JOIN   reg_users   u ON u.id = q.user_id
		LEFT   JOIN admin_users a ON a.id = q.assigned_to`
	args := []interface{}{}
	if eventID > 0 {
		query += ` WHERE q.event_id = $1`
		args = append(args, eventID)
	}
	query += ` ORDER BY q.created_at DESC`

	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("question ForExport: %w", err)
	}

	result := make([]struct {
		ID           int
		EventTitle   string
		UserEmail    string
		UserName     string
		Subject      string
		Status       string
		CreatedAt    time.Time
		FirstReplyAt *time.Time
		ClosedAt     *time.Time
		ReplyMinutes float64
		AssignedName string
	}, len(rows))
	for i, r := range rows {
		result[i].ID = r.ID
		result[i].EventTitle = r.EventTitle
		result[i].UserEmail = r.UserEmail
		result[i].UserName = r.UserName
		result[i].Subject = r.Subject
		result[i].Status = r.Status
		result[i].CreatedAt = r.CreatedAt
		result[i].FirstReplyAt = r.FirstReplyAt
		result[i].ClosedAt = r.ClosedAt
		result[i].ReplyMinutes = r.ReplyMinutes
		result[i].AssignedName = r.AssignedName
	}
	return result, nil
}
