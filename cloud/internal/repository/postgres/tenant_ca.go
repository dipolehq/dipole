/*
Package postgres — tenant_ca.go implements the TenantCARepository using
SELECT ... FOR UPDATE to safely handle concurrent CA generation attempts.
*/
package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type TenantCARepo struct {
	pool TxPool
}

func (r *TenantCARepo) GetOrCreateCA(ctx context.Context, tenantID uuid.UUID, certPEM string, keyEncrypted []byte) (string, []byte, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var storedCert *string
	var storedKey []byte
	err = tx.QueryRow(ctx,
		`SELECT ca_cert_pem, ca_key_encrypted FROM tenants WHERE id = $1 FOR UPDATE`,
		tenantID).Scan(&storedCert, &storedKey)
	if err != nil {
		return "", nil, fmt.Errorf("lock tenant row: %w", err)
	}

	if storedCert != nil && *storedCert != "" {
		if err := tx.Commit(ctx); err != nil {
			return "", nil, fmt.Errorf("commit read: %w", err)
		}
		return *storedCert, storedKey, nil
	}

	_, err = tx.Exec(ctx,
		`UPDATE tenants SET ca_cert_pem = $1, ca_key_encrypted = $2, updated_at = NOW() WHERE id = $3`,
		certPEM, keyEncrypted, tenantID)
	if err != nil {
		return "", nil, fmt.Errorf("store tenant CA: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", nil, fmt.Errorf("commit CA store: %w", err)
	}

	return certPEM, keyEncrypted, nil
}

func (r *TenantCARepo) GetCA(ctx context.Context, tenantID uuid.UUID) (string, []byte, error) {
	var storedCert *string
	var storedKey []byte
	err := r.pool.QueryRow(ctx,
		`SELECT ca_cert_pem, ca_key_encrypted FROM tenants WHERE id = $1`,
		tenantID).Scan(&storedCert, &storedKey)
	if err != nil {
		return "", nil, fmt.Errorf("read tenant CA: %w", err)
	}
	if storedCert == nil {
		return "", nil, nil
	}
	return *storedCert, storedKey, nil
}

func (r *TenantCARepo) HasCA(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	var cert *string
	err := r.pool.QueryRow(ctx,
		`SELECT ca_cert_pem FROM tenants WHERE id = $1`, tenantID).Scan(&cert)
	if err != nil {
		return false, fmt.Errorf("check tenant CA: %w", err)
	}
	return cert != nil && *cert != "", nil
}

func (r *TenantCARepo) GetLeafLifetime(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var lifetime int
	err := r.pool.QueryRow(ctx,
		`SELECT leaf_lifetime_seconds FROM tenants WHERE id = $1`,
		tenantID).Scan(&lifetime)
	if err != nil {
		return 86400, fmt.Errorf("get leaf lifetime: %w", err)
	}
	return lifetime, nil
}

func (r *TenantCARepo) SetLeafLifetime(ctx context.Context, tenantID uuid.UUID, lifetimeSeconds int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`UPDATE tenants SET leaf_lifetime_seconds = $1, updated_at = NOW() WHERE id = $2`,
		lifetimeSeconds, tenantID)
	if err != nil {
		return fmt.Errorf("set leaf lifetime: %w", err)
	}

	return tx.Commit(ctx)
}
