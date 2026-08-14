# SecureMessage

Sistema de mensajeria con integridad criptografica usando el triangulo CIA (Confidentiality, Integrity, Availability).

> **Proyecto academico/demo** — No usar en produccion sin las mejoras de seguridad documentadas en `docs/superpowers/plans/2026-08-13-security-remediation.md`.

## Arquitectura

```
Frontend (Vue 3)  ──WebSocket──▶  Backend (Go)  ──HTTP──▶  CIA API (Go)
                                    │                         │
                                    ▼                         ▼
                              PostgreSQL                   Crypto
                            (Fernet + RSA-SHA256)
```

## Requisitos

- Go 1.21+
- Node.js 18+
- Docker + Docker Compose

## Instalacion

### 1. Configurar variables de entorno

```bash
cp .env.example .env
# Editar .env con tus valores (especialmente POSTGRES_PASSWORD)
```

### 2. Base de datos (Docker)

```bash
docker compose up -d
```

### 3. CIA API (Go)

```bash
cd cia-api
go run main.go
```

La CIA API debe estar corriendo en `http://127.0.0.1:8000`

### 4. Backend (Go)

```bash
cd backend
go mod tidy
go run main.go
```

### 5. Frontend (Vue)

```bash
cd frontend
npm install
npm run dev
```

## Uso

1. Abrir `http://localhost:3000` en dos pestanas
2. Ingresar nombre de usuario diferente en cada una
3. Ingresar el **mismo ID de sala** en ambas
4. Enviar mensajes
5. El receptor puede hacer clic en "Verificar" para validar la integridad

## Flujo de Criptografia

1. **Cifrado**: Frontend envia texto plano → Backend cifra via CIA API (Fernet/AES-CBC+HMAC)
2. **Firma**: Backend firma el ciphertext via CIA API (RSA-SHA256)
3. **Almacenamiento**: Solo ciphertext + firma se guardan en PostgreSQL
4. **Verificacion**: Backend verifica firma contra ciphertext via CIA API

## Seguridad

Este proyecto es un **demo academico**. Antes de usar en produccion, revisar:

- [ ] Autenticacion de usuarios (actualmente sin auth)
- [ ] Credenciales de base de datos (actualmente hardcodeadas)
- [ ] TLS en todas las conexiones
- [ ] Rotacion de llaves criptograficas
- [ ] Rate limiting

Ver plan completo de remediacion: `docs/superpowers/plans/2026-08-13-security-remediation.md`

## Estructura del Proyecto

```
.
├── backend/              # Servidor Go (WebSocket + HTTP)
│   ├── config/           # Configuracion via env vars
│   ├── database/         # Conexion PostgreSQL
│   ├── handlers/         # HTTP y WebSocket handlers
│   ├── migrations/       # SQL migrations
│   ├── models/           # Structs de dominio
│   ├── repository/       # Capa de acceso a datos
│   └── services/         # Cliente CIA API
├── cia-api/              # API de criografia (Go)
│   ├── availability/     # Simulacion de disponibilidad
│   └── crypto/           # Fernet + RSA-SHA256
├── frontend/             # Vue 3 + Vite + Tailwind
│   └── src/
│       ├── components/   # UI components
│       ├── composables/  # WebSocket logic
│       └── assets/       # CSS
├── docs/                 # Documentacion y planes
├── docker-compose.yml    # PostgreSQL
├── .env.example          # Template de variables
└── AGENTS.md             # Guia para AI agents
```

## Build

```bash
# Backend
cd backend && go build -o backend .

# CIA API
cd cia-api && go build -o cia-api .

# Frontend
cd frontend && npm run build
```

## Licencia

Proyecto academico — consultar al autor.
