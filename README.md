# clinic-backend

New backend for clinic management system powered by Gin, with better architecture, better database table design, better document.

## Run locally

```bash
# under clinic-backend

# 1. Seed sample data (room, service dates, announcement, staff)
go run fake/seed.go

# 2. Start the fake CAS server (now gives admin role)
go run fake/fake_cas.go   # runs on :9999

# 3. Start the backend
export CLINIC_API_KEY=local-dev-key
export CAS_SERVER_URL=http://127.0.0.1:9999
export APP_BASE_URL=http://127.0.0.1:5173
export CAS_DEFAULT_REDIRECT=/
export SESSION_COOKIE_SAMESITE=lax

go run main.go                   # runs on :8080

# 4. Start the frontend (separate terminal)
cd /path/to/clinic_admin_frontend
pnpm dev                         # runs on :5173
```
