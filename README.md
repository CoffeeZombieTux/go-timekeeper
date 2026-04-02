# GoTimekeeper

GoTimekeeper is a REST API for personal time tracking:
- user authentication with JWT access tokens and hashed refresh tokens
- project and task management
- active timer workflow (`start`/`stop`) with timezone-aware day splitting
- manual time session CRUD
- aggregated reports (general, project, task)

## Stack

- Go `1.25`
- Gin
- PostgreSQL
- Logrus
- Cron (`robfig/cron`)
- Docker + Docker Compose

## Project Layout

```text
cmd/api                  # app entrypoint
internal/auth            # JWT + password hashing
internal/config          # env-based config
internal/database        # DB bootstrap + health checks
internal/handler         # HTTP handlers + API response mapping
internal/middleware      # auth, request-id, request/response logging
internal/model           # domain and API DTOs
internal/repository      # DB access
internal/router          # route registration
internal/service         # business logic
internal/uow             # transaction unit-of-work helper
migrations               # SQL schema
http                     # IDE HTTP client requests
postman                  # Postman collection + local environment
```

## Configuration

Configuration is loaded from environment variables (with defaults):

- `DB_HOST` (default `localhost`)
- `DB_PORT` (default `5432`)
- `DB_USER` (default `postgres`)
- `DB_PASSWORD` (default empty)
- `DB_NAME` (default `timekeeper`)
- `DB_SSL_MODE` (default `disable`)
- `SERVER_PORT` (default `8080`)
- `SERVER_HOST` (default `0.0.0.0`)
- `LOG_LEVEL` (`debug|info|warn|error`, default `info`)
- `LOG_FORMAT` (`json|text`, default `json`)
- `JWT_SECRET`
- `ACCESS_TOKEN_TTL_MINUTES` (default `15`)
- `REFRESH_TOKEN_TTL_HOURS` (default `168`)
- `CRON_CLEANUP_REVOKED_TOKENS_SPEC` (default `@every 12h`)
- `CRON_CLEANUP_REVOKED_TOKENS_INTERVAL_DAYS` (default `7`)

## Run Locally

### 1) Start PostgreSQL

```bash
docker compose up -d db
```

### 2) Apply schema

```bash
psql "host=localhost port=5432 user=root password=RooT!@123 dbname=timekeeper sslmode=disable" -f migrations/0001_init.up.sql
```

### 3) Start API

```bash
go run ./cmd/api
```

Health check:

```bash
curl http://localhost:8080/ping
```

## Run With Docker (App + DB)

```bash
docker compose up --build
```

The app container uses `air` for live reload (`.air.toml`) and Delve for debugging.

## API Groups

- `GET /ping`
- `POST /api/auth/register`
- `POST /api/auth/login`
- `POST /api/auth/logout`
- `POST /api/auth/refresh`
- `POST /api/auth/change-password`
- `GET /api/user/me`
- `DELETE /api/user/me`
- `POST /api/project`
- `GET /api/project/:id`
- `GET /api/project/list`
- `PATCH /api/project`
- `DELETE /api/project/:id`
- `POST /api/task`
- `GET /api/task/:id`
- `GET /api/task/list/project/:id`
- `PATCH /api/task`
- `DELETE /api/task/:id`
- `PATCH /api/task/:id/start` (requires `X-Timezone`)
- `PATCH /api/task/:id/stop`
- `PATCH /api/task/:id/close`
- `POST /api/task/session`
- `PATCH /api/task/session`
- `DELETE /api/task/session/:id`
- `POST /api/report/general`
- `POST /api/report/project`
- `POST /api/report/task`

## Request Collections

- IDE HTTP requests: `http/*.http`
- Postman:
  - `postman/GoTimekeeper.postman_collection.json`
  - `postman/GoTimekeeper.local.postman_environment.json`

## Demo Client

A simple local demo UI is available in `demo-client/`.

Run it with:

```bash
cd demo-client
go run ./server.go
```

Then open `http://localhost:5173` while backend is running.

## Response Envelope

All responses use a common envelope:

```json
{
  "success": true,
  "message": "Task created",
  "data": {}
}
```

Errors:

```json
{
  "success": false,
  "message": "Validation failed",
  "error": {
    "code": "VALIDATION_ERROR",
    "requestId": "..."
  }
}
```

## Notes

- `workDate` is stored in DB as `work_date` (`DATE`), while session boundaries are `TIMESTAMPTZ`.
- Active session conflicts are validated against manual time records for the same task/day.
- Repository + service layers are separated, and transactions are coordinated via unit-of-work.
