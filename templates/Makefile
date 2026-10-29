.DEFAULT_GOAL := run

MIGRATE := $(CURDIR)/.tools/bin/migrate
MIGRATE_VERSION := v4.19.0

.PHONY: run build test vet fmt-check check tools migrate-create migrate-up migrate-down integration db-up db-down

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test -race ./...

vet:
	go vet ./...

fmt-check:
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './vendor/*'))" || { echo "Run gofmt on Go source files"; exit 1; }

check: fmt-check vet test build

tools:
	GOBIN="$(CURDIR)/.tools/bin" go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION)

migrate-create:
	@test -n "$(NAME)" || { echo 'Usage: make migrate-create NAME=create_users'; exit 1; }
	@"$(MIGRATE)" create -ext sql -dir migrations -seq "$(NAME)"

migrate-up:
	@test -n "$$DATABASE_URL" || { echo 'Export DATABASE_URL before running migrations'; exit 1; }
	@"$(MIGRATE)" -path migrations -database "$$DATABASE_URL" up

migrate-down:
	@test -n "$$DATABASE_URL" || { echo 'Export DATABASE_URL before running migrations'; exit 1; }
	@"$(MIGRATE)" -path migrations -database "$$DATABASE_URL" down 1

integration:
	@test -n "$$TEST_DATABASE_URL" || { echo 'Export TEST_DATABASE_URL pointing to a test database'; exit 1; }
	go test -race -count=1 ./internal/adapter/db/...

db-up:
	docker compose up -d --wait postgres

db-down:
	docker compose down
