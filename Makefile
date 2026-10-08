SHELL := /bin/bash
GO ?= go
SERVER_DIR := backend

.PHONY: build server seed test fmt vet backup restore clean help

help:
	@echo "Targets:"
	@echo "  build          - build the server and seed binaries"
	@echo "  server         - run the Go server on the host (dev)"
	@echo "  seed EMAIL=… PW=… NAME=… - run the seed CLI to create the first manager"
	@echo "  test           - go test ./..."
	@echo "  fmt            - go fmt ./..."
	@echo "  vet            - go vet ./..."
	@echo "  backup         - run the host-side backup script (dev)"
	@echo "  restore DATE=YYYY-MM-DD - restore a backup to ./data/ludo.db"
	@echo "  clean          - remove ./data/*.db"

build:
	cd $(SERVER_DIR) && $(GO) build -o ../bin/server ./cmd/server
	cd $(SERVER_DIR) && $(GO) build -o ../bin/seed   ./cmd/seed

server:
	cd $(SERVER_DIR) && DB_PATH=../data/ludo.db COOKIE_SECURE=false \
	  SESSION_KEY=$$(head -c 32 /dev/urandom | base64) \
	  SMTP_HOST=localhost SMTP_PORT=1025 SMTP_FROM=dev@example.com \
	  PUBLIC_URL=http://localhost:8080 \
	  $(GO) run ./cmd/server

seed:
	cd $(SERVER_DIR) && $(GO) run ./cmd/seed --email "$(EMAIL)" --password "$(PW)" --name "$(NAME)"

test:
	cd $(SERVER_DIR) && $(GO) test ./...

fmt:
	cd $(SERVER_DIR) && $(GO) fmt ./...

vet:
	cd $(SERVER_DIR) && $(GO) vet ./...

backup:
	bash deploy/scripts/backup.sh

restore:
	cp backups/ludo-$(DATE).db data/ludo.db
	@echo "restored data/ludo.db from backups/ludo-$(DATE).db"

clean:
	rm -f data/*.db data/*.db-shm data/*.db-wal