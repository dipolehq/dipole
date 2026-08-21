package postgres

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// TxPool extends Pool with transaction support. Satisfied by *pgxpool.Pool.
type TxPool interface {
	Pool
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Repositories struct {
	pool                   Pool
	Tenants                *TenantRepo
	ClientWorkloads        *ClientWorkloadRepo
	ServerWorkloads        *ServerWorkloadRepo
	TrustProviders         *TrustProviderRepo
	CredentialProviders    *CredentialProviderRepo
	AccessConditions       *AccessConditionRepo
	AccessPolicies         *AccessPolicyRepo
	AgentControllers       *AgentControllerRepo
	AgentProxies           *AgentProxyRepo
	Events                 *EventRepo
	ResourceSets           *ResourceSetRepo
	Users                  *UserRepo
	Roles                  *RoleRepo
	IdentityProviders      *IdentityProviderRepo
	SignOnPolicies         *SignOnPolicyRepo
	LogStreams             *LogStreamRepo
	GlobalPolicyCompliance *GlobalPolicyComplianceRepo
	TenantProfiles         *TenantProfileRepo
	TenantCAs              *TenantCARepo
	MCPClients             *MCPClientRepo
	MCPUserCredentials     *MCPUserCredentialRepo
	RegistrationTokens     *RegistrationTokenRepo
	PasswordResetTokens    *PasswordResetTokenRepo
	UserMFA                *UserMFARepo
	APITokens              *APITokenRepo
}

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	config.MaxConns = int32(envInt("DIPOLE_DB_MAX_CONNS", 25))
	config.MinConns = int32(envInt("DIPOLE_DB_MIN_CONNS", 5))
	config.MaxConnLifetime = time.Duration(envInt("DIPOLE_DB_MAX_CONN_LIFETIME_SECS", 3600)) * time.Second
	config.MaxConnIdleTime = time.Duration(envInt("DIPOLE_DB_MAX_CONN_IDLE_SECS", 300)) * time.Second
	config.HealthCheckPeriod = time.Duration(envInt("DIPOLE_DB_HEALTH_CHECK_SECS", 30)) * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

func NewRepositories(pool *pgxpool.Pool) *Repositories {
	return &Repositories{
		pool:                   pool,
		Tenants:                &TenantRepo{pool: pool},
		ClientWorkloads:        &ClientWorkloadRepo{pool: pool},
		ServerWorkloads:        &ServerWorkloadRepo{pool: pool},
		TrustProviders:         &TrustProviderRepo{pool: pool},
		CredentialProviders:    &CredentialProviderRepo{pool: pool},
		AccessConditions:       &AccessConditionRepo{pool: pool},
		AccessPolicies:         &AccessPolicyRepo{pool: pool},
		AgentControllers:       &AgentControllerRepo{pool: pool},
		AgentProxies:           &AgentProxyRepo{pool: pool},
		Events:                 &EventRepo{pool: pool},
		ResourceSets:           &ResourceSetRepo{pool: pool},
		Users:                  &UserRepo{pool: pool},
		Roles:                  &RoleRepo{pool: pool},
		IdentityProviders:      &IdentityProviderRepo{pool: pool},
		SignOnPolicies:         &SignOnPolicyRepo{pool: pool},
		LogStreams:             &LogStreamRepo{pool: pool},
		GlobalPolicyCompliance: &GlobalPolicyComplianceRepo{pool: pool},
		TenantProfiles:         &TenantProfileRepo{pool: pool},
		TenantCAs:              &TenantCARepo{pool: pool},
		MCPClients:             NewMCPClientRepo(pool),
		MCPUserCredentials:     NewMCPUserCredentialRepo(pool, nil),
		RegistrationTokens:     &RegistrationTokenRepo{pool: pool},
		PasswordResetTokens:    &PasswordResetTokenRepo{pool: pool},
		UserMFA:                &UserMFARepo{pool: pool},
		APITokens:              &APITokenRepo{pool: pool},
	}
}

func (r *Repositories) SetMCPEncryptionKey(key []byte) {
	r.MCPUserCredentials.encKey = key
}

func (r *Repositories) SetLogStreamEncryptionKey(key []byte) {
	r.LogStreams.encKey = key
}

func (r *Repositories) SetTrustProviderEncryptionKey(key []byte) {
	r.TrustProviders.encKey = key
}

func (r *Repositories) Close() {
	if p, ok := r.pool.(*pgxpool.Pool); ok {
		p.Close()
	}
}

func (r *Repositories) Ping(ctx context.Context) error {
	if p, ok := r.pool.(*pgxpool.Pool); ok {
		return p.Ping(ctx)
	}
	return nil
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
