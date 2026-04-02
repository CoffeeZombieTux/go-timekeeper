# GoTimekeeper Local Frontend Client

Local browser client for using the GoTimekeeper API from one dashboard.

## Features

- Connection panel with API `ping`, base URL, timezone, and data sync
- Auth workflow: register, login, token refresh, logout, get profile, change password
- Project workspace: create, rename, delete, select
- Task board for selected project: create/update/delete/start/stop/close and status filtering
- Manual session form (create/update/delete time records)
- Reports with date range:
  - General report (optionally scoped to selected project)
  - Project report (selected project)
  - Task report (selected task)
- API trace panel showing last request + response payloads
- Local persistence (tokens, selected IDs, form values) via `localStorage`

## Run locally

1. Start the API server (default `http://localhost:8080`):

```bash
go run ./cmd/api
```

2. In a new terminal, run the static frontend server:

```bash
cd demo-client
go run ./server.go
```

3. Open `http://localhost:5173`.

## Recommended first flow

1. Register or login.
2. Create/select a project.
3. Create/select a task.
4. Start/stop task timer or add a manual session.
5. Run reports for a date range.
