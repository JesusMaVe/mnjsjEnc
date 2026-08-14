# SecureMessage — Arquitectura del Sistema

## 1. Diagrama de Arquitectura

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLIENTE (Browser)                        │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    Vue 3 App                            │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │   │
│  │  │ConnectionForm│  │  ChatRoom    │  │ MessageItem  │  │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │   │
│  │                         │                               │   │
│  │                  ┌──────┴──────┐                        │   │
│  │                  │ useWebSocket│ (composable)           │   │
│  │                  └──────┬──────┘                        │   │
│  └─────────────────────────┼───────────────────────────────┘   │
│                            │ WebSocket                         │
└────────────────────────────┼───────────────────────────────────┘
                             │
┌────────────────────────────┼───────────────────────────────────┐
│                     SERVIDOR (Go)                              │
│  ┌─────────────────────────┴───────────────────────────────┐   │
│  │                   WebSocket Handler                     │   │
│  │  - Manejo de conexiones                                 │   │
│  │  - Broadcast a sala                                      │   │
│  └─────────────────────────┬───────────────────────────────┘   │
│                            │                                   │
│  ┌─────────────────────────┴───────────────────────────────┐   │
│  │                  Message Service                        │   │
│  │  - Encriptación AES-256-GCM                             │   │
│  │  - Generación SHA-256                                    │   │
│  │  - Firma RSA                                            │   │
│  │  - Validación de integridad                             │   │
│  └─────────────────────────┬───────────────────────────────┘   │
│                            │                                   │
│  ┌─────────────────────────┴───────────────────────────────┐   │
│  │                   Repository                           │   │
│  │  - Queries PostgreSQL                                   │   │
│  │  - Mapeo de datos                                       │   │
│  └─────────────────────────┬───────────────────────────────┘   │
│                            │                                   │
└────────────────────────────┼───────────────────────────────────┘
                             │
┌────────────────────────────┼───────────────────────────────────┐
│                     PostgreSQL                                │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  rooms │ room_users │ messages │ message_integrity      │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Estructura del Proyecto

### Backend (Go)

```
backend/
├── main.go                    ← Entry point, config
├── go.mod
├── internal/
│   ├── handlers/
│   │   └── websocket.go       ← Manejo de conexiones WebSocket
│   ├── models/
│   │   ├── message.go         ← Estructura de mensaje
│   │   ├── room.go            ← Estructura de sala
│   │   └── integrity.go       ← Estructura de integridad
│   ├── services/
│   │   ├── crypto.go          ← RSA, AES, SHA-256
│   │   ├── message.go         ← Lógica de mensajes
│   │   └── validation.go      ← Validación de integridad
│   ├── repository/
│   │   ├── message.go         ← Queries de mensajes
│   │   └── room.go            ← Queries de salas
│   └── database/
│       └── postgres.go        ← Conexión a DB
├── migrations/
│   └── 001_initial.sql        ← Schema SQL
└── config/
    └── config.go              ← Variables de entorno
```

### Frontend (Vue)

```
frontend/
├── src/
│   ├── components/
│   │   ├── ConnectionForm.vue    ← Ingreso nombre + room ID
│   │   ├── ChatRoom.vue          ← Sala de chat principal
│   │   ├── MessageItem.vue       ← Mensaje individual + status
│   │   └── PublicKeyDisplay.vue  ← Muestra clave pública del usuario
│   ├── composables/
│   │   ├── useWebSocket.js       ← Conexión WebSocket
│   │   └── useCrypto.js          ← Generación de claves RSA
│   ├── views/
│   │   └── Home.vue              ← Página principal
│   ├── App.vue
│   └── main.js
├── package.json
├── tailwind.config.js
└── index.html
```

---

## 3. Protocolo WebSocket

### Eventos

| Evento | Dirección | Payload |
|--------|-----------|---------|
| `join` | Cliente → Server | `{ username, roomId }` |
| `joined` | Server → Cliente | `{ username, publicKey }` |
| `message` | Cliente → Server | `{ content }` |
| `new_message` | Server → Cliente | `{ id, sender, content, status }` |
| `validate` | Cliente → Server | `{ messageId }` |
| `validated` | Server → Cliente | `{ messageId, status }` |
| `error` | Server → Cliente | `{ message }` |

---

## 4. Paleta de Colores

| Elemento | Color | HEX |
|----------|-------|-----|
| Background | Blanco roto | `#F8F9FA` |
| Surface | Gris claro | `#E9ECEF` |
| Primary | Azul profundo | `#2563EB` |
| Primary Hover | Azul claro | `#3B82F6` |
| Text | Negro suave | `#1F2937` |
| Text Secondary | Gris | `#6B7280` |
| Error | Rojo | `#DC2626` |
| Success | Verde | `#16A34A` |

---

## 5. Tipografía

| Elemento | Font | Peso | Tamaño |
|----------|------|------|--------|
| H1 | Plus Jakarta Sans | 700 | 32px |
| H2 | Plus Jakarta Sans | 700 | 24px |
| H3 | Plus Jakarta Sans | 600 | 20px |
| Body | Inter | 400 | 16px |
| Small | Inter | 400 | 12px |

---

## 6. Dependencias

### Backend (Go)

```
github.com/gorilla/websocket v1.5.0
github.com/lib/pq v1.10.9
github.com/google/uuid v1.4.0
```

### Frontend (npm)

```
vue@3.4.0
tailwindcss@3.4.0
@vueuse/core@10.7.0
```
