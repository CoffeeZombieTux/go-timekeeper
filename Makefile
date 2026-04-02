# === CONFIG ===

ifneq (,$(wildcard .env))
	include .env
	export
endif

MIGRATIONS_DIR = migrations
DB_HOST_LOCAL = localhost
DB_SSL_MODE ?= disable

DB_URL = postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST_LOCAL):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)

# === MIGRATIONS ===
migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" down

migrate-drop:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" drop -f

migrate-force:
	@echo "⚠️ Forcing to version: $(VERSION)"
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" force $(VERSION)

migrate-version:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" version

migrate-new:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $$name

# === BUILD / RUN ===
build:
	docker compose build

up:
	docker compose up

down:
	docker compose down

restart:
	docker compose down && docker compose up --build

psql:
	docker compose exec db psql -U user -d timekeeper

update-sum:
	go mod tidy

test-unit:
	go test ./...

test-unit-no-cache:
	go test -count=1 ./...

test-coverage:
	go test ./... -cover

test-coverage-profile:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

test-coverage-html:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
