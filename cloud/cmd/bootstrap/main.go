// Command bootstrap provisions (or re-tokens) a tenant and prints a fresh API
// token once. Intended for local development and initial environment setup.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository/postgres"
	"github.com/dipolehq/dipole/cloud/internal/service"
	"github.com/google/uuid"
)

func main() {
	name := flag.String("name", "Dev Tenant", "tenant display name")
	subdomain := flag.String("subdomain", "dev", "tenant subdomain (unique)")
	adminEmail := flag.String("email", "admin@dipole.dev", "admin user email")
	adminPassword := flag.String("password", "admin123", "admin user password")
	flag.Parse()

	databaseURL := os.Getenv("DIPOLE_DATABASE_URL")
	if databaseURL == "" {
		fmt.Fprintln(os.Stderr, "DIPOLE_DATABASE_URL is required")
		os.Exit(1)
	}

	ctx := context.Background()

	pool, err := postgres.NewPool(ctx, databaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	repos := postgres.NewRepositories(pool)

	token, err := service.GenerateAPIToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate token: %v\n", err)
		os.Exit(1)
	}
	hash := service.HashAPIToken(token)

	existing, err := repos.Tenants.GetBySubdomain(ctx, *subdomain)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lookup tenant: %v\n", err)
		os.Exit(1)
	}

	var tenantID string
	if existing == nil {
		tenant := &domain.Tenant{Name: *name, Subdomain: *subdomain, APITokenHash: hash}
		if err := repos.Tenants.Create(ctx, tenant); err != nil {
			fmt.Fprintf(os.Stderr, "create tenant: %v\n", err)
			os.Exit(1)
		}
		tenantID = tenant.ID.String()
	} else {
		existing.APITokenHash = hash
		if err := repos.Tenants.Update(ctx, existing); err != nil {
			fmt.Fprintf(os.Stderr, "update tenant: %v\n", err)
			os.Exit(1)
		}
		tenantID = existing.ID.String()
	}

	// Create or update admin user with password
	tID, _ := uuid.Parse(tenantID)
	passwordHash, err := service.HashPassword(*adminPassword)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hash password: %v\n", err)
		os.Exit(1)
	}

	existingUser, _ := repos.Users.GetByEmail(ctx, tID, *adminEmail)
	if existingUser == nil {
		user := &domain.User{
			Email:        *adminEmail,
			Name:         "Admin",
			FirstName:    "Admin",
			IsActive:     true,
			PasswordHash: passwordHash,
		}
		if err := repos.Users.Create(ctx, tID, user); err != nil {
			fmt.Fprintf(os.Stderr, "create admin user: %v\n", err)
			os.Exit(1)
		}
	} else {
		existingUser.PasswordHash = passwordHash
		if err := repos.Users.Update(ctx, tID, existingUser); err != nil {
			fmt.Fprintf(os.Stderr, "update admin password: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("Tenant provisioned.")
	fmt.Printf("  subdomain: %s\n", *subdomain)
	fmt.Printf("  tenantId:  %s\n", tenantID)
	fmt.Println()
	fmt.Println("Admin user:")
	fmt.Printf("  email:    %s\n", *adminEmail)
	fmt.Printf("  password: %s\n", *adminPassword)
	fmt.Println()
	fmt.Println("API token (shown once — store it now):")
	fmt.Printf("  %s\n", token)
}
