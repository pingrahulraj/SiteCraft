# SiteCraft

SiteCraft is a website builder starter with a Go API, SQLite persistence, and a Vue 3 frontend.

## Layout

- `cmd/api` - Go API entrypoint
- `internal/app` - API, config, and SQLite persistence
- `frontend` - Vue 3 app

## Run

Backend:

```powershell
go run ./cmd/api
```

Frontend:

```powershell
cd frontend
npm install
npm run dev
```
