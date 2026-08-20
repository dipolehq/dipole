/*
log_stream.go implements storage for log stream (SIEM/S3/GCS export)
configurations. Config (destination credentials: Splunk HEC token,
CrowdStrike client secret, AWS/GCS keys) is encrypted at rest with
AES-256-GCM via internal/crypto, following the same pattern as
CredentialProvider.ConfigEncrypted and MCPUserCredentialRepo. The legacy
plaintext `config` column is retained only until a follow-up migration
drops it once BackfillConfigEncryption has been run in every environment;
Create/Update always scrub it to '{}' going forward so no new plaintext
secret is ever written to it.
*/
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/crypto"
	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type LogStreamRepo struct {
	pool   Pool
	encKey []byte
}

func (r *LogStreamRepo) Create(ctx context.Context, tenantID uuid.UUID, ls *domain.LogStream) error {
	ls.ID = uuid.New()
	ls.TenantID = tenantID

	configEncrypted, err := r.encryptConfig(ls.Config)
	if err != nil {
		return err
	}

	err = r.pool.QueryRow(ctx,
		`INSERT INTO log_streams (id, tenant_id, name, description, is_active, event_type, destination_type, config, config_encrypted)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, '{}', $8)
		 RETURNING created_at, updated_at`,
		ls.ID, tenantID, ls.Name, ls.Description, ls.IsActive, ls.EventType, ls.DestinationType, configEncrypted).
		Scan(&ls.CreatedAt, &ls.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create log stream: %w", err)
	}
	return nil
}

func (r *LogStreamRepo) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.LogStream, error) {
	ls := &domain.LogStream{}
	var configEncrypted []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, is_active, event_type, destination_type, config_encrypted, created_at, updated_at, last_exported_at, last_export_error
		 FROM log_streams WHERE tenant_id = $1 AND id = $2`, tenantID, id).
		Scan(&ls.ID, &ls.TenantID, &ls.Name, &ls.Description, &ls.IsActive, &ls.EventType, &ls.DestinationType, &configEncrypted, &ls.CreatedAt, &ls.UpdatedAt, &ls.LastExportedAt, &ls.LastExportError)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get log stream: %w", err)
	}
	config, err := r.decryptConfig(configEncrypted)
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
		`SELECT id, tenant_id, name, description, is_active, event_type, destination_type, config_encrypted, created_at, updated_at, last_exported_at, last_export_error
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
		var configEncrypted []byte
		if err := rows.Scan(&ls.ID, &ls.TenantID, &ls.Name, &ls.Description, &ls.IsActive, &ls.EventType, &ls.DestinationType, &configEncrypted, &ls.CreatedAt, &ls.UpdatedAt, &ls.LastExportedAt, &ls.LastExportError); err != nil {
			return nil, 0, fmt.Errorf("scan log stream: %w", err)
		}
		config, err := r.decryptConfig(configEncrypted)
		if err != nil {
			return nil, 0, fmt.Errorf("list log streams: %w", err)
		}
		ls.Config = config
		results = append(results, ls)
	}
	return results, total, nil
}

func (r *LogStreamRepo) Update(ctx context.Context, tenantID uuid.UUID, ls *domain.LogStream) error {
	configEncrypted, err := r.encryptConfig(ls.Config)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx,
		`UPDATE log_streams SET name=$1, description=$2, is_active=$3, event_type=$4, destination_type=$5, config='{}', config_encrypted=$6, updated_at=NOW()
		 WHERE tenant_id = $7 AND id = $8`,
		ls.Name, ls.Description, ls.IsActive, ls.EventType, ls.DestinationType, configEncrypted, tenantID, ls.ID)
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

func (r *LogStreamRepo) ListAllActive(ctx context.Context) ([]domain.LogStream, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, description, is_active, event_type, destination_type, config_encrypted, created_at, updated_at, last_exported_at, last_export_error
		 FROM log_streams WHERE is_active = true`)
	if err != nil {
		return nil, fmt.Errorf("list all active log streams: %w", err)
	}
	defer rows.Close()

	var results []domain.LogStream
	for rows.Next() {
		var ls domain.LogStream
		var configEncrypted []byte
		if err := rows.Scan(&ls.ID, &ls.TenantID, &ls.Name, &ls.Description, &ls.IsActive, &ls.EventType, &ls.DestinationType, &configEncrypted, &ls.CreatedAt, &ls.UpdatedAt, &ls.LastExportedAt, &ls.LastExportError); err != nil {
			return nil, fmt.Errorf("scan log stream: %w", err)
		}
		config, err := r.decryptConfig(configEncrypted)
		if err != nil {
			return nil, fmt.Errorf("list all active log streams: %w", err)
		}
		ls.Config = config
		results = append(results, ls)
	}
	return results, nil
}

// UpdateLastExportedAt records a successful export and clears any
// previously recorded export error, so a stream that failed once and then
// recovers stops showing a stale failure to the customer.
func (r *LogStreamRepo) UpdateLastExportedAt(ctx context.Context, id uuid.UUID, t time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE log_streams SET last_exported_at = $1, last_export_error = NULL WHERE id = $2`, t, id)
	if err != nil {
		return fmt.Errorf("update last_exported_at: %w", err)
	}
	return nil
}

// SetLastExportError records that the most recent export attempt failed,
// without touching last_exported_at (which reflects the last *successful*
// export).
func (r *LogStreamRepo) SetLastExportError(ctx context.Context, id uuid.UUID, errMsg string) error {
	_, err := r.pool.Exec(ctx, `UPDATE log_streams SET last_export_error = $1 WHERE id = $2`, errMsg, id)
	if err != nil {
		return fmt.Errorf("set last_export_error: %w", err)
	}
	return nil
}

// BackfillConfigEncryption is the one-time, idempotent migrate step of the
// expand-migrate-contract rollout of encrypted log stream config: it
// encrypts the plaintext `config` of every row that hasn't been migrated
// yet (config_encrypted IS NULL) and clears the plaintext column. Safe to
// run on every startup — rows already migrated are not touched. Must be
// called after SetEncryptionKey and before the HTTP/gRPC servers start
// accepting traffic, so no request ever observes a row mid-migration.
//
// Concurrency note: during a rolling deploy, an old-code instance may still
// be serving traffic (and writing plaintext `config` via the old
// Create/Update path, which is unaware of `config_encrypted`) while a
// new-code instance runs this backfill. Without a guard, the backfill could
// read a row's plaintext config, then have a concurrent old-code write
// change it, then blindly overwrite `config_encrypted` with the stale value
// it read — silently losing the customer's newer update. Each row's UPDATE
// therefore includes `AND config = <the exact value just read>` as an
// optimistic-concurrency check: if the plaintext changed since the SELECT,
// the UPDATE affects 0 rows and that row is safely left for the next
// backfill run (which happens on every startup) instead of clobbering it.
func (r *LogStreamRepo) BackfillConfigEncryption(ctx context.Context) (int, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, config FROM log_streams WHERE config_encrypted IS NULL`)
	if err != nil {
		return 0, fmt.Errorf("query unmigrated log streams: %w", err)
	}

	type pending struct {
		id     uuid.UUID
		config []byte
	}
	var toMigrate []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.config); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan unmigrated log stream: %w", err)
		}
		toMigrate = append(toMigrate, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate unmigrated log streams: %w", err)
	}

	migrated := 0
	for _, p := range toMigrate {
		originalConfig := p.config
		config := originalConfig
		if len(config) == 0 {
			config = []byte(`{}`)
		}
		encrypted, err := crypto.Encrypt(config, r.encKey)
		if err != nil {
			return migrated, fmt.Errorf("encrypt config for log stream %s: %w", p.id, err)
		}

		// originalConfig (not the {}-substituted `config`) is compared
		// against the current row so the guard matches exactly what the
		// initial SELECT observed, including a genuinely-NULL/empty column.
		tag, err := r.pool.Exec(ctx,
			`UPDATE log_streams SET config_encrypted = $1, config = '{}'
			 WHERE id = $2 AND config_encrypted IS NULL AND config IS NOT DISTINCT FROM $3`,
			encrypted, p.id, originalConfig)
		if err != nil {
			return migrated, fmt.Errorf("backfill config_encrypted for log stream %s: %w", p.id, err)
		}
		if tag.RowsAffected() == 1 {
			migrated++
		}
		// RowsAffected() == 0 means the plaintext config (or
		// config_encrypted) changed concurrently since the SELECT above —
		// leave the row for the next backfill run rather than overwriting
		// a newer write with the stale value read here.
	}

	return migrated, nil
}

func (r *LogStreamRepo) encryptConfig(config json.RawMessage) ([]byte, error) {
	if len(config) == 0 {
		config = json.RawMessage(`{}`)
	}
	encrypted, err := crypto.Encrypt(config, r.encKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt log stream config: %w", err)
	}
	return encrypted, nil
}

func (r *LogStreamRepo) decryptConfig(configEncrypted []byte) (json.RawMessage, error) {
	if len(configEncrypted) == 0 {
		return json.RawMessage(`{}`), nil
	}
	plaintext, err := crypto.Decrypt(configEncrypted, r.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt log stream config: %w", err)
	}
	return json.RawMessage(plaintext), nil
}
