package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type EventRepo struct {
	pool Pool
}

func (r *EventRepo) InsertAuthEvent(ctx context.Context, tenantID uuid.UUID, e *domain.AuthEvent) error {
	e.ID = uuid.New()
	e.TenantID = tenantID

	_, err := r.pool.Exec(ctx,
		`INSERT INTO access_authorization_events (id, tenant_id, event_type, context_id, severity, client_ip, timestamp, payload, resource_set_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		e.ID, tenantID, e.EventType, e.ContextID, e.Severity, e.ClientIP, e.Timestamp, e.Payload, e.ResourceSetID)
	if err != nil {
		return fmt.Errorf("insert auth event: %w", err)
	}
	return nil
}

func (r *EventRepo) InsertWorkloadEvent(ctx context.Context, tenantID uuid.UUID, e *domain.WorkloadEvent) error {
	e.ID = uuid.New()
	e.TenantID = tenantID

	_, err := r.pool.Exec(ctx,
		`INSERT INTO workload_events (id, tenant_id, connection_id, event_type, policy_id, client_workload_id, server_workload_id, outcome, severity, timestamp, payload, resource_set_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		e.ID, tenantID, e.ConnectionID, e.EventType, e.PolicyID, e.ClientWorkloadID, e.ServerWorkloadID, e.Outcome, e.Severity, e.Timestamp, e.Payload, e.ResourceSetID)
	if err != nil {
		return fmt.Errorf("insert workload event: %w", err)
	}
	return nil
}

func (r *EventRepo) InsertAuditLog(ctx context.Context, tenantID uuid.UUID, l *domain.AuditLog) error {
	l.ID = uuid.New()
	l.TenantID = tenantID

	_, err := r.pool.Exec(ctx,
		`INSERT INTO audit_logs (id, tenant_id, actor_id, actor_email, category, activity, target_type, target_id, target_name, severity, client_ip, timestamp)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		l.ID, tenantID, l.ActorID, l.ActorEmail, l.Category, l.Activity, l.TargetType, l.TargetID, l.TargetName, l.Severity, l.ClientIP, l.Timestamp)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func (r *EventRepo) QueryAuthEvents(ctx context.Context, tenantID uuid.UUID, opts domain.EventQueryOptions) ([]domain.AuthEvent, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM access_authorization_events WHERE tenant_id = $1`, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count auth events: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, event_type, context_id, severity, client_ip, timestamp, payload, resource_set_id
		 FROM access_authorization_events WHERE tenant_id = $1 ORDER BY timestamp DESC LIMIT $2 OFFSET $3`,
		tenantID, opts.PerPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query auth events: %w", err)
	}
	defer rows.Close()

	var results []domain.AuthEvent
	for rows.Next() {
		var e domain.AuthEvent
		if err := rows.Scan(&e.ID, &e.TenantID, &e.EventType, &e.ContextID, &e.Severity, &e.ClientIP, &e.Timestamp, &e.Payload, &e.ResourceSetID); err != nil {
			return nil, 0, fmt.Errorf("scan auth event: %w", err)
		}
		results = append(results, e)
	}
	return results, total, nil
}

func (r *EventRepo) QueryWorkloadEvents(ctx context.Context, tenantID uuid.UUID, opts domain.EventQueryOptions) ([]domain.WorkloadEvent, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM workload_events WHERE tenant_id = $1`, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count workload events: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, connection_id, event_type, policy_id, client_workload_id, server_workload_id, outcome, severity, timestamp, payload, resource_set_id
		 FROM workload_events WHERE tenant_id = $1 ORDER BY timestamp DESC LIMIT $2 OFFSET $3`,
		tenantID, opts.PerPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query workload events: %w", err)
	}
	defer rows.Close()

	var results []domain.WorkloadEvent
	for rows.Next() {
		var e domain.WorkloadEvent
		if err := rows.Scan(&e.ID, &e.TenantID, &e.ConnectionID, &e.EventType, &e.PolicyID, &e.ClientWorkloadID, &e.ServerWorkloadID, &e.Outcome, &e.Severity, &e.Timestamp, &e.Payload, &e.ResourceSetID); err != nil {
			return nil, 0, fmt.Errorf("scan workload event: %w", err)
		}
		results = append(results, e)
	}
	return results, total, nil
}

func (r *EventRepo) QueryAuditLogs(ctx context.Context, tenantID uuid.UUID, opts domain.EventQueryOptions) ([]domain.AuditLog, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1`, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, actor_id, actor_email, category, activity, target_type, target_id, target_name, severity, client_ip, timestamp
		 FROM audit_logs WHERE tenant_id = $1 ORDER BY timestamp DESC LIMIT $2 OFFSET $3`,
		tenantID, opts.PerPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query audit logs: %w", err)
	}
	defer rows.Close()

	var results []domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		if err := rows.Scan(&l.ID, &l.TenantID, &l.ActorID, &l.ActorEmail, &l.Category, &l.Activity, &l.TargetType, &l.TargetID, &l.TargetName, &l.Severity, &l.ClientIP, &l.Timestamp); err != nil {
			return nil, 0, fmt.Errorf("scan audit log: %w", err)
		}
		results = append(results, l)
	}
	return results, total, nil
}
