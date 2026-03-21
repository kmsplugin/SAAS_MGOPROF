package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"platform/api/internal/model"
)

type UserRepository struct{ db *sqlx.DB }

func NewUserRepository(db *sqlx.DB) *UserRepository { return &UserRepository{db: db} }

func (r *UserRepository) FindByEmail(ctx context.Context, tenantID, email string) (*model.User, error) {
	var u model.User
	err := r.db.GetContext(ctx, &u,
		`SELECT * FROM users WHERE tenant_id = $1 AND lower(email) = lower($2) LIMIT 1`,
		tenantID, email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("user FindByEmail: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) FindByID(ctx context.Context, tenantID, id string) (*model.User, error) {
	var u model.User
	err := r.db.GetContext(ctx, &u,
		`SELECT * FROM users WHERE id = $1 AND tenant_id = $2 LIMIT 1`, id, tenantID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("user FindByID: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, tenantID, email, firstName, lastName, passwordHash, role string) (*model.User, error) {
	var u model.User
	err := r.db.GetContext(ctx, &u,
		`INSERT INTO users (tenant_id, email, first_name, last_name, password_hash, role)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING *`,
		tenantID, strings.ToLower(email), firstName, lastName, passwordHash, role,
	)
	if err != nil {
		return nil, fmt.Errorf("user Create: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) SetLastLogin(ctx context.Context, tenantID, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET last_login_at = NOW() WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return fmt.Errorf("user SetLastLogin: %w", err)
	}
	return nil
}

func (r *UserRepository) UpdateProfile(ctx context.Context, tenantID, id, firstName, lastName, avatarURL string) (*model.User, error) {
	var u model.User
	err := r.db.GetContext(ctx, &u,
		`UPDATE users SET first_name = $1, last_name = $2, avatar_url = $3
		 WHERE id = $4 AND tenant_id = $5 RETURNING *`,
		firstName, lastName, avatarURL, id, tenantID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("user UpdateProfile: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]model.User, error) {
	var users []model.User
	err := r.db.SelectContext(ctx, &users,
		`SELECT * FROM users WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("user ListByTenant: %w", err)
	}
	return users, nil
}
