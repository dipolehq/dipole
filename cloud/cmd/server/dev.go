/*
dev.go contains the local-development bootstrap that creates a tenant, admin
user, and API token so a fresh database is immediately usable. Gated behind
DIPOLE_DEV_BOOTSTRAP=true and must never run in a deployed environment.
*/
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dipolehq/dipole/cloud/internal/config"
	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository/postgres"
	"github.com/dipolehq/dipole/cloud/internal/service/auth"
)

func devBootstrap(ctx context.Context, repos *postgres.Repositories, cfg *config.Config, logger *slog.Logger) error {
	if !cfg.DevBootstrap {
		return nil
	}
	const devSubdomain = "dev"
	existing, err := repos.Tenants.GetBySubdomain(ctx, devSubdomain)
	if err != nil {
		return fmt.Errorf("lookup dev tenant: %w", err)
	}
	if existing != nil {
		logger.Info("dev tenant already present", "id", existing.ID, "subdomain", existing.Subdomain)
		return nil
	}
	token, err := auth.GenerateAPIToken()
	if err != nil {
		return fmt.Errorf("generate api token: %w", err)
	}
	tenant := &domain.Tenant{
		Name:         "Dev Tenant",
		Subdomain:    devSubdomain,
		APITokenHash: auth.HashAPIToken(token),
	}
	if err := repos.Tenants.Create(ctx, tenant); err != nil {
		return fmt.Errorf("create dev tenant: %w", err)
	}
	passwordHash, err := auth.HashPassword("admin123")
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	adminUser := &domain.User{
		Email:        "admin@dipole.dev",
		Name:         "Admin",
		FirstName:    "Admin",
		IsActive:     true,
		PasswordHash: passwordHash,
	}
	if err := repos.Users.Create(ctx, tenant.ID, adminUser); err != nil {
		return fmt.Errorf("create dev admin user: %w", err)
	}
	logger.Warn("DEV BOOTSTRAP: created tenant with a well-known password — local development only",
		"tenant_id", tenant.ID,
		"subdomain", devSubdomain,
		"email", "admin@dipole.dev",
		"password", "admin123",
		"api_token", token,
	)
	return nil
}
