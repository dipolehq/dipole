package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type GlobalPolicyComplianceRepo struct {
	pool Pool
}

func (r *GlobalPolicyComplianceRepo) Get(ctx context.Context, tenantID uuid.UUID) (*domain.GlobalPolicyCompliance, error) {
	p := &domain.GlobalPolicyCompliance{
		TenantID:                   tenantID,
		APTrustProvider:            "required",
		APAccessCondition:          "recommended",
		ACTrustProvider:            "recommended",
		ACTLSHostname:              "recommended",
		AccessTokenLifetimeSeconds: 604800,
	}
	err := r.pool.QueryRow(ctx,
		`SELECT ap_trust_provider, ap_access_condition, ac_trust_provider, ac_tls_hostname, access_token_lifetime_seconds, updated_at
		 FROM global_policy_compliance WHERE tenant_id = $1`, tenantID).
		Scan(&p.APTrustProvider, &p.APAccessCondition, &p.ACTrustProvider, &p.ACTLSHostname, &p.AccessTokenLifetimeSeconds, &p.UpdatedAt)
	if err == pgx.ErrNoRows {
		return p, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get global policy compliance: %w", err)
	}
	return p, nil
}

func (r *GlobalPolicyComplianceRepo) Upsert(ctx context.Context, tenantID uuid.UUID, p *domain.GlobalPolicyCompliance) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO global_policy_compliance (tenant_id, ap_trust_provider, ap_access_condition, ac_trust_provider, ac_tls_hostname, access_token_lifetime_seconds, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())
		 ON CONFLICT (tenant_id) DO UPDATE SET ap_trust_provider=$2, ap_access_condition=$3, ac_trust_provider=$4, ac_tls_hostname=$5, access_token_lifetime_seconds=$6, updated_at=NOW()`,
		tenantID, p.APTrustProvider, p.APAccessCondition, p.ACTrustProvider, p.ACTLSHostname, p.AccessTokenLifetimeSeconds)
	if err != nil {
		return fmt.Errorf("upsert global policy compliance: %w", err)
	}
	return nil
}
