package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"platform/api/internal/model"
)

type TenantRepository struct{ db *sqlx.DB }

func NewTenantRepository(db *sqlx.DB) *TenantRepository { return &TenantRepository{db: db} }

func (r *TenantRepository) FindBySlug(ctx context.Context, slug string) (*model.Tenant, error) {
	var t model.Tenant
	err := r.db.GetContext(ctx, &t,
		`SELECT * FROM tenants WHERE slug = $1 AND is_active = TRUE LIMIT 1`, slug)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("tenant FindBySlug: %w", err)
	}
	return &t, nil
}

func (r *TenantRepository) FindByID(ctx context.Context, id string) (*model.Tenant, error) {
	var t model.Tenant
	err := r.db.GetContext(ctx, &t, `SELECT * FROM tenants WHERE id = $1 LIMIT 1`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("tenant FindByID: %w", err)
	}
	return &t, nil
}

func (r *TenantRepository) Create(ctx context.Context, slug, name, plan string) (*model.Tenant, error) {
	var t model.Tenant
	err := r.db.GetContext(ctx, &t,
		`INSERT INTO tenants (slug, name, plan) VALUES ($1, $2, $3) RETURNING *`,
		slug, name, plan,
	)
	if err != nil {
		return nil, fmt.Errorf("tenant Create: %w", err)
	}
	return &t, nil
}
