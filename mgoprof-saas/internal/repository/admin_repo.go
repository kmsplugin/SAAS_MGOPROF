package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"mgoprof-saas/internal/model"
)

// AdminRepository manages admin_users, permissions, role_permissions and audit logs.
type AdminRepository struct {
	db *sqlx.DB
}

func NewAdminRepository(db *sqlx.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

// ── Admin user CRUD ────────────────────────────────────────────────────────

func (r *AdminRepository) FindByEmail(ctx context.Context, email string) (*model.AdminUser, error) {
	var u model.AdminUser
	err := r.db.GetContext(ctx, &u,
		`SELECT * FROM admin_users WHERE lower(email) = lower($1) AND is_active = TRUE LIMIT 1`, email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("admin FindByEmail: %w", err)
	}
	return &u, nil
}

func (r *AdminRepository) FindByID(ctx context.Context, id int) (*model.AdminUser, error) {
	var u model.AdminUser
	err := r.db.GetContext(ctx, &u, `SELECT * FROM admin_users WHERE id = $1 LIMIT 1`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("admin FindByID: %w", err)
	}
	return &u, nil
}

func (r *AdminRepository) ListAll(ctx context.Context) ([]model.AdminUser, error) {
	var rows []model.AdminUser
	err := r.db.SelectContext(ctx, &rows,
		`SELECT * FROM admin_users ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("admin ListAll: %w", err)
	}
	return rows, nil
}

func (r *AdminRepository) Create(ctx context.Context, req model.CreateAdminRequest, createdBy int, hash string) (int, error) {
	var id int
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO admin_users (email, name, password_hash, role, created_by)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		req.Email, req.Name, hash, req.Role, createdBy,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("admin Create: %w", err)
	}
	return id, nil
}

func (r *AdminRepository) Update(ctx context.Context, id int, req model.UpdateAdminUserRequest) error {
	if req.Name == "" && req.Role == "" && req.IsActive == nil {
		return nil
	}
	query := `UPDATE admin_users SET updated_at = NOW()`
	args := []interface{}{}
	idx := 1
	if req.Name != "" {
		query += fmt.Sprintf(", name = $%d", idx)
		args = append(args, req.Name)
		idx++
	}
	if req.Role != "" {
		query += fmt.Sprintf(", role = $%d", idx)
		args = append(args, req.Role)
		idx++
	}
	if req.IsActive != nil {
		query += fmt.Sprintf(", is_active = $%d", idx)
		args = append(args, *req.IsActive)
		idx++
	}
	query += fmt.Sprintf(" WHERE id = $%d", idx)
	args = append(args, id)

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("admin Update: %w", err)
	}
	return nil
}

func (r *AdminRepository) SetLastLogin(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE admin_users SET last_login_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("admin SetLastLogin: %w", err)
	}
	return nil
}

// ── Permissions ────────────────────────────────────────────────────────────

// PermissionsForRole returns all permission keys granted to a role.
func (r *AdminRepository) PermissionsForRole(ctx context.Context, role string) (map[string]bool, error) {
	var keys []string
	err := r.db.SelectContext(ctx, &keys, `
		SELECT p.key
		FROM   role_permissions rp
		JOIN   permissions      p ON p.id = rp.permission_id
		WHERE  rp.role = $1`, role)
	if err != nil {
		return nil, fmt.Errorf("admin PermissionsForRole: %w", err)
	}
	m := make(map[string]bool, len(keys))
	for _, k := range keys {
		m[k] = true
	}
	return m, nil
}

// HasPermission returns true if the admin has the given permission key.
func (r *AdminRepository) HasPermission(ctx context.Context, adminID int, permKey string) (bool, error) {
	admin, err := r.FindByID(ctx, adminID)
	if err != nil || admin == nil {
		return false, err
	}
	if admin.Role == "super_admin" {
		return true, nil
	}
	var count int
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM   role_permissions rp
		JOIN   permissions      p ON p.id = rp.permission_id
		WHERE  rp.role = $1 AND p.key = $2`, admin.Role, permKey,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("admin HasPermission: %w", err)
	}
	return count > 0, nil
}

// ── Audit log ──────────────────────────────────────────────────────────────

// WriteAuditLog records an admin action. Never fails silently — callers should log the error.
func (r *AdminRepository) WriteAuditLog(
	ctx context.Context,
	adminID int,
	adminEmail, action, targetType string,
	targetID *int,
	oldValue, newValue interface{},
	ip string,
) error {
	oldJSON, _ := json.Marshal(oldValue)
	newJSON, _ := json.Marshal(newValue)

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO admin_action_logs
		   (admin_id, admin_email, action, target_type, target_id, old_value, new_value, ip)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		adminID, adminEmail, action, targetType, targetID,
		oldJSON, newJSON, ip,
	)
	if err != nil {
		return fmt.Errorf("admin WriteAuditLog: %w", err)
	}
	return nil
}

// ListAuditLogs returns audit log entries (super_admin only, paginated).
func (r *AdminRepository) ListAuditLogs(ctx context.Context, adminID int, limit, offset int) ([]model.AdminActionLog, error) {
	query := `SELECT * FROM admin_action_logs`
	args := []interface{}{}
	idx := 1
	if adminID > 0 {
		query += fmt.Sprintf(" WHERE admin_id = $%d", idx)
		args = append(args, adminID)
		idx++
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", idx, idx+1)
	args = append(args, limit, offset)

	var rows []model.AdminActionLog
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("admin ListAuditLogs: %w", err)
	}
	return rows, nil
}

// ── User profile change log ─────────────────────────────────────────────────

// WriteProfileChangeLog records a field-level change to a user's profile.
func (r *AdminRepository) WriteProfileChangeLog(
	ctx context.Context,
	userID int,
	changedByAdminID *int,
	fieldName, oldValue, newValue string,
) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_profile_change_logs (user_id, changed_by, field_name, old_value, new_value)
		 VALUES ($1, $2, $3, $4, $5)`,
		userID, changedByAdminID, fieldName, oldValue, newValue,
	)
	if err != nil {
		return fmt.Errorf("admin WriteProfileChangeLog: %w", err)
	}
	return nil
}

// ListProfileChangeLogs returns the change history for a user.
func (r *AdminRepository) ListProfileChangeLogs(ctx context.Context, userID, limit int) ([]model.UserProfileChangeLog, error) {
	var rows []model.UserProfileChangeLog
	err := r.db.SelectContext(ctx, &rows, `
		SELECT * FROM user_profile_change_logs
		WHERE  user_id = $1
		ORDER  BY changed_at DESC
		LIMIT  $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("admin ListProfileChangeLogs: %w", err)
	}
	return rows, nil
}

// ── Export jobs ────────────────────────────────────────────────────────────

type ExportJob struct {
	ID        int        `db:"id"         json:"id"`
	AdminID   *int       `db:"admin_id"   json:"admin_id,omitempty"`
	Type      string     `db:"type"       json:"type"`
	Status    string     `db:"status"     json:"status"`
	FileURL   *string    `db:"file_url"   json:"file_url,omitempty"`
	Error     *string    `db:"error"      json:"error,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	DoneAt    *time.Time `db:"done_at"    json:"done_at,omitempty"`
}

func (r *AdminRepository) CreateExportJob(ctx context.Context, adminID int, jobType string, params interface{}) (int, error) {
	paramsJSON, _ := json.Marshal(params)
	var id int
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO export_jobs (admin_id, type, params, expires_at)
		 VALUES ($1, $2, $3, NOW() + INTERVAL '24 hours') RETURNING id`,
		adminID, jobType, paramsJSON,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("admin CreateExportJob: %w", err)
	}
	return id, nil
}

func (r *AdminRepository) CompleteExportJob(ctx context.Context, id int, fileURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE export_jobs
		 SET status = 'done', file_url = $1, done_at = NOW()
		 WHERE id = $2`,
		fileURL, id,
	)
	if err != nil {
		return fmt.Errorf("admin CompleteExportJob: %w", err)
	}
	return nil
}

func (r *AdminRepository) FailExportJob(ctx context.Context, id int, errMsg string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE export_jobs SET status = 'failed', error = $1, done_at = NOW() WHERE id = $2`,
		errMsg, id,
	)
	if err != nil {
		return fmt.Errorf("admin FailExportJob: %w", err)
	}
	return nil
}
