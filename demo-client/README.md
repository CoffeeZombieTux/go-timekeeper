# GoTimekeeper Demo Client

Simple local UI for demonstrating the backend API.

This is intentionally demo-grade. It is designed for local usage, not production.

## What it covers

- Health: `/ping`
- Auth: register, login, refresh, logout, change password
- User: get me, delete me
- Projects: create, get, list, update, delete
- Tasks: create, get, list by project, update, delete, start, stop, close
- Time records (sessions): create, update, delete
- Reports: general, project, task

## Run locally

1. Start backend (`http://localhost:8080` by default).
2. From repo root, serve this folder:

```bash
cd demo-client
go run ./server.go
```

3. Open `http://localhost:5173`.
4. Keep `Base URL` as `http://localhost:8080` (or change it if your backend runs elsewhere).

## Usage flow (recommended)

1. `Register` or `Login`
2. `Create Project`
3. `Create Task`
4. `Start Task` / `Stop Task` or use `Create Time Record`
5. Run report endpoints

The page stores values in `localStorage`, so IDs and tokens survive reloads.
