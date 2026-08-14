# SecureMessage — Diseño del Sistema

## 1. Visión General

Aplicación de mensajería 1:1 para demostrar conceptos de integridad criptográfica en el contexto de una materia de ciberseguridad.

**Stack:**
- Frontend: Vue 3 + Tailwind CSS
- Backend: Go + Gorilla WebSocket
- Base de datos: PostgreSQL

**Objetivo:** Enviar mensajes encriptados y validar su integridad usando criptografía asimétrica (RSA) y hashing (SHA-256).

---

## 2. Requisitos Funcionales

| ID | Requisito | Prioridad |
|----|-----------|-----------|
| RF-01 | Dos usuarios pueden conectarse a una sala compartida | Alta |
| RF-02 | Los mensajes se encriptan con AES antes de guardarse | Alta |
| RF-03 | Cada mensaje tiene un hash SHA-256 del contenido original | Alta |
| RF-04 | Los mensajes se firman con clave privada RSA del emisor | Alta |
| RF-05 | El receptor puede validar la integridad del mensaje | Alta |
| RF-06 | El status cambia de "no_validado" a "validado" tras validación | Alta |
| RF-07 | Los mensajes se persisten en PostgreSQL | Media |
| RF-08 | Se muestran badges de status (validado/no validado) | Media |

---

## 3. Requisitos No Funcionales

| ID | Requisito | Detalle |
|----|-----------|---------|
| RNF-01 | Rendimiento | Mensajes en < 100ms |
| RNF-02 | Seguridad | Claves RSA de 2048 bits, AES-256-GCM |
| RNF-03 | Usabilidad | Interfaz intuitiva sin documentación |
| RNF-04 | Compatibilidad | Chrome, Firefox, Safari (últimas 2 versiones) |

---

## 4. Diseño de Base de Datos

```sql
CREATE TABLE rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE room_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID REFERENCES rooms(id) ON DELETE CASCADE,
    username VARCHAR(50) NOT NULL,
    public_key TEXT NOT NULL,
    joined_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(room_id, username)
);

CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID REFERENCES rooms(id) ON DELETE CASCADE,
    sender_username VARCHAR(50) NOT NULL,
    content_encrypted TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE message_integrity (
    message_id UUID PRIMARY KEY REFERENCES messages(id) ON DELETE CASCADE,
    hash_original VARCHAR(64) NOT NULL,
    signature TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'no_validado'
        CHECK (status IN ('no_validado', 'validado')),
    validated_at TIMESTAMP
);
```

---

## 5. Flujo de Criptografía

### Envío de Mensaje

1. Usuario escribe mensaje en texto plano
2. Backend calcula SHA-256 del mensaje → `hash`
3. Backend encripta mensaje con AES-256-GCM → `content_encrypted`
4. Backend firma `hash` con clave privada RSA del emisor → `signature`
5. Guarda en DB: `content_encrypted`, `hash`, `signature`, `status='no_validado'`

### Validación de Mensaje

1. Receptor solicita validar mensaje
2. Backend descifra `content_encrypted` con AES → mensaje original
3. Backend calcula SHA-256 del mensaje descifrado → `hash_calculado`
4. Backend verifica `hash_calculado == hash_original`
5. Backend verifica `signature` con clave pública del emisor
6. Si ambas verifican → `status='validado'`

---

## 6. Diseño de Frontend

### Componentes

- **ConnectionForm.vue** — Ingreso de nombre y room ID
- **ChatRoom.vue** — Sala de chat principal
- **MessageItem.vue** — Mensaje con badge de status
- **PublicKeyDisplay.vue** — Muestra clave pública (debug)

### Diseño Visual

- Paleta: "Encrypted" — Minimalismo Elegante
- Tipografía: Plus Jakarta Sans (headers) + Inter (body)
- Colores: Azul profundo `#2563EB`, Verde `#16A34A`, Rojo `#DC2626`
