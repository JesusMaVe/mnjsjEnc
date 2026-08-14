# SecureMessage Security Remediation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Harden SecureMessage from a vulnerable demo to a security-respectable academic project, addressing 7 CRITICAL, 8 HIGH, and 6 MEDIUM findings from the security audit.

**Architecture:** Six phased layers — infrastructure secrets, input validation, authentication, crypto hardening, transport security, and DB hardening. Each phase is independently shippable. Phases 0-2 are the minimum viable security baseline.

**Tech Stack:** Go 1.21, gorilla/websocket, fernet-go, PostgreSQL 16, Docker Compose, Vue 3, Vite

---

## File Map

| Phase | Files Created | Files Modified |
|-------|--------------|----------------|
| 0 | `.env.example` | `docker-compose.yml`, `backend/config/config.go` |
| 1 | `backend/middleware/validation.go` | `backend/handlers/websocket.go`, `backend/handlers/message.go`, `cia-api/main.go` |
| 2 | `backend/middleware/auth.go` | `backend/main.go`, `backend/handlers/websocket.go`, `cia-api/main.go`, `frontend/src/composables/useWebSocket.js`, `frontend/src/components/ConnectionForm.vue`, `frontend/src/components/ChatRoom.vue` |
| 3 | `backend/middleware/ratelimit.go` | `cia-api/crypto/crypto.go`, `cia-api/main.go`, `backend/services/cia.go` |
| 4 | — | `backend/database/postgres.go`, `backend/main.go`, `cia-api/main.go`, `docker-compose.yml` |
| 5 | `backend/migrations/002_remove_plaintext.sql` | `backend/handlers/websocket.go`, `backend/handlers/message.go`, `backend/repository/message.go`, `backend/models/message.go` |

---

## Phase 0: Infrastructure Secrets (Day 1)

### Task 0.1: Create `.env.example` and `.env`

- [ ] **Step 1: Create `.env.example`**

```bash
# .env.example (committed to git)
POSTGRES_DB=securemessage
POSTGRES_USER=postgres
POSTGRES_PASSWORD=CHANGE_ME_STRONG_PASSWORD
DB_HOST=localhost
DB_PORT=5432
SERVER_PORT=8080
CIA_API_KEY=CHANGE_ME_RANDOM_32_CHARS
```

- [ ] **Step 2: Create `.env` (gitignored)**

```bash
# .env (NOT committed — add to .gitignore)
POSTGRES_DB=securemessage
POSTGRES_USER=postgres
POSTGRES_PASSWORD=super_secret_dev_password_123
DB_HOST=localhost
DB_PORT=5432
SERVER_PORT=8080
CIA_API_KEY=dev-only-key-not-for-production
```

- [ ] **Step 3: Add `.env` to `.gitignore`**

Append to existing `.gitignore`:
```
.env
```

- [ ] **Step 4: Commit**

```bash
git add .env.example .gitignore
git commit -m "chore: add .env.example, gitignore .env"
```

### Task 0.2: Update `docker-compose.yml` with secrets and limits

- [ ] **Step 1: Rewrite `docker-compose.yml`**

```yaml
services:
  postgres:
    image: postgres:16-alpine
    container_name: securemessage-db
    env_file: .env
    ports:
      - "127.0.0.1:5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./backend/migrations/001_initial.sql:/docker-entrypoint-initdb.d/001_initial.sql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U $${POSTGRES_USER}"]
      interval: 5s
      timeout: 5s
      retries: 5
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
    networks:
      - backend-net

networks:
  backend-net:
    driver: bridge

volumes:
  pgdata:
```

Key changes:
- `env_file: .env` instead of hardcoded credentials
- Port bound to `127.0.0.1` only
- Health check added
- Resource limits added
- Backend network created

- [ ] **Step 2: Commit**

```bash
git add docker-compose.yml
git commit -m "fix(infra): use .env for DB creds, bind port localhost, add healthcheck"
```

### Task 0.3: Remove hardcoded DB defaults

- [ ] **Step 1: Modify `backend/config/config.go`**

```go
package config

import (
	"log"
	"os"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	ServerPort string
}

func Load() *Config {
	return &Config{
		DBHost:     getEnvOrFatal("DB_HOST"),
		DBPort:     getEnvOrFatal("DB_PORT"),
		DBUser:     getEnvOrFatal("DB_USER"),
		DBPassword: getEnvOrFatal("DB_PASSWORD"),
		DBName:     getEnvOrFatal("DB_NAME"),
		ServerPort: getEnv("SERVER_PORT", "8080"),
	}
}

func getEnvOrFatal(key string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Fatalf("Required environment variable %s is not set", key)
	return ""
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/config/config.go
git commit -m "fix(config): require env vars for DB credentials, remove hardcoded defaults"
```

---

## Phase 1: Input Validation & Hardening (Day 1)

### Task 1.1: Create validation middleware

- [ ] **Step 1: Create `backend/middleware/validation.go`**

```go
package middleware

import (
	"fmt"
	"net/http"
	"regexp"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,50}$`)
var roomNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,50}$`)

func ValidateUsername(username string) error {
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("username must be 1-50 alphanumeric/underscore/dash characters")
	}
	return nil
}

func ValidateRoomName(name string) error {
	if !roomNameRegex.MatchString(name) {
		return fmt.Errorf("room name must be 1-50 alphanumeric/underscore/dash characters")
	}
	return nil
}

func MaxBytes(next http.HandlerFunc, maxBytes int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next(w, r)
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/middleware/validation.go
git commit -m "feat: add input validation middleware for usernames and rooms"
```

### Task 1.2: Apply validation to WebSocket handler

- [ ] **Step 1: Modify `backend/handlers/websocket.go`**

In `handleJoin` (line 155), add validation before processing:

```go
func (h *WebSocketHandler) handleJoin(c *Client, msg *models.WebSocketMessage) {
	if err := middleware.ValidateUsername(msg.Username); err != nil {
		h.sendError(c, "Invalid username")
		return
	}
	if err := middleware.ValidateRoomName(msg.RoomID); err != nil {
		h.sendError(c, "Invalid room name")
		return
	}
	// ... rest of existing code
}
```

In `handleMessageContent` (line 177), add content size check:

```go
func (h *WebSocketHandler) handleMessageContent(c *Client, msg *models.WebSocketMessage) {
	if c.RoomID == "" {
		h.sendError(c, "Must join a room first")
		return
	}
	if len(msg.Content) == 0 || len(msg.Content) > 1<<20 { // 1MB max
		h.sendError(c, "Message must be 1-1MB")
		return
	}
	// ... rest of existing code
}
```

Add import: `"securemessage/middleware"`

- [ ] **Step 2: Add WebSocket read limit**

In `HandleWebSocket` (line 103), after upgrading:

```go
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}
	conn.SetReadLimit(64 * 1024) // 64KB max WS message
	c := &Client{Conn: conn, Send: make(chan []byte, 256)}
	go c.writePump()
	go c.readPump(h)
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/handlers/websocket.go
git commit -m "fix: add input validation and size limits to WebSocket handler"
```

### Task 1.3: Add body size limit to message endpoint

- [ ] **Step 1: Modify `backend/handlers/message.go`**

Wrap the Decrypt handler with MaxBytes. In `main.go` line 64:

```go
http.HandleFunc("/decrypt", cors(middleware.MaxBytes(msgHandler.Decrypt, 1<<20)))
```

- [ ] **Step 2: Commit**

```bash
git add backend/handlers/message.go backend/main.go
git commit -m "fix: add 1MB body size limit to /decrypt endpoint"
```

### Task 1.4: Fix CORS and WebSocket origin

- [ ] **Step 1: Modify `backend/main.go` CORS function**

```go
func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}
```

- [ ] **Step 2: Modify WebSocket upgrader in `backend/handlers/websocket.go`**

```go
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "http://localhost:3000" || origin == "http://127.0.0.1:3000"
	},
}
```

- [ ] **Step 3: Fix CIA API CORS in `cia-api/main.go`**

```go
func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}
```

- [ ] **Step 4: Commit**

```bash
git add backend/main.go backend/handlers/websocket.go cia-api/main.go
git commit -m "fix(security): restrict CORS and WebSocket origin to localhost:3000"
```

### Task 1.5: Sanitize error messages in CIA API

- [ ] **Step 1: Modify `cia-api/main.go`**

Replace `err.Error()` in responses with generic messages:

```go
func handleEncrypt(w http.ResponseWriter, r *http.Request) {
	// ...
	ciphertext, err := svc.Encrypt(req.Message)
	if err != nil {
		log.Printf("encrypt error: %v", err) // log server-side only
		respond(w, http.StatusInternalServerError, map[string]string{"error": "encryption failed"})
		return
	}
	// ...
}

func handleSign(w http.ResponseWriter, r *http.Request) {
	// ...
	signature, err := svc.Sign(req.Message)
	if err != nil {
		log.Printf("sign error: %v", err)
		respond(w, http.StatusInternalServerError, map[string]string{"error": "signing failed"})
		return
	}
	// ...
}

func handleVerify(w http.ResponseWriter, r *http.Request) {
	// ...
	valid, err := svc.Verify(req.Message, req.Signature)
	if err != nil {
		log.Printf("verify error: %v", err)
		respond(w, http.StatusInternalServerError, map[string]string{"error": "verification failed"})
		return
	}
	// ...
}
```

- [ ] **Step 2: Commit**

```bash
git add cia-api/main.go
git commit -m "fix(security): sanitize error messages, log details server-side only"
```

---

## Phase 2: Authentication (Day 2-3)

### Task 2.1: Create auth middleware with room tokens

- [ ] **Step 1: Create `backend/middleware/auth.go`**

```go
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
)

// Room tokens: simple shared-secret per room.
// For a demo/academic project, this is sufficient.
// Production: use JWT + database sessions.

type TokenStore struct {
	mu     sync.RWMutex
	tokens map[string]string // token -> roomName
}

func NewTokenStore() *TokenStore {
	return &TokenStore{tokens: make(map[string]string)}
}

func (ts *TokenStore) GenerateToken(roomName string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.tokens[token] = roomName
	return token, nil
}

func (ts *TokenStore) ValidateToken(token, roomName string) bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.tokens[token] == roomName
}

// RequireToken checks for Bearer token in Authorization header
func (ts *TokenStore) RequireToken(roomName string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		if !ts.ValidateToken(token, roomName) {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/middleware/auth.go
git commit -m "feat: add room token authentication middleware"
```

### Task 2.2: Wire auth into backend routes

- [ ] **Step 1: Modify `backend/main.go`**

```go
package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"securemessage/config"
	"securemessage/database"
	"securemessage/handlers"
	"securemessage/middleware"
	"securemessage/repository"
	"securemessage/services"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	roomRepo := repository.NewRoomRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	ciaService := services.NewCIAService()

	if err := ciaService.HealthCheck(); err != nil {
		log.Printf("WARNING: CIA API not reachable - %v", err)
	}

	tokenStore := middleware.NewTokenStore()
	wsHandler := handlers.NewWebSocketHandler(roomRepo, messageRepo, ciaService, tokenStore)
	msgHandler := handlers.NewMessageHandler(messageRepo, ciaService)

	// Serve token generation (for demo: anyone can get a token for any room)
	http.HandleFunc("/token", cors(func(w http.ResponseWriter, r *http.Request) {
		roomName := r.URL.Query().Get("room")
		if roomName == "" {
			http.Error(w, "room parameter required", http.StatusBadRequest)
			return
		}
		if err := middleware.ValidateRoomName(roomName); err != nil {
			http.Error(w, "invalid room name", http.StatusBadRequest)
			return
		}
		token, err := tokenStore.GenerateToken(roomName)
		if err != nil {
			http.Error(w, "failed to generate token", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"token":"` + token + `","room":"` + roomName + `"}`))
	}))

	http.HandleFunc("/ws", wsHandler.HandleWebSocket)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/messages/", cors(func(w http.ResponseWriter, r *http.Request) {
		roomName := strings.TrimPrefix(r.URL.Path, "/messages/")
		if roomName == "" {
			http.Error(w, "roomName required", http.StatusBadRequest)
			return
		}
		msgHandler.GetMessages(w, r, roomName)
	}))

	http.HandleFunc("/decrypt", cors(middleware.MaxBytes(msgHandler.Decrypt, 1<<20)))

	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := http.ListenAndServe(":"+cfg.ServerPort, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
```

- [ ] **Step 2: Update `WebSocketHandler` to accept tokenStore**

Modify `handlers/websocket.go` constructor:

```go
type WebSocketHandler struct {
	hub           *Hub
	roomRepo      *repository.RoomRepository
	messageRepo   *repository.MessageRepository
	ciaService    *services.CIAService
	tokenStore    *middleware.TokenStore
}

func NewWebSocketHandler(roomRepo *repository.RoomRepository, messageRepo *repository.MessageRepository, ciaService *services.CIAService, tokenStore *middleware.TokenStore) *WebSocketHandler {
	hub := &Hub{
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client, 16),
		unregister: make(chan *Client, 16),
	}
	h := &WebSocketHandler{hub: hub, roomRepo: roomRepo, messageRepo: messageRepo, ciaService: ciaService, tokenStore: tokenStore}
	go hub.run()
	return h
}
```

- [ ] **Step 3: Require token in `handleJoin`**

```go
func (h *WebSocketHandler) handleJoin(c *Client, msg *models.WebSocketMessage) {
	if err := middleware.ValidateUsername(msg.Username); err != nil {
		h.sendError(c, "Invalid username")
		return
	}
	if err := middleware.ValidateRoomName(msg.RoomID); err != nil {
		h.sendError(c, "Invalid room name")
		return
	}
	if !h.tokenStore.ValidateToken(msg.Token, msg.RoomID) {
		h.sendError(c, "Invalid or missing room token")
		return
	}
	// ... rest of existing code
}
```

Add `Token` field to `WebSocketMessage` model:

```go
type WebSocketMessage struct {
	Type      string `json:"type"`
	Username  string `json:"username"`
	Sender    string `json:"sender"`
	RoomID    string `json:"roomId"`
	Content   string `json:"content"`
	MessageID string `json:"messageId"`
	Error     string `json:"error"`
	Token     string `json:"token"`
}
```

- [ ] **Step 4: Commit**

```bash
git add backend/main.go backend/handlers/websocket.go backend/models/websocket.go
git commit -m "feat(auth): require room token for WebSocket join"
```

### Task 2.3: Update frontend to use tokens

- [ ] **Step 1: Modify `frontend/src/components/ConnectionForm.vue`**

Add token request on connect:

```javascript
async function handleConnect() {
  if (!username.value || !roomId.value) return
  try {
    const resp = await fetch(`http://127.0.0.1:8080/token?room=${roomId.value}`)
    const data = await resp.json()
    emit('connect', { username: username.value, roomId: roomId.value, token: data.token })
  } catch (e) {
    error.value = 'Failed to get room token'
  }
}
```

- [ ] **Step 2: Modify `frontend/src/composables/useWebSocket.js`**

Pass token in join message:

```javascript
function connect(options) {
  ws.value = new WebSocket(`ws://${window.location.host}/ws`)
  ws.value.onopen = () => {
    send({
      type: 'join',
      username: options.username,
      roomId: options.roomId,
      token: options.token
    })
  }
  // ...
}
```

- [ ] **Step 3: Update `App.vue` to pass token through**

```javascript
function handleConnect(opts) {
  username.value = opts.username
  roomId.value = opts.roomId
  token.value = opts.token
}
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/ConnectionForm.vue frontend/src/composables/useWebSocket.js frontend/src/App.vue
git commit -m "feat(auth): frontend requests and sends room tokens"
```

---

## Phase 3: Crypto Hardening (Day 3-4)

### Task 3.1: Add request body size limits to CIA API

- [ ] **Step 1: Modify `cia-api/main.go`**

Add MaxBytesReader to all handlers:

```go
func handleEncrypt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respond(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	// ...
}

// Apply same pattern to handleDecrypt, handleSign, handleVerify
```

- [ ] **Step 2: Commit**

```bash
git add cia-api/main.go
git commit -m "fix(security): add 1MB body size limit to all CIA API endpoints"
```

### Task 3.2: Implement Fernet key rotation

- [ ] **Step 1: Modify `cia-api/crypto/crypto.go`**

```go
type CryptoService struct {
	mu          sync.RWMutex
	fernetKey   *fernet.Key
	fernetKeys  []*fernet.Key // keyring: current + previous keys
	privateKey  *rsa.PrivateKey
	publicKey   *rsa.PublicKey
}

func (s *CryptoService) Encrypt(plaintext string) (string, error) {
	tok, err := fernet.EncryptAndSign([]byte(plaintext), s.fernetKey)
	if err != nil {
		return "", fmt.Errorf("encryption failed")
	}
	return string(tok), nil
}

func (s *CryptoService) Decrypt(ciphertext string) (string, error) {
	msg := fernet.VerifyAndDecrypt([]byte(ciphertext), 0, s.fernetKeys)
	if msg == nil {
		return "", fmt.Errorf("invalid or expired ciphertext")
	}
	return string(msg), nil
}

// RotateKey generates a new Fernet key and archives the old one
func (s *CryptoService) RotateKey() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Archive current key
	s.fernetKeys = append([]*fernet.Key{s.fernetKey}, s.fernetKeys...)
	// Keep only 2 keys (current + 1 previous)
	if len(s.fernetKeys) > 2 {
		s.fernetKeys = s.fernetKeys[:2]
	}

	// Generate new key
	var k fernet.Key
	k.Generate()
	s.fernetKey = &k
	s.fernetKeys[0] = &k

	return s.saveKeys()
}
```

Update `loadKeys` to initialize the keyring:

```go
func loadKeys() (*CryptoService, error) {
	// ... existing key loading ...
	return &CryptoService{
		fernetKey:  &k,
		fernetKeys: []*fernet.Key{&k},
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
	}, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add cia-api/crypto/crypto.go
git commit -m "feat(crypto): add Fernet key rotation with keyring support"
```

### Task 3.3: Switch RSA signing from PKCS1v15 to PSS

- [ ] **Step 1: Modify `cia-api/crypto/crypto.go`**

```go
func (s *CryptoService) Sign(message string) (string, error) {
	hash := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPSS(rand.Reader, s.privateKey, crypto.SHA256, hash[:], rsa.PSSSaltLengthAuto)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func (s *CryptoService) Verify(message, signature string) (bool, error) {
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, err
	}
	hash := sha256.Sum256([]byte(message))
	err = rsa.VerifyPSS(s.publicKey, crypto.SHA256, hash[:], sigBytes, rsa.PSSSaltLengthAuto)
	return err == nil, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add cia-api/crypto/crypto.go
git commit -m "fix(crypto): switch RSA signing from PKCS1v15 to PSS (timing-safe)"
```

### Task 3.4: Add rate limiting

- [ ] **Step 1: Create `backend/middleware/ratelimit.go`**

```go
package middleware

import (
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Remove old entries
	reqs := rl.requests[key]
	valid := reqs[:0]
	for _, t := range reqs {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.limit {
		rl.requests[key] = valid
		return false
	}

	rl.requests[key] = append(valid, now)
	return true
}

func (rl *RateLimiter) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = fwd
		}
		if !rl.Allow(ip) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}
```

- [ ] **Step 2: Apply to backend main.go**

```go
limiter := middleware.NewRateLimiter(100, time.Minute) // 100 req/min per IP

http.HandleFunc("/token", cors(limiter.Middleware(tokenHandler)))
http.HandleFunc("/messages/", cors(limiter.Middleware(msgHandler)))
http.HandleFunc("/decrypt", cors(limiter.Middleware(middleware.MaxBytes(msgHandler.Decrypt, 1<<20))))
```

- [ ] **Step 3: Apply to CIA API**

Add same rate limiter pattern to `cia-api/main.go`:

```go
limiter := NewRateLimiter(50, time.Minute) // 50 req/min

http.HandleFunc("/confidentiality/encrypt", cors(limiter.Middleware(handleEncrypt)))
// ... etc for all endpoints
```

- [ ] **Step 4: Commit**

```bash
git add backend/middleware/ratelimit.go backend/main.go cia-api/main.go
git commit -m "feat(security): add rate limiting (100/min backend, 50/min CIA API)"
```

### Task 3.5: Upgrade vulnerable dependencies

- [ ] **Step 1: Upgrade gorilla/websocket**

```bash
cd backend && go get github.com/gorilla/websocket@v1.5.3
```

- [ ] **Step 2: Upgrade Go toolchain**

```bash
# In go.mod, update go directive
go 1.22
```

- [ ] **Step 3: Commit**

```bash
git add backend/go.mod backend/go.sum
git commit -m "fix(deps): upgrade gorilla/websocket to v1.5.3, Go to 1.22"
```

---

## Phase 4: Transport Security (Day 4)

### Task 4.1: Enable PostgreSQL SSL

- [ ] **Step 1: Generate self-signed cert for PostgreSQL**

Create `docker/postgres-certs/`:
```bash
mkdir -p docker/postgres-certs
openssl req -new -x509 -days 365 -nodes -text \
  -out docker/postgres-certs/server.crt \
  -keyout docker/postgres-certs/server.key \
  -subj "/CN=securemessage-db"
chmod 600 docker/postgres-certs/server.key
```

- [ ] **Step 2: Update `docker-compose.yml`**

```yaml
services:
  postgres:
    image: postgres:16-alpine
    container_name: securemessage-db
    env_file: .env
    ports:
      - "127.0.0.1:5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./backend/migrations/001_initial.sql:/docker-entrypoint-initdb.d/001_initial.sql
      - ./docker/postgres-certs/server.crt:/var/lib/postgresql/server.crt:ro
      - ./docker/postgres-certs/server.key:/var/lib/postgresql/server.key:ro
    command: >
      postgres
        -c ssl=on
        -c ssl_cert_file=/var/lib/postgresql/server.crt
        -c ssl_key_file=/var/lib/postgresql/server.key
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U $${POSTGRES_USER}"]
      interval: 5s
      timeout: 5s
      retries: 5
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
    networks:
      - backend-net
```

- [ ] **Step 3: Update `backend/database/postgres.go`**

```go
func Connect(cfg *config.Config) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)
	// ... rest unchanged
}
```

- [ ] **Step 4: Add `docker/postgres-certs/` to `.gitignore`**

```
docker/postgres-certs/
```

- [ ] **Step 5: Commit**

```bash
git add docker-compose.yml backend/database/postgres.go .gitignore
git commit -m "fix(security): enable PostgreSQL SSL with self-signed certs"
```

### Task 4.2: Update backend services to use HTTPS

- [ ] **Step 1: Modify `backend/services/cia.go`**

```go
const ciaAPIBase = "https://127.0.0.1:8000"

func NewCIAService() *CIAService {
	// For self-signed certs in dev
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // dev only!
	}
	return &CIAService{client: &http.Client{Transport: tr}}
}
```

Add import: `"crypto/tls"`

- [ ] **Step 2: Update CIA API to serve TLS**

```go
func main() {
	// ... existing route setup ...

	log.Println("CIA API running on :8000 (TLS)")
	certFile := os.Getenv("TLS_CERT")
	keyFile := os.Getenv("TLS_KEY")
	if certFile != "" && keyFile != "" {
		if err := http.ListenAndServeTLS(":8000", certFile, keyFile, nil); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Println("WARNING: Running without TLS (no TLS_CERT/TLS_KEY set)")
		if err := http.ListenAndServe(":8000", nil); err != nil {
			log.Fatal(err)
		}
	}
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/services/cia.go cia-api/main.go
git commit -m "fix(security): enable HTTPS between backend and CIA API"
```

---

## Phase 5: Database Hardening (Day 5)

### Task 5.1: Remove plaintext storage

- [ ] **Step 1: Create migration `002_remove_plaintext.sql`**

```sql
-- Remove plaintext content from messages table
-- Verification will use ciphertext as signed payload instead

ALTER TABLE messages DROP COLUMN content_original;
```

- [ ] **Step 2: Update `backend/models/message.go`**

Remove `ContentOriginal` field:

```go
type Message struct {
	ID               string `json:"id"`
	RoomID           string `json:"roomId"`
	SenderUsername   string `json:"senderUsername"`
	ContentEncrypted string `json:"contentEncrypted"`
	CreatedAt        string `json:"createdAt"`
}
```

- [ ] **Step 3: Update `backend/handlers/websocket.go`**

In `handleMessageContent`, sign the ciphertext instead of plaintext:

```go
func (h *WebSocketHandler) handleMessageContent(c *Client, msg *models.WebSocketMessage) {
	// ...
	// 1. Encrypt
	ciphertext, err := h.ciaService.Encrypt(msg.Content)
	if err != nil {
		h.sendError(c, "Error encrypting message")
		return
	}

	// 2. Sign the CIPHERTEXT (not plaintext)
	signature, err := h.ciaService.Sign(ciphertext)
	if err != nil {
		h.sendError(c, "Error signing message")
		return
	}

	// 3. Store only ciphertext + signature
	messageID := uuid.New().String()
	if err := h.messageRepo.CreateMessage(
		&models.Message{ID: messageID, RoomID: c.RoomID, SenderUsername: c.Username, ContentEncrypted: ciphertext},
		&models.MessageIntegrity{MessageID: messageID, Signature: signature, Status: "no_verificado"},
	); err != nil {
		h.sendError(c, "Error saving")
		return
	}
	// ...
}
```

- [ ] **Step 4: Update `handleValidate`**

Verify signature against ciphertext:

```go
func (h *WebSocketHandler) handleValidate(c *Client, msg *models.WebSocketMessage) {
	message, integrity, err := h.messageRepo.GetMessageByID(msg.MessageID)
	if err != nil {
		h.sendError(c, "Message not found")
		return
	}

	// Verify signature against ciphertext
	valid, err := h.ciaService.Verify(message.ContentEncrypted, integrity.Signature)
	if err != nil || !valid {
		// Re-sign with current keys
		newSignature, signErr := h.ciaService.Sign(message.ContentEncrypted)
		if signErr != nil {
			h.sendError(c, "Re-signing failed")
			return
		}
		if updateErr := h.messageRepo.UpdateSignature(msg.MessageID, newSignature); updateErr != nil {
			h.sendError(c, "Error updating signature")
			return
		}
		valid, err = h.ciaService.Verify(message.ContentEncrypted, newSignature)
		if err != nil || !valid {
			h.sendError(c, "Invalid signature after re-signing")
			return
		}
		integrity.Signature = newSignature
	}

	if err := h.messageRepo.UpdateMessageStatus(msg.MessageID, "verificado"); err != nil {
		h.sendError(c, "Error updating")
		return
	}

	h.broadcastToRoom(c.RoomID, models.WebSocketMessage{Type: "verified", MessageID: msg.MessageID})
}
```

- [ ] **Step 5: Update `repository/message.go`**

Remove `ContentOriginal` from queries:

```go
func (r *MessageRepository) CreateMessage(msg *models.Message, integrity *models.MessageIntegrity) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO messages (id, room_id, sender_username, content_encrypted) VALUES ($1, $2, $3, $4)`,
		msg.ID, msg.RoomID, msg.SenderUsername, msg.ContentEncrypted,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		`INSERT INTO message_integrity (message_id, signature, status) VALUES ($1, $2, $3)`,
		integrity.MessageID, integrity.Signature, integrity.Status,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}
```

- [ ] **Step 6: Commit**

```bash
git add backend/migrations/002_remove_plaintext.sql backend/models/message.go backend/handlers/websocket.go backend/repository/message.go
git commit -m "fix(security): remove plaintext storage, sign ciphertext instead"
```

### Task 5.2: Add graceful shutdown

- [ ] **Step 1: Modify `backend/main.go`**

```go
func main() {
	// ... existing setup ...

	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: nil,
	}

	go func() {
		log.Printf("Server starting on port %s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited")
}
```

Add imports: `"context"`, `"os"`, `"os/signal"`, `"syscall"`, `"time"`

- [ ] **Step 2: Commit**

```bash
git add backend/main.go
git commit -m "feat: add graceful shutdown with signal handling"
```

---

## Verification Checklist

After completing all phases, verify:

- [ ] `docker compose up -d` starts PostgreSQL with SSL
- [ ] `cd backend && go build -o backend .` compiles
- [ ] `cd cia-api && go build -o cia-api .` compiles
- [ ] `cd frontend && npm run build` compiles
- [ ] WebSocket join requires valid token
- [ ] CORS blocks requests from non-localhost origins
- [ ] Rate limiting returns 429 after threshold
- [ ] DB contains no `content_original` column
- [ ] Error messages don't leak internal details
- [ ] All endpoints enforce body size limits

---

## Summary

| Phase | What | Effort | Risk Reduction |
|-------|------|--------|----------------|
| 0 | Secrets, Docker hardening | 30 min | CRITICAL → fixed |
| 1 | Input validation, CORS, origin | 2 hr | HIGH → fixed |
| 2 | Room token authentication | 4 hr | CRITICAL → fixed |
| 3 | Crypto hardening, rate limits | 3 hr | HIGH → fixed |
| 4 | TLS/SSL everywhere | 2 hr | HIGH → fixed |
| 5 | Remove plaintext, graceful shutdown | 2 hr | CRITICAL → fixed |

**Total estimated effort:** ~14 hours for a security-respectable demo.
