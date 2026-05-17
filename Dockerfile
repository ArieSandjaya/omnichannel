FROM golang:1.22-alpine AS builder
WORKDIR /app

# Install build tools
RUN apk add --no-cache curl wget

# Install Tailwind standalone CLI
RUN wget -q https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 \
    -O /usr/local/bin/tailwindcss && chmod +x /usr/local/bin/tailwindcss

# Install Templ code generator
RUN go install github.com/a-h/templ/cmd/templ@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Generate Templ Go code
RUN templ generate

# Generate Tailwind CSS (minified)
RUN tailwindcss -i static/css/input.css -o static/css/app.css --minify

# Build single static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /omnichannel ./cmd/server


FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /omnichannel /omnichannel
COPY --from=builder /app/static /static

EXPOSE 8080

ENTRYPOINT ["/omnichannel"]
