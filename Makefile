.PHONY: run build migrate-up migrate-down sqlc tidy test clean install-tools

# Load .env if present
-include .env
export

DSN ?= postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)

run:
	go run ./cmd/server/main.go

build:
	go build -o bin/server ./cmd/server/main.go

migrate-up:
	migrate -path ./migrations -database "$(DSN)" up

migrate-down:
	migrate -path ./migrations -database "$(DSN)" down 1

sqlc:
	sqlc generate

tidy:
	go mod tidy

test:
	go test ./... -v -race

clean:
	rm -rf bin/ db/sqlc/

install-tools:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
