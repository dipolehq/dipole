package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type AccessConditionRepo struct {
	pool Pool
}

func (r *AccessConditionRepo) Create(ctx context.Context, tenantID uuid.UUID, ac *domain.AccessCondition) error {
	ac.ID = uuid.New()
	ac.TenantID = tenantID
	tags, _ := json.Marshal(ac.Tags)

	err := r.pool.QueryRow(ctx,
		`INSERT INTO access_conditions (id, tenant_id, name, description, is_active, type, config, tags, resource_set_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING created_at, updated_at`,
		ac.ID, tenantID, ac.Name, ac.Description, ac.IsActive, ac.Type, ac.Config, tags, ac.ResourceSetID).
		Scan(&ac.CreatedAt, &ac.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create access condition: %w", err)
	}
	return nil
}

func (r *AccessConditionRepo) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.AccessCondition, error) {
	ac := &domain.AccessCondition{}
	var tags []byte

	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, is_active, type, config, tags, resource_set_id, created_at, updated_at
		 FROM access_conditions WHERE tenant_id = $1 AND id = $2`, tenantID, id).
		Scan(&ac.ID, &ac.TenantID, &ac.Name, &ac.Description, &ac.IsActive, &ac.Type, &ac.Config, &tags, &ac.ResourceSetID, &ac.CreatedAt, &ac.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get access condition: %w", err)
	}

	json.Unmarshal(tags, &ac.Tags)
	return ac, nil
}

func (r *AccessConditionRepo) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.AccessCondition, int, error) {
	countArgs := []interface{}{tenantID, opts.Search}
	n := 2
	rsFilter := ResourceSetFilter(ctx, &countArgs, &n)

	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM access_conditions
		 WHERE tenant_id = $1 AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR description ILIKE '%' || $2 || '%')`+rsFilter,
		countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count access conditions: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	listArgs := []interface{}{tenantID, opts.Search}
	n = 2
	rsFilter = ResourceSetFilter(ctx, &listArgs, &n)
	n++
	listArgs = append(listArgs, opts.PerPage)
	limitParam := fmt.Sprintf("$%d", n)
	n++
	listArgs = append(listArgs, offset)
	offsetParam := fmt.Sprintf("$%d", n)

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, description, is_active, type, config, tags, resource_set_id, created_at, updated_at,
		        (SELECT COUNT(*) FROM policy_access_conditions pac WHERE pac.access_condition_id = access_conditions.id) AS policy_count
		 FROM access_conditions
		 WHERE tenant_id = $1 AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR description ILIKE '%' || $2 || '%')`+rsFilter+`
		 ORDER BY created_at DESC LIMIT `+limitParam+` OFFSET `+offsetParam,
		listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list access conditions: %w", err)
	}
	defer rows.Close()

	var results []domain.AccessCondition
	for rows.Next() {
		var ac domain.AccessCondition
		var tags []byte
		if err := rows.Scan(&ac.ID, &ac.TenantID, &ac.Name, &ac.Description, &ac.IsActive, &ac.Type, &ac.Config, &tags, &ac.ResourceSetID, &ac.CreatedAt, &ac.UpdatedAt, &ac.PolicyCount); err != nil {
			return nil, 0, fmt.Errorf("scan access condition: %w", err)
		}
		json.Unmarshal(tags, &ac.Tags)
		results = append(results, ac)
	}
	return results, total, nil
}

func (r *AccessConditionRepo) Update(ctx context.Context, tenantID uuid.UUID, ac *domain.AccessCondition) error {
	tags, _ := json.Marshal(ac.Tags)

	err := r.pool.QueryRow(ctx,
		`UPDATE access_conditions SET name=$1, description=$2, is_active=$3, type=$4, config=$5, tags=$6, resource_set_id=$7, updated_at=NOW()
		 WHERE tenant_id = $8 AND id = $9
		 RETURNING created_at, updated_at`,
		ac.Name, ac.Description, ac.IsActive, ac.Type, ac.Config, tags, ac.ResourceSetID, tenantID, ac.ID).
		Scan(&ac.CreatedAt, &ac.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update access condition: %w", err)
	}
	return nil
}

func (r *AccessConditionRepo) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM access_conditions WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete access condition: %w", err)
	}
	return nil
}
