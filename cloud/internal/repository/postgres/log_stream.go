package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type LogStreamRepo struct {
	pool Pool
}

func (r *LogStreamRepo) Create(ctx context.Context, tenantID uuid.UUID, ls *domain.LogStream) error {
	ls.ID = uuid.New()
	ls.TenantID = tenantID
	config := ls.Config
	if config == nil {
		config = json.RawMessage(`{}`)
	}

	err := r.pool.QueryRow(ctx,
		`INSERT INTO log_streams (id, tenant_id, name, description, is_active, event_type, destination_type, config)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING created_at, updated_at`,
		ls.ID, tenantID, ls.Name, ls.Description, ls.IsActive, ls.EventType, ls.DestinationType, config).
		Scan(&ls.CreatedAt, &ls.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create log stream: %w", err)
	}
	return nil
}

func (r *LogStreamRepo) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.LogStream, error) {
	ls := &domain.LogStream{}
	var config []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, is_active, event_type, destination_type, config, created_at, updated_at
		 FROM log_streams WHERE tenant_id = $1 AND id = $2`, tenantID, id).
		Scan(&ls.ID, &ls.TenantID, &ls.Name, &ls.Description, &ls.IsActive, &ls.EventType, &ls.DestinationType, &config, &ls.CreatedAt, &ls.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get log stream: %w", err)
	}
	ls.Config = config
	return ls, nil
}

func (r *LogStreamRepo) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.LogStream, int, error) {
	search := escapeILIKE(opts.Search)
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM log_streams
		 WHERE tenant_id = $1 AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR description ILIKE '%' || $2 || '%')`,
		tenantID, search).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count log streams: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, description, is_active, event_type, destination_type, config, created_at, updated_at
		 FROM log_streams
		 WHERE tenant_id = $1 AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR description ILIKE '%' || $2 || '%')
		 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		tenantID, search, opts.PerPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list log streams: %w", err)
	}
	defer rows.Close()

	var results []domain.LogStream
	for rows.Next() {
		var ls domain.LogStream
		var config []byte
		if err := rows.Scan(&ls.ID, &ls.TenantID, &ls.Name, &ls.Description, &ls.IsActive, &ls.EventType, &ls.DestinationType, &config, &ls.CreatedAt, &ls.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan log stream: %w", err)
		}
		ls.Config = config
		results = append(results, ls)
	}
	return results, total, nil
}

func (r *LogStreamRepo) Update(ctx context.Context, tenantID uuid.UUID, ls *domain.LogStream) error {
	config := ls.Config
	if config == nil {
		config = json.RawMessage(`{}`)
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE log_streams SET name=$1, description=$2, is_active=$3, event_type=$4, destination_type=$5, config=$6, updated_at=NOW()
		 WHERE tenant_id = $7 AND id = $8`,
		ls.Name, ls.Description, ls.IsActive, ls.EventType, ls.DestinationType, config, tenantID, ls.ID)
	if err != nil {
		return fmt.Errorf("update log stream: %w", err)
	}
	return nil
}

func (r *LogStreamRepo) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM log_streams WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete log stream: %w", err)
	}
	return nil
}
