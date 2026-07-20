package postgres

import (
	"context"
	"fmt"

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
	pool                Pool
	Tenants             *TenantRepo
	ClientWorkloads     *ClientWorkloadRepo
	ServerWorkloads     *ServerWorkloadRepo
	TrustProviders      *TrustProviderRepo
	CredentialProviders *CredentialProviderRepo
	AccessConditions    *AccessConditionRepo
	AccessPolicies      *AccessPolicyRepo
	AgentControllers    *AgentControllerRepo
	Events              *EventRepo
	ResourceSets        *ResourceSetRepo
	Users               *UserRepo
	Roles               *RoleRepo
	IdentityProviders   *IdentityProviderRepo
	SignOnPolicies      *SignOnPolicyRepo
}

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

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
		pool:                pool,
		Tenants:             &TenantRepo{pool: pool},
		ClientWorkloads:     &ClientWorkloadRepo{pool: pool},
		ServerWorkloads:     &ServerWorkloadRepo{pool: pool},
		TrustProviders:      &TrustProviderRepo{pool: pool},
		CredentialProviders: &CredentialProviderRepo{pool: pool},
		AccessConditions:    &AccessConditionRepo{pool: pool},
		AccessPolicies:      &AccessPolicyRepo{pool: pool},
		AgentControllers:    &AgentControllerRepo{pool: pool},
		Events:              &EventRepo{pool: pool},
		ResourceSets:        &ResourceSetRepo{pool: pool},
		Users:               &UserRepo{pool: pool},
		Roles:               &RoleRepo{pool: pool},
		IdentityProviders:   &IdentityProviderRepo{pool: pool},
		SignOnPolicies:      &SignOnPolicyRepo{pool: pool},
	}
}

func (r *Repositories) Close() {
	if p, ok := r.pool.(*pgxpool.Pool); ok {
		p.Close()
	}
}
