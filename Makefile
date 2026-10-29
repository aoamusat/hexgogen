.DEFAULT_GOAL := build
VERSION ?= dev

.PHONY: build test vet fmt-check check integration

build:
	go build -ldflags "-X github.com/aoamusat/hexgogen/internal/cli.Version=$(VERSION)" -o bin/hexgogen ./cmd/hexgogen

test:
	go test -race ./...

vet:
	go vet ./...

fmt-check:
	@test -z "$$(gofmt -l $$(find . \( -name '*.go' -o -name '*.go.tmpl' \) -not -path './vendor/*'))" || { echo 'Run gofmt on Go source and template files'; exit 1; }

check: fmt-check vet test build

integration:
	@test -n "$$TEST_DATABASE_URL" || { echo 'Export TEST_DATABASE_URL pointing to a test database'; exit 1; }
	go test -race -count=1 ./internal/generator -run '^TestGeneratedPostgresIntegration$$'
