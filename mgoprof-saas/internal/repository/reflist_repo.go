package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"mgoprof-saas/internal/model"
)

// RefListRepository provides CRUD for reference lists and their items.
type RefListRepository struct {
	db *sqlx.DB
}

func NewRefListRepository(db *sqlx.DB) *RefListRepository {
	return &RefListRepository{db: db}
}

// ListAll returns all active reference lists.
func (r *RefListRepository) ListAll(ctx context.Context) ([]model.RefList, error) {
	var rows []model.RefList
	err := r.db.SelectContext(ctx, &rows,
		`SELECT id, slug, title, description, is_active, created_at, updated_at
		 FROM ref_lists ORDER BY title`)
	return rows, err
}

// GetByID returns a single list.
func (r *RefListRepository) GetByID(ctx context.Context, id int) (*model.RefList, error) {
	var row model.RefList
	err := r.db.GetContext(ctx, &row,
		`SELECT id, slug, title, description, is_active, created_at, updated_at
		 FROM ref_lists WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// GetBySlug returns a list by its slug.
func (r *RefListRepository) GetBySlug(ctx context.Context, slug string) (*model.RefList, error) {
	var row model.RefList
	err := r.db.GetContext(ctx, &row,
		`SELECT id, slug, title, description, is_active, created_at, updated_at
		 FROM ref_lists WHERE slug = $1`, slug)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// Create inserts a new reference list.
func (r *RefListRepository) Create(ctx context.Context, req model.CreateRefListRequest) (*model.RefList, error) {
	var row model.RefList
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO ref_lists (slug, title, description)
		VALUES ($1,$2,$3)
		RETURNING id, slug, title, description, is_active, created_at, updated_at`,
		req.Slug, req.Title, req.Description,
	).StructScan(&row)
	return &row, err
}

// Delete removes a reference list (cascades to items).
func (r *RefListRepository) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM ref_lists WHERE id = $1`, id)
	return err
}

// ListItems returns all items for a given list, ordered by sort_order.
func (r *RefListRepository) ListItems(ctx context.Context, listID int) ([]model.RefListItem, error) {
	var rows []model.RefListItem
	err := r.db.SelectContext(ctx, &rows, `
		SELECT id, list_id, value, label, sort_order, is_active
		FROM ref_list_items
		WHERE list_id = $1
		ORDER BY sort_order, label`, listID)
	return rows, err
}

// AddItem inserts a new item into a list.
func (r *RefListRepository) AddItem(ctx context.Context, listID int, req model.CreateRefListItemRequest) (*model.RefListItem, error) {
	var row model.RefListItem
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO ref_list_items (list_id, value, label, sort_order)
		VALUES ($1,$2,$3,$4)
		RETURNING id, list_id, value, label, sort_order, is_active`,
		listID, req.Value, req.Label, req.SortOrder,
	).StructScan(&row)
	return &row, err
}

// UpdateItem updates label and sort_order for an existing item.
func (r *RefListRepository) UpdateItem(ctx context.Context, itemID int, label string, sortOrder int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ref_list_items SET label = $1, sort_order = $2 WHERE id = $3`,
		label, sortOrder, itemID)
	return err
}

// DeleteItem removes a single item from a list.
func (r *RefListRepository) DeleteItem(ctx context.Context, itemID int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM ref_list_items WHERE id = $1`, itemID)
	return err
}

// UpdateListUpdatedAt bumps the updated_at timestamp on a list.
func (r *RefListRepository) UpdateListUpdatedAt(ctx context.Context, listID int) {
	now := time.Now()
	_, _ = r.db.ExecContext(ctx, `UPDATE ref_lists SET updated_at = $1 WHERE id = $2`, now, listID)
}
