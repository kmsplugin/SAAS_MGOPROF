package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"mgoprof-saas/internal/model"
)

// UserRepository handles all DB operations for reg_users.
type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	err := r.db.GetContext(ctx, &u,
		`SELECT * FROM reg_users WHERE lower(email) = lower($1) LIMIT 1`, email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("user FindByEmail: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id int) (*model.User, error) {
	var u model.User
	err := r.db.GetContext(ctx, &u, `SELECT * FROM reg_users WHERE id = $1 LIMIT 1`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("user FindByID: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) Create(
	ctx context.Context,
	email string,
	req model.RegisterRequest,
	ip string,
	geo model.Geo,
	userAgent string,
	passwordHash string,
) (int, error) {
	var id int
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO reg_users
			(email, last_name, first_name, patronymic, organization, district,
			 is_union_member, union_ticket, extra_info,
			 last_ip, geo_country, geo_region, geo_city, user_agent,
			 password_hash, password_updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,NOW())
		 RETURNING id`,
		email,
		req.LastName, req.FirstName, req.Patronymic, req.Organization, req.District,
		req.IsUnionMember, req.UnionTicket, req.ExtraInfo,
		ip, geo.Country, geo.Region, geo.City, userAgent,
		passwordHash,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("user Create: %w", err)
	}
	return id, nil
}

func (r *UserRepository) Update(
	ctx context.Context,
	userID int,
	req model.RegisterRequest,
	ip string,
	geo model.Geo,
	userAgent string,
) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE reg_users
		 SET last_name=$1, first_name=$2, patronymic=$3, organization=$4,
		     district=$5, is_union_member=$6, union_ticket=$7, extra_info=$8,
		     last_ip=$9, geo_country=$10, geo_region=$11, geo_city=$12,
		     user_agent=$13, updated_at=NOW()
		 WHERE id=$14`,
		req.LastName, req.FirstName, req.Patronymic, req.Organization,
		req.District, req.IsUnionMember, req.UnionTicket, req.ExtraInfo,
		ip, geo.Country, geo.Region, geo.City,
		userAgent, userID,
	)
	if err != nil {
		return fmt.Errorf("user Update: %w", err)
	}
	return nil
}

func (r *UserRepository) SetPassword(ctx context.Context, userID int, hash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE reg_users
		 SET password_hash=$1, password_updated_at=NOW(), updated_at=NOW()
		 WHERE id=$2`,
		hash, userID,
	)
	if err != nil {
		return fmt.Errorf("user SetPassword: %w", err)
	}
	return nil
}

// WithTx runs fn inside a serializable transaction.
func (r *UserRepository) WithTx(ctx context.Context, fn func(ctx context.Context, tx *sqlx.Tx) error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if err := fn(ctx, tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// FindByEmailTx is the tx-scoped variant of FindByEmail.
func (r *UserRepository) FindByEmailTx(ctx context.Context, tx *sqlx.Tx, email string) (*model.User, error) {
	var u model.User
	err := tx.GetContext(ctx, &u,
		`SELECT * FROM reg_users WHERE lower(email) = lower($1) LIMIT 1`, email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("user FindByEmailTx: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) CreateTx(
	ctx context.Context,
	tx *sqlx.Tx,
	email string,
	req model.RegisterRequest,
	ip string,
	geo model.Geo,
	userAgent string,
	passwordHash string,
) (int, error) {
	var id int
	err := tx.QueryRowContext(ctx,
		`INSERT INTO reg_users
			(email, last_name, first_name, patronymic, organization, district,
			 is_union_member, union_ticket, extra_info,
			 last_ip, geo_country, geo_region, geo_city, user_agent,
			 password_hash, password_updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,NOW())
		 RETURNING id`,
		email,
		req.LastName, req.FirstName, req.Patronymic, req.Organization, req.District,
		req.IsUnionMember, req.UnionTicket, req.ExtraInfo,
		ip, geo.Country, geo.Region, geo.City, userAgent,
		passwordHash,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("user CreateTx: %w", err)
	}
	return id, nil
}

func (r *UserRepository) UpdateTx(
	ctx context.Context,
	tx *sqlx.Tx,
	userID int,
	req model.RegisterRequest,
	ip string,
	geo model.Geo,
	userAgent string,
) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE reg_users
		 SET last_name=$1, first_name=$2, patronymic=$3, organization=$4,
		     district=$5, is_union_member=$6, union_ticket=$7, extra_info=$8,
		     last_ip=$9, geo_country=$10, geo_region=$11, geo_city=$12,
		     user_agent=$13, updated_at=NOW()
		 WHERE id=$14`,
		req.LastName, req.FirstName, req.Patronymic, req.Organization,
		req.District, req.IsUnionMember, req.UnionTicket, req.ExtraInfo,
		ip, geo.Country, geo.Region, geo.City,
		userAgent, userID,
	)
	if err != nil {
		return fmt.Errorf("user UpdateTx: %w", err)
	}
	return nil
}

func (r *UserRepository) SetPasswordTx(ctx context.Context, tx *sqlx.Tx, userID int, hash string) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE reg_users
		 SET password_hash=$1, password_updated_at=NOW(), updated_at=NOW()
		 WHERE id=$2`,
		hash, userID,
	)
	if err != nil {
		return fmt.Errorf("user SetPasswordTx: %w", err)
	}
	return nil
}
