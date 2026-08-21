// user.go implements the UserRepository for PostgreSQL.
package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type UserRepo struct {
	pool Pool
}

func (r *UserRepo) Create(ctx context.Context, tenantID uuid.UUID, u *domain.User) error {
	u.ID = uuid.New()
	u.TenantID = tenantID

	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (id, tenant_id, email, name, first_name, last_name, phone, is_active, role_id, password_hash)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING created_at, updated_at`,
		u.ID, tenantID, u.Email, u.Name, u.FirstName, u.LastName, u.Phone, u.IsActive, u.RoleID, u.PasswordHash).
		Scan(&u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.User, error) {
	u := &domain.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, email, name, first_name, last_name, phone, is_active, role_id, created_at, updated_at
		 FROM users WHERE tenant_id = $1 AND id = $2`, tenantID, id).
		Scan(&u.ID, &u.TenantID, &u.Email, &u.Name, &u.FirstName, &u.LastName, &u.Phone, &u.IsActive, &u.RoleID, &u.CreatedAt, &u.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.User, error) {
	u := &domain.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, email, name, first_name, last_name, phone, is_active, role_id, password_hash, created_at, updated_at
		 FROM users WHERE tenant_id = $1 AND email = $2`, tenantID, email).
		Scan(&u.ID, &u.TenantID, &u.Email, &u.Name, &u.FirstName, &u.LastName, &u.Phone, &u.IsActive, &u.RoleID, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

func (r *UserRepo) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.User, int, error) {
	search := escapeILIKE(opts.Search)
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM users
		 WHERE tenant_id = $1 AND ($2 = '' OR first_name ILIKE '%' || $2 || '%' OR last_name ILIKE '%' || $2 || '%' OR email ILIKE '%' || $2 || '%')`,
		tenantID, search).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, email, name, first_name, last_name, phone, is_active, role_id, created_at, updated_at
		 FROM users
		 WHERE tenant_id = $1 AND ($2 = '' OR first_name ILIKE '%' || $2 || '%' OR last_name ILIKE '%' || $2 || '%' OR email ILIKE '%' || $2 || '%')
		 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		tenantID, search, opts.PerPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var results []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.TenantID, &u.Email, &u.Name, &u.FirstName, &u.LastName, &u.Phone, &u.IsActive, &u.RoleID, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		results = append(results, u)
	}
	return results, total, nil
}

func (r *UserRepo) Update(ctx context.Context, tenantID uuid.UUID, u *domain.User) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET email=$1, name=$2, first_name=$3, last_name=$4, phone=$5, is_active=$6, role_id=$7, password_hash=CASE WHEN $10 = '' THEN password_hash ELSE $10 END, updated_at=NOW()
		 WHERE tenant_id = $8 AND id = $9`,
		u.Email, u.Name, u.FirstName, u.LastName, u.Phone, u.IsActive, u.RoleID, tenantID, u.ID, u.PasswordHash)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (r *UserRepo) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM users WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (r *UserRepo) GetTenantsByEmail(ctx context.Context, email string) ([]domain.TenantSummary, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT t.id, t.name, t.subdomain
		 FROM users u
		 JOIN tenants t ON t.id = u.tenant_id
		 WHERE u.email = $1 AND u.is_active = true
		 ORDER BY t.name ASC`,
		email)
	if err != nil {
		return nil, fmt.Errorf("get tenants by email: %w", err)
	}
	defer rows.Close()

	var results []domain.TenantSummary
	for rows.Next() {
		var ts domain.TenantSummary
		if err := rows.Scan(&ts.ID, &ts.Name, &ts.Subdomain); err != nil {
			return nil, fmt.Errorf("scan tenant summary: %w", err)
		}
		results = append(results, ts)
	}
	return results, nil
}
