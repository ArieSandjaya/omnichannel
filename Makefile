.PHONY: run build templ css css-watch migrate-up migrate-down sqlc test lint docker-build install-tools

# ── Development ──────────────────────────────────────────────────────────────

run:
	go run ./cmd/server

run-templ: templ css
	go run ./cmd/server

# ── Build ─────────────────────────────────────────────────────────────────────

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/omnichannel ./cmd/server

build-all: templ css build

# ── Templates (Templ → Go) ────────────────────────────────────────────────────

templ:
	templ generate

# ── CSS (Tailwind standalone binary) ─────────────────────────────────────────

css:
	./bin/tailwindcss -i static/css/input.css -o static/css/app.css --minify

css-watch:
	./bin/tailwindcss -i static/css/input.css -o static/css/app.css --watch

# ── Database migrations (golang-migrate) ─────────────────────────────────────

migrate-up:
	golang-migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	golang-migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-down-all:
	golang-migrate -path migrations -database "$(DATABASE_URL)" down -all

migrate-status:
	golang-migrate -path migrations -database "$(DATABASE_URL)" version

# ── sqlc code generation ──────────────────────────────────────────────────────

sqlc:
	sqlc generate

sqlc-vet:
	sqlc vet

# ── Tests ─────────────────────────────────────────────────────────────────────

test:
	go test ./... -v -race

test-short:
	go test ./... -short

test-cover:
	go test ./... -race -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# ── Lint ──────────────────────────────────────────────────────────────────────

lint:
	golangci-lint run ./...

# ── Docker ────────────────────────────────────────────────────────────────────

docker-build:
	docker build -t omnichannel:latest .

docker-run:
	docker run --env-file .env -p 8080:8080 omnichannel:latest

# ── Tool installation (run once) ──────────────────────────────────────────────

install-tools:
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@mkdir -p bin
	@echo "Downloading Tailwind CSS standalone binary..."
	@if [ "$$(uname -s)" = "Darwin" ]; then \
	  wget -q https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-macos-arm64 \
	    -O bin/tailwindcss; \
	else \
	  wget -q https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 \
	    -O bin/tailwindcss; \
	fi
	chmod +x bin/tailwindcss

# ── go.sum ────────────────────────────────────────────────────────────────────

tidy:
	go mod tidy
