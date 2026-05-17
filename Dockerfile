# ── Stage 1: Builder ──────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder
WORKDIR /app

RUN apk add --no-cache wget ca-certificates

# Install templ code generator
RUN go install github.com/a-h/templ/cmd/templ@latest

# Download Tailwind CSS standalone binary
RUN wget -q \
    https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 \
    -O /usr/local/bin/tailwindcss \
  && chmod +x /usr/local/bin/tailwindcss

# Download Go module dependencies (cached layer)
COPY go.mod ./
RUN go mod download

# Copy source
COPY . .

# Generate Templ → Go code
RUN templ generate

# Compile Tailwind CSS (skip if static/css/input.css does not exist yet)
RUN [ -f static/css/input.css ] \
  && tailwindcss -i static/css/input.css -o static/css/app.css --minify \
  || mkdir -p static/css

# Build single Go binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /omnichannel ./cmd/server


# ── Stage 2: Runtime ──────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

# Copy binary and static assets
COPY --from=builder /omnichannel   /omnichannel
COPY --from=builder /app/static    /static
COPY --from=builder /app/templates /templates

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s \
  CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/omnichannel"]
