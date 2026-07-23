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
