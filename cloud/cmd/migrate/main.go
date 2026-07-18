// Command migrate applies dipole cloud database schema migrations against
// DIPOLE_DATABASE_URL. Commands: up | down | steps <n> | version | force <v>.
package main

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/dipolehq/dipole/cloud/internal/repository/postgres"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	os.Exit(run(logger, os.Args[1:]))
}

func run(logger *slog.Logger, args []string) int {
	databaseURL := os.Getenv("DIPOLE_DATABASE_URL")
	if databaseURL == "" {
		logger.Error("DIPOLE_DATABASE_URL is required")
		return 1
	}

	cmd := "up"
	if len(args) > 0 {
		cmd = args[0]
	}

	migrator, err := postgres.NewMigrator(databaseURL)
	if err != nil {
		logger.Error("failed to initialize migrator", "error", err)
		return 1
	}
	defer func() {
		if cerr := migrator.Close(); cerr != nil {
			logger.Warn("error closing migrator", "error", cerr)
		}
	}()

	switch cmd {
	case "up":
		if err := migrator.Up(); err != nil {
			logger.Error("migration failed", "command", "up", "error", err)
			return 1
		}
		logPostState(logger, migrator, "migrations applied")

	case "down":
		if err := migrator.Down(); err != nil {
			logger.Error("migration failed", "command", "down", "error", err)
			return 1
		}
		logger.Info("all migrations rolled back")

	case "steps":
		if len(args) < 2 {
			logger.Error("steps requires an integer argument, e.g. `migrate steps -1`")
			return 1
		}
		n, perr := strconv.Atoi(args[1])
		if perr != nil {
			logger.Error("invalid steps argument", "value", args[1], "error", perr)
			return 1
		}
		if err := migrator.Steps(n); err != nil {
			logger.Error("migration failed", "command", "steps", "n", n, "error", err)
			return 1
		}
		logPostState(logger, migrator, "migration steps applied")

	case "version":
		logPostState(logger, migrator, "current schema version")

	case "force":
		if len(args) < 2 {
			logger.Error("force requires a version argument, e.g. `migrate force 7`")
			return 1
		}
		v, perr := strconv.Atoi(args[1])
		if perr != nil {
			logger.Error("invalid force argument", "value", args[1], "error", perr)
			return 1
		}
		if err := migrator.Force(v); err != nil {
			logger.Error("migration failed", "command", "force", "version", v, "error", err)
			return 1
		}
		logger.Info("schema version forced", "version", v)

	default:
		logger.Error("unknown command", "command", cmd,
			"valid", "up | down | steps <n> | version | force <v>")
		return 1
	}

	return 0
}

func logPostState(logger *slog.Logger, migrator *postgres.Migrator, msg string) {
	version, dirty, err := migrator.Version()
	if err != nil {
		logger.Warn("could not read schema version", "error", err)
		return
	}
	logger.Info(msg, "version", version, "dirty", dirty)
	if dirty {
		logger.Warn("schema is DIRTY — a previous migration failed mid-way; " +
			"fix the cause then run `migrate force <last-good-version>`")
	}
}
