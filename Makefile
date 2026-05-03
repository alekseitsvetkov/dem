.PHONY: build-all build-poller build-downloader build-parser \
        migrate-up migrate-down migrate-create \
        docker-up docker-down docker-logs \
        vet test test-race

# ── Build ──────────────────────────────────────────────

build-all: build-poller build-downloader build-parser

build-poller:
	go build -o bin/poller ./cmd/poller

build-downloader:
	go build -o bin/downloader ./cmd/downloader

build-parser:
	go build -o bin/parser ./cmd/parser

# ── Migrations ─────────────────────────────────────────

migrate-up:
	@echo "Usage: make migrate-up DATABASE_URL=postgres://..."
	@echo "Example: make migrate-up DATABASE_URL=\"postgres://dem:dem@localhost:5432/dem?sslmode=disable\""
	cd sql/migrations && migrate -database "$(DATABASE_URL)" -path . up

migrate-down:
	cd sql/migrations && migrate -database "$(DATABASE_URL)" -path . down

migrate-create:
	@read -p "Migration name: " NAME; \
	migrate create -ext sql -dir sql/migrations -seq $$NAME

# ── Docker ─────────────────────────────────────────────

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

# ── Quality ────────────────────────────────────────────

vet:
	go vet ./cmd/... ./internal/... $$(test -d ./pkg && echo './pkg/...')

test:
	go test ./cmd/... ./internal/... ./pkg/...

test-race:
	go test -race ./cmd/... ./internal/... ./pkg/...
