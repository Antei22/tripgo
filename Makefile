-include .env
export

MIGRATIONS_DIR := migrations

.PHONY: migrate migrate-down run

migrate:
	go tool goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

migrate-down:
	go tool goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

run:
	go run ./cmd/trip-service
