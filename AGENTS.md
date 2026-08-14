# SecureMessage

## Architecture

Three services, must run in order:

1. **CIA API** (`cia-api/`) — Go crypto service on `:8000` (Fernet + RSA-SHA256)
2. **Backend** (`backend/`) — Go WebSocket server on `:8080`
3. **Frontend** (`frontend/`) — Vue 3 + Vite on `:3000`

PostgreSQL runs via Docker Compose on `:5432`.

## Startup Order

```bash
# 1. PostgreSQL
docker compose up -d

# 2. CIA API
cd cia-api && go run main.go

# 3. Backend (separate terminal)
cd backend && go run main.go

# 4. Frontend (separate terminal)
cd frontend && npm run dev
```

## Key Gotchas

- **CIA API must start before backend** — backend health-checks it on startup
- **Room lookup uses name, not UUID** — frontend sends room name (e.g. "999"), backend resolves to UUID internally
- **Messages sent over WebSocket are ciphertext** — frontend decrypts via `POST /decrypt` on `:8080` (backend proxy), NOT directly to CIA API
- **CORS** — CIA API and backend both allow `*` origins; if adding auth, update CORS handlers
- **Database auto-migrates** — `docker-compose.yml` mounts `backend/migrations/001_initial.sql` into init dir
- **Clean DB** — `docker exec -i securemessage-db psql -U postgres -d securemessage -c "TRUNCATE message_integrity, messages, room_users, rooms CASCADE;"`

## Crypto Keys

- Keys persist in `cia-api/keys/crypto.json` (Fernet key + RSA-2048 private key)
- **NEVER delete `cia-api/keys/`** if you want to preserve existing messages — deleting generates new keys, making old ciphertexts undecipherable
- `cia-api/keys/` is in `.gitignore` — never commit keys
- First run: generates and saves keys. Subsequent runs: loads from disk.

## Message Flow

1. Frontend sends plaintext over WebSocket
2. Backend encrypts (Fernet) + signs (RSA-SHA256) via CIA API, stores both in DB
3. Backend broadcasts **ciphertext only** to room members
4. Frontend decrypts via `POST /decrypt` (backend proxy → CIA API)
5. Verification: backend loads original plaintext + signature from DB, verifies via CIA API
6. If verification fails (e.g. keys changed), backend re-signs with current keys automatically

## Availability

CIA API includes `/availability/status` and `/availability/request` endpoints simulating a 3-node cluster with health checks (30% fail rate). Used for demonstrating the Availability pillar of the CIA Triad.

## Build Commands

```bash
# Backend
cd backend && go build -o backend .

# CIA API
cd cia-api && go build -o cia-api .

# Frontend
cd frontend && npm run build
```

## Module Names

- `securemessage` — backend (Go)
- `securemessage-cia` — CIA API (Go)
- `securemessage-frontend` — frontend (Vue)
