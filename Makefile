# dipole developer Makefile
#
# Common local-development and database tasks. Migrations are driven by the
# `migrate` command in cloud/cmd/migrate, which embeds the SQL files and runs
# them via golang-migrate. Override DIPOLE_DATABASE_URL to target another DB.

# Local dev database (docker-compose postgres service).
DIPOLE_DATABASE_URL ?= postgres://dipole:dipole@localhost:5432/dipole?sslmode=disable
export DIPOLE_DATABASE_URL

# Default GeoIP path for local dev (download with `make geoip` first).
DIPOLE_GEOIP_DB_PATH ?= $(shell pwd)/data/GeoLite2-City.mmdb
export DIPOLE_GEOIP_DB_PATH

# Local-dev encryption key (32 bytes hex). Production must supply this via a secret.
DIPOLE_ENCRYPTION_KEY ?= 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
export DIPOLE_ENCRYPTION_KEY

CLOUD_DIR := cloud

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

# ---------------------------------------------------------------------------
# Database (docker-compose)
# ---------------------------------------------------------------------------

.PHONY: db-up
db-up: ## Start the local Postgres container
	docker compose up -d postgres

.PHONY: db-down
db-down: ## Stop the local Postgres container (keeps data volume)
	docker compose stop postgres

.PHONY: db-reset
db-reset: ## Destroy and recreate the local Postgres (DROPS ALL LOCAL DATA)
	docker compose rm -sfv postgres
	docker volume rm dipole_pgdata || true
	docker compose up -d postgres

.PHONY: db-psql
db-psql: ## Open a psql shell in the local Postgres container
	docker exec -it dipole-postgres-1 psql -U dipole -d dipole

# ---------------------------------------------------------------------------
# Migrations
# ---------------------------------------------------------------------------

.PHONY: migrate-up
migrate-up: ## Apply all pending migrations
	cd $(CLOUD_DIR) && go run ./cmd/migrate up

.PHONY: migrate-down
migrate-down: ## Roll back ALL migrations (dev/test only)
	cd $(CLOUD_DIR) && go run ./cmd/migrate down

.PHONY: migrate-version
migrate-version: ## Print current schema version and dirty flag
	cd $(CLOUD_DIR) && go run ./cmd/migrate version

.PHONY: migrate-force
migrate-force: ## Force schema to VERSION and clear dirty flag (recovery only): make migrate-force VERSION=9
	cd $(CLOUD_DIR) && go run ./cmd/migrate force $(VERSION)

# ---------------------------------------------------------------------------
# Build & test
# ---------------------------------------------------------------------------

.PHONY: run
run: ## Run the cloud server (REST + gRPC)
	cd $(CLOUD_DIR) && DIPOLE_ENCRYPTION_KEY=$(DIPOLE_ENCRYPTION_KEY) go run ./cmd/server

.PHONY: bootstrap
bootstrap: ## Provision a dev tenant and print an API token: make bootstrap [SUBDOMAIN=dev]
	cd $(CLOUD_DIR) && go run ./cmd/bootstrap $(if $(SUBDOMAIN),-subdomain $(SUBDOMAIN),)

.PHONY: build
build: ## Build all cloud binaries
	cd $(CLOUD_DIR) && go build ./...

.PHONY: test
test: ## Run cloud unit tests
	cd $(CLOUD_DIR) && go test ./...

# ---------------------------------------------------------------------------
# GeoIP Database
# ---------------------------------------------------------------------------

.PHONY: geoip
geoip: ## Download MaxMind GeoLite2-City database (requires MAXMIND_ACCOUNT_ID + MAXMIND_LICENSE_KEY env vars)
	./scripts/download-geoip-db.sh
