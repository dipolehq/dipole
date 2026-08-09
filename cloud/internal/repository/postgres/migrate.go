/*
Package postgres — migrate.go embeds the SQL migration files and drives
schema migrations (up/down/steps/version/force) via golang-migrate.
*/
package postgres

import (
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Migrator struct {
	m *migrate.Migrate
}

func NewMigrator(databaseURL string) (*Migrator, error) {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("load embedded migrations: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, toPgxURL(databaseURL))
	if err != nil {
		return nil, fmt.Errorf("init migrator: %w", err)
	}
	return &Migrator{m: m}, nil
}

func (mg *Migrator) Up() error {
	if err := mg.m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

func (mg *Migrator) Down() error {
	if err := mg.m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate down: %w", err)
	}
	return nil
}

func (mg *Migrator) Steps(n int) error {
	if err := mg.m.Steps(n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate steps %d: %w", n, err)
	}
	return nil
}

func (mg *Migrator) Version() (version uint, dirty bool, err error) {
	version, dirty, err = mg.m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("migrate version: %w", err)
	}
	return version, dirty, nil
}

func (mg *Migrator) Force(version int) error {
	if err := mg.m.Force(version); err != nil {
		return fmt.Errorf("migrate force %d: %w", version, err)
	}
	return nil
}

func (mg *Migrator) Close() error {
	srcErr, dbErr := mg.m.Close()
	return errors.Join(srcErr, dbErr)
}

func toPgxURL(databaseURL string) string {
	switch {
	case strings.HasPrefix(databaseURL, "postgresql://"):
		return "pgx5://" + strings.TrimPrefix(databaseURL, "postgresql://")
	case strings.HasPrefix(databaseURL, "postgres://"):
		return "pgx5://" + strings.TrimPrefix(databaseURL, "postgres://")
	default:
		return databaseURL
	}
}
