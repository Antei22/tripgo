-include .env.example
-include .env
export

MIGRATIONS_DIR := migrations
GOLANGCI_LINT ?= go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0

.PHONY: generate migrate migrate-down migrate-status run build test test-integration lint

generate:
	go tool oapi-codegen --config oapi-codegen.yaml contracts/openapi/trip-service.openapi.yaml
	go generate ./...

migrate:
	go tool goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

migrate-down:
	go tool goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

migrate-status:
	go tool goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" status

run:
	go run ./cmd/trip-service

build:
	go build ./...

test:
	@coverage=$$(mktemp); trap 'rm -f "$$coverage"' EXIT; \
	  go test -race -coverprofile="$$coverage" -coverpkg=./internal/domain/...,./internal/usecase/... ./... && \
	  go tool cover -func="$$coverage" | tail -n 1

test-integration:
	go test -race -tags=integration ./internal/repository/...

lint:
	$(GOLANGCI_LINT) run ./...
