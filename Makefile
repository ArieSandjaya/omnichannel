.PHONY: run build templ css css-watch migrate-up migrate-down sqlc test \
        install-templ install-tailwind install-tools clean

# ── Tools ──────────────────────────────────────────────────────────
install-templ:
	go install github.com/a-h/templ/cmd/templ@latest

install-tailwind:
	@mkdir -p bin
	@echo "Downloading Tailwind CSS standalone CLI..."
	curl -sLo bin/tailwindcss \
		https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64
	chmod +x bin/tailwindcss

install-sqlc:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

install-migrate:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

install-tools: install-templ install-tailwind install-sqlc install-migrate

# ── Generate ───────────────────────────────────────────────────────
templ:
	templ generate

sqlc:
	sqlc generate

# ── CSS ────────────────────────────────────────────────────────────
css:
	./bin/tailwindcss -i static/css/input.css -o static/css/app.css --minify

css-watch:
	./bin/tailwindcss -i static/css/input.css -o static/css/app.css --watch

# ── Dev / Build ────────────────────────────────────────────────────
generate: templ sqlc

run: templ
	go run ./cmd/server

dev:
	@templ generate --watch &
	@./bin/tailwindcss -i static/css/input.css -o static/css/app.css --watch &
	go run ./cmd/server

build: generate css
	CGO_ENABLED=0 go build -o bin/omnichannel ./cmd/server

# ── Database ───────────────────────────────────────────────────────
migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-reset:
	migrate -path migrations -database "$(DATABASE_URL)" drop -f
	migrate -path migrations -database "$(DATABASE_URL)" up

# ── Test & Quality ─────────────────────────────────────────────────
test:
	go test ./... -v -race

lint:
	golangci-lint run ./...

# ── Docker ─────────────────────────────────────────────────────────
docker-build:
	docker build -t omnichannel:latest .

docker-run:
	docker run -p 8080:8080 --env-file .env omnichannel:latest

# ── Cleanup ────────────────────────────────────────────────────────
clean:
	rm -f bin/omnichannel
	rm -f static/css/app.css
	find templates -name "*_templ.go" -delete
