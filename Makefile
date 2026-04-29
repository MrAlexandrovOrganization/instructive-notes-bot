.PHONY: setup proto test up down build lint migrate

DOCKER_COMPOSE = docker compose
WHISPER_PROTO_SRC ?= ../../backends/transcriber/proto/whisper.proto

# Tools
BUF := buf
GOLANGCI := golangci-lint

# Use system go; on broken installs fall back to /tmp/go
GOBIN := $(shell which go 2>/dev/null)
ifeq ($(GOBIN),)
GOBIN := /tmp/go/bin/go
endif

setup:
	@echo "Installing tools..."
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	pre-commit install || true

# Install all dev tools: buf + Go protoc plugins.
# buf install: https://buf.build/docs/installation
#   macOS: brew install bufbuild/buf/buf
install:
	go install github.com/bufbuild/buf/cmd/buf@v1.67.0
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.1

# Sync whisper.proto from the canonical source (backends/transcriber), then regenerate.
proto:
	@echo "Syncing whisper.proto from $(WHISPER_PROTO_SRC)..."
	@if echo "$(WHISPER_PROTO_SRC)" | grep -qE "^https?://"; then \
		curl -sSfL "$(WHISPER_PROTO_SRC)" -o proto/whisper/whisper.proto; \
	else \
		cp "$(WHISPER_PROTO_SRC)" proto/whisper/whisper.proto; \
	fi
	@sed -i.bak 's|option go_package = "[^"]*";|option go_package = "notes-bot/proto/whisper";|' proto/whisper/whisper.proto
	@rm -f proto/whisper/whisper.proto.bak
	buf generate

proto-lint:
	$(BUF) lint proto

test:
	go test ./... -race -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint:
	$(GOLANGCI) run ./...

vet:
	go vet ./...

build-core:
	go build -o bin/core ./core/cmd/server

build-telegram:
	go build -o bin/telegram ./frontends/telegram/cmd/bot

build: build-core build-telegram

up: proto
	$(DOCKER_COMPOSE) up -d --build

down:
	$(DOCKER_COMPOSE) down

logs:
	$(DOCKER_COMPOSE) logs -f

migrate:
	$(DOCKER_COMPOSE) exec core goose -dir /app/internal/db/migrations postgres "$$DATABASE_URL" up

deploy: proto
	$(DOCKER_COMPOSE) up --build -d

tidy:
	go mod tidy
