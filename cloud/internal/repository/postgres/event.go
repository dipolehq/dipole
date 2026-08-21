// event.go implements the EventRepository for PostgreSQL, the read+write path for auth/workload events, audit logs, and dashboard stats.
package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/scope"
)

type EventRepo struct {
	pool Pool
}

func (r *EventRepo) QueryAuthEvents(ctx context.Context, tenantID uuid.UUID, opts domain.EventQueryOptions) ([]domain.AuthEvent, int, error) {
	where, args := buildAuthEventFilters(ctx, tenantID, opts)

	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM access_authorization_events WHERE `+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count auth events: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	args = append(args, opts.PerPage, offset)
	limitParam := fmt.Sprintf("$%d", len(args)-1)
	offsetParam := fmt.Sprintf("$%d", len(args))

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, event_type, context_id, severity, client_ip, timestamp, payload, resource_set_id, source
		 FROM access_authorization_events WHERE `+where+` ORDER BY timestamp DESC, id DESC LIMIT `+limitParam+` OFFSET `+offsetParam,
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query auth events: %w", err)
	}
	defer rows.Close()

	var results []domain.AuthEvent
	for rows.Next() {
		var e domain.AuthEvent
		if err := rows.Scan(&e.ID, &e.TenantID, &e.EventType, &e.ContextID, &e.Severity, &e.ClientIP, &e.Timestamp, &e.Payload, &e.ResourceSetID, &e.Source); err != nil {
			return nil, 0, fmt.Errorf("scan auth event: %w", err)
		}
		results = append(results, e)
	}
	return results, total, nil
}

func (r *EventRepo) QueryWorkloadEvents(ctx context.Context, tenantID uuid.UUID, opts domain.EventQueryOptions) ([]domain.WorkloadEvent, int, error) {
	where, args := buildWorkloadEventFilters(ctx, tenantID, opts)

	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM workload_events WHERE `+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count workload events: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	args = append(args, opts.PerPage, offset)
	limitParam := fmt.Sprintf("$%d", len(args)-1)
	offsetParam := fmt.Sprintf("$%d", len(args))

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, connection_id, event_type, policy_id, client_workload_id, server_workload_id, outcome, severity, timestamp, payload, resource_set_id
		 FROM workload_events WHERE `+where+` ORDER BY timestamp DESC, id DESC LIMIT `+limitParam+` OFFSET `+offsetParam,
		args...)
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
	where, args := buildAuditLogFilters(ctx, tenantID, opts)

	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE `+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	args = append(args, opts.PerPage, offset)
	limitParam := fmt.Sprintf("$%d", len(args)-1)
	offsetParam := fmt.Sprintf("$%d", len(args))

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, actor_id, actor_email, category, activity, target_type, target_id, target_name, severity, client_ip, timestamp
		 FROM audit_logs WHERE `+where+` ORDER BY timestamp DESC, id DESC LIMIT `+limitParam+` OFFSET `+offsetParam,
		args...)
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

func buildAuthEventFilters(ctx context.Context, tenantID uuid.UUID, opts domain.EventQueryOptions) (string, []interface{}) {
	args := []interface{}{tenantID}
	clauses := []string{"tenant_id = $1"}
	n := 1

	if opts.Search != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("(event_type ILIKE '%%' || $%d || '%%' OR context_id ILIKE '%%' || $%d || '%%' OR client_ip ILIKE '%%' || $%d || '%%')", n, n, n))
		args = append(args, opts.Search)
	}
	if len(opts.Severities) > 0 {
		n++
		clauses = append(clauses, fmt.Sprintf("severity = ANY($%d)", n))
		args = append(args, opts.Severities)
	} else if opts.Severity != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("severity = $%d", n))
		args = append(args, opts.Severity)
	}
	if opts.EventType != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("event_type = $%d", n))
		args = append(args, opts.EventType)
	}
	if opts.Source != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("source = $%d", n))
		args = append(args, opts.Source)
	}
	if opts.StartDate != nil {
		n++
		clauses = append(clauses, fmt.Sprintf("timestamp >= $%d", n))
		args = append(args, *opts.StartDate)
	}
	if opts.EndDate != nil {
		n++
		clauses = append(clauses, fmt.Sprintf("timestamp <= $%d", n))
		args = append(args, *opts.EndDate)
	}

	appendResourceSetFilter(ctx, &clauses, &args, &n)

	return strings.Join(clauses, " AND "), args
}

func buildWorkloadEventFilters(ctx context.Context, tenantID uuid.UUID, opts domain.EventQueryOptions) (string, []interface{}) {
	args := []interface{}{tenantID}
	clauses := []string{"tenant_id = $1"}
	n := 1

	if opts.Search != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("(event_type ILIKE '%%' || $%d || '%%' OR connection_id ILIKE '%%' || $%d || '%%' OR outcome ILIKE '%%' || $%d || '%%')", n, n, n))
		args = append(args, opts.Search)
	}
	if len(opts.Severities) > 0 {
		n++
		clauses = append(clauses, fmt.Sprintf("severity = ANY($%d)", n))
		args = append(args, opts.Severities)
	} else if opts.Severity != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("severity = $%d", n))
		args = append(args, opts.Severity)
	}
	if opts.EventType != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("event_type = $%d", n))
		args = append(args, opts.EventType)
	}
	if opts.StartDate != nil {
		n++
		clauses = append(clauses, fmt.Sprintf("timestamp >= $%d", n))
		args = append(args, *opts.StartDate)
	}
	if opts.EndDate != nil {
		n++
		clauses = append(clauses, fmt.Sprintf("timestamp <= $%d", n))
		args = append(args, *opts.EndDate)
	}
	if len(opts.ClientWorkloadIDs) > 0 {
		n++
		clauses = append(clauses, fmt.Sprintf("client_workload_id::text = ANY($%d)", n))
		args = append(args, opts.ClientWorkloadIDs)
	}
	if len(opts.ServerWorkloadIDs) > 0 {
		n++
		clauses = append(clauses, fmt.Sprintf("server_workload_id::text = ANY($%d)", n))
		args = append(args, opts.ServerWorkloadIDs)
	}
	if opts.UserID != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("payload->'mcp'->>'userId' = $%d", n))
		args = append(args, opts.UserID)
	}

	appendResourceSetFilter(ctx, &clauses, &args, &n)

	return strings.Join(clauses, " AND "), args
}

func buildAuditLogFilters(ctx context.Context, tenantID uuid.UUID, opts domain.EventQueryOptions) (string, []interface{}) {
	args := []interface{}{tenantID}
	clauses := []string{"tenant_id = $1"}
	n := 1

	if opts.Search != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("(actor_email ILIKE '%%' || $%d || '%%' OR category ILIKE '%%' || $%d || '%%' OR activity ILIKE '%%' || $%d || '%%' OR target_name ILIKE '%%' || $%d || '%%')", n, n, n, n))
		args = append(args, opts.Search)
	}
	if len(opts.Severities) > 0 {
		n++
		clauses = append(clauses, fmt.Sprintf("severity = ANY($%d)", n))
		args = append(args, opts.Severities)
	} else if opts.Severity != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("severity = $%d", n))
		args = append(args, opts.Severity)
	}
	if opts.EventType != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("category = $%d", n))
		args = append(args, opts.EventType)
	}
	if opts.StartDate != nil {
		n++
		clauses = append(clauses, fmt.Sprintf("timestamp >= $%d", n))
		args = append(args, *opts.StartDate)
	}
	if opts.EndDate != nil {
		n++
		clauses = append(clauses, fmt.Sprintf("timestamp <= $%d", n))
		args = append(args, *opts.EndDate)
	}

	return strings.Join(clauses, " AND "), args
}

func (r *EventRepo) DashboardEventStats(ctx context.Context, tenantID uuid.UUID, timeRange string) (*domain.DashboardEventData, error) {
	data := &domain.DashboardEventData{}

	interval, truncInterval := resolveTimeGranularity(timeRange)

	rsClause := ""
	weRsClause := ""
	aeRsClause := ""
	baseArgs := []interface{}{tenantID}
	if s := scope.FromContext(ctx); s != nil {
		rsClause = " AND resource_set_id = $2"
		weRsClause = " AND we.resource_set_id = $2"
		aeRsClause = " AND ae.resource_set_id = $2"
		baseArgs = append(baseArgs, s.ResourceSetID)
	}

	err := r.pool.QueryRow(ctx,
		`SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE severity = 'Info') AS authorized,
			COUNT(*) FILTER (WHERE severity = 'Error') AS denied
		 FROM access_authorization_events
		 WHERE tenant_id = $1 AND event_type = 'access.authorization' AND timestamp >= NOW() - `+interval+rsClause,
		baseArgs...,
	).Scan(&data.TotalLast6h, &data.AuthorizedLast6h, &data.DeniedLast6h)
	if err != nil {
		return nil, fmt.Errorf("dashboard event counts: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT
			date_trunc('`+truncInterval+`', timestamp) AS bucket,
			COUNT(*) FILTER (WHERE severity = 'Info') AS info,
			COUNT(*) FILTER (WHERE severity = 'Warning') AS warning,
			COUNT(*) FILTER (WHERE severity = 'Error') AS error
		 FROM access_authorization_events
		 WHERE tenant_id = $1 AND timestamp >= NOW() - `+interval+rsClause+`
		 GROUP BY bucket
		 ORDER BY bucket`,
		baseArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("dashboard time series: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var pt domain.DashboardTimePoint
		if err := rows.Scan(&pt.Timestamp, &pt.Info, &pt.Warning, &pt.Error); err != nil {
			return nil, fmt.Errorf("scan time series: %w", err)
		}
		data.TimeSeries = append(data.TimeSeries, pt)
	}
	rows.Close()

	rows, err = r.pool.Query(ctx,
		`SELECT
			date_trunc('`+truncInterval+`', timestamp) AS bucket,
			COUNT(*) FILTER (WHERE severity = 'Info') AS authorized,
			COUNT(*) FILTER (WHERE severity = 'Error') AS denied
		 FROM access_authorization_events
		 WHERE tenant_id = $1 AND event_type = 'access.authorization' AND timestamp >= NOW() - `+interval+rsClause+`
		 GROUP BY bucket
		 ORDER BY bucket`,
		baseArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("dashboard decision series: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var pt domain.DashboardDecisionPoint
		if err := rows.Scan(&pt.Timestamp, &pt.Authorized, &pt.Denied); err != nil {
			return nil, fmt.Errorf("scan decision series: %w", err)
		}
		data.AccessDecisionSeries = append(data.AccessDecisionSeries, pt)
	}
	rows.Close()

	rows, err = r.pool.Query(ctx,
		`SELECT we.client_workload_id::text, COALESCE(cw.name, 'Unknown'), COUNT(*)
		 FROM workload_events we
		 LEFT JOIN client_workloads cw ON cw.id = we.client_workload_id AND cw.tenant_id = we.tenant_id
		 WHERE we.tenant_id = $1 AND we.client_workload_id IS NOT NULL AND we.timestamp >= NOW() - INTERVAL '24 hours'`+weRsClause+`
		 GROUP BY we.client_workload_id, cw.name
		 ORDER BY COUNT(*) DESC
		 LIMIT 5`,
		baseArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("dashboard top clients: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var wc domain.DashboardWorkloadCount
		if err := rows.Scan(&wc.ID, &wc.Name, &wc.Count); err != nil {
			return nil, fmt.Errorf("scan top client: %w", err)
		}
		data.TopClients = append(data.TopClients, wc)
	}
	rows.Close()

	rows, err = r.pool.Query(ctx,
		`SELECT we.server_workload_id::text, COALESCE(sw.name, 'Unknown'), COUNT(*)
		 FROM workload_events we
		 LEFT JOIN server_workloads sw ON sw.id = we.server_workload_id AND sw.tenant_id = we.tenant_id
		 WHERE we.tenant_id = $1 AND we.server_workload_id IS NOT NULL AND we.timestamp >= NOW() - INTERVAL '24 hours'`+weRsClause+`
		 GROUP BY we.server_workload_id, sw.name
		 ORDER BY COUNT(*) DESC
		 LIMIT 5`,
		baseArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("dashboard top servers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var wc domain.DashboardWorkloadCount
		if err := rows.Scan(&wc.ID, &wc.Name, &wc.Count); err != nil {
			return nil, fmt.Errorf("scan top server: %w", err)
		}
		data.TopServers = append(data.TopServers, wc)
	}
	rows.Close()

	rows, err = r.pool.Query(ctx,
		`SELECT payload->'credentialProvider'->>'type' AS cp_type, COUNT(*)
		 FROM access_authorization_events
		 WHERE tenant_id = $1 AND event_type = 'access.credential' AND severity = 'Info'
		   AND timestamp >= NOW() - INTERVAL '24 hours'
		   AND payload->'credentialProvider'->>'type' IS NOT NULL`+rsClause+`
		 GROUP BY cp_type
		 ORDER BY COUNT(*) DESC`,
		baseArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("dashboard credential usage: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tc domain.DashboardTypeCount
		if err := rows.Scan(&tc.Type, &tc.Count); err != nil {
			return nil, fmt.Errorf("scan credential usage: %w", err)
		}
		data.CredentialUsage = append(data.CredentialUsage, tc)
	}
	rows.Close()

	rows, err = r.pool.Query(ctx,
		`SELECT split_part(event_type, '.', 1) AS protocol, COUNT(*)
		 FROM workload_events
		 WHERE tenant_id = $1 AND timestamp >= NOW() - INTERVAL '24 hours'`+rsClause+`
		 GROUP BY protocol
		 ORDER BY COUNT(*) DESC`,
		baseArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("dashboard protocol usage: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tc domain.DashboardTypeCount
		if err := rows.Scan(&tc.Type, &tc.Count); err != nil {
			return nil, fmt.Errorf("scan protocol usage: %w", err)
		}
		data.ProtocolUsage = append(data.ProtocolUsage, tc)
	}
	rows.Close()

	rows, err = r.pool.Query(ctx,
		`SELECT ae.payload->'accessPolicy'->>'id' AS policy_id, COALESCE(ae.payload->'accessPolicy'->>'name', 'Unknown'), COUNT(*)
		 FROM access_authorization_events ae
		 WHERE ae.tenant_id = $1 AND ae.event_type = 'access.authorization' AND ae.severity = 'Error'
		   AND ae.timestamp >= NOW() - INTERVAL '24 hours'
		   AND ae.payload->'accessPolicy'->>'id' IS NOT NULL`+aeRsClause+`
		 GROUP BY policy_id, ae.payload->'accessPolicy'->>'name'
		 ORDER BY COUNT(*) DESC
		 LIMIT 5`,
		baseArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("dashboard policy failures: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var pf domain.DashboardPolicyFailure
		if err := rows.Scan(&pf.PolicyID, &pf.PolicyName, &pf.Count); err != nil {
			return nil, fmt.Errorf("scan policy failure: %w", err)
		}
		data.PolicyFailures = append(data.PolicyFailures, pf)
	}

	attRows, err := r.pool.Query(ctx,
		`SELECT result, COUNT(*) FROM (
			SELECT jsonb_array_elements(payload->'trustProviders')->>'result' AS result
			FROM access_authorization_events
			WHERE tenant_id = $1 AND event_type = 'access.authorization'
			  AND timestamp >= NOW() - INTERVAL '24 hours'
			  AND payload->'trustProviders' IS NOT NULL
			  AND jsonb_array_length(payload->'trustProviders') > 0`+rsClause+`
		) sub
		WHERE result IS NOT NULL
		GROUP BY result
		ORDER BY COUNT(*) DESC`,
		baseArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("dashboard attestation results: %w", err)
	}
	defer attRows.Close()
	for attRows.Next() {
		var tc domain.DashboardTypeCount
		if err := attRows.Scan(&tc.Type, &tc.Count); err != nil {
			return nil, fmt.Errorf("scan attestation result: %w", err)
		}
		data.AttestationResults = append(data.AttestationResults, tc)
	}

	return data, nil
}

func resolveTimeGranularity(timeRange string) (interval string, truncGranularity string) {
	switch timeRange {
	case "1h":
		return "INTERVAL '1 hour'", "minute"
	case "6h":
		return "INTERVAL '6 hours'", "minute"
	case "24h":
		return "INTERVAL '24 hours'", "hour"
	case "7d":
		return "INTERVAL '7 days'", "hour"
	case "30d":
		return "INTERVAL '30 days'", "day"
	default:
		return "INTERVAL '6 hours'", "hour"
	}
}
