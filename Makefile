-include .env.example
-include .env
export

MIGRATIONS_DIR := migrations

.PHONY: generate migrate migrate-down migrate-status run build test

generate:
	go tool oapi-codegen --config oapi-codegen.yaml contracts/openapi/trip-service.openapi.yaml

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
	go test -race ./...