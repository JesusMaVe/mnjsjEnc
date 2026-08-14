package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"securemessage-cia/availability"
	"securemessage-cia/crypto"
	"sync"
	"time"
)

var (
	svc     = crypto.New()
	cluster = availability.NewCluster(3, 0.3)
)

type Request struct {
	Message    string `json:"message"`
	Ciphertext string `json:"ciphertext"`
	Signature  string `json:"signature"`
}

// Simple rate limiter
type rateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{requests: make(map[string][]time.Time), limit: limit, window: window}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	start := now.Add(-rl.window)
	reqs := rl.requests[key]
	valid := reqs[:0]
	for _, t := range reqs {
		if t.After(start) {
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

var limiter = newRateLimiter(50, time.Minute) // 50 req/min per IP

func rateLimited(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = fwd
		}
		if !limiter.allow(ip) {
			respond(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			return
		}
		next(w, r)
	}
}

func respond(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

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
	ciphertext, err := svc.Encrypt(req.Message)
	if err != nil {
		log.Printf("encrypt error: %v", err)
		respond(w, http.StatusInternalServerError, map[string]string{"error": "encryption failed"})
		return
	}
	respond(w, http.StatusOK, map[string]string{"ciphertext": ciphertext})
}

func handleDecrypt(w http.ResponseWriter, r *http.Request) {
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
	plaintext, err := svc.Decrypt(req.Ciphertext)
	if err != nil {
		respond(w, http.StatusForbidden, map[string]string{"error": "invalid ciphertext"})
		return
	}
	respond(w, http.StatusOK, map[string]string{"message": plaintext})
}

func handleSign(w http.ResponseWriter, r *http.Request) {
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
	signature, err := svc.Sign(req.Message)
	if err != nil {
		log.Printf("sign error: %v", err)
		respond(w, http.StatusInternalServerError, map[string]string{"error": "signing failed"})
		return
	}
	respond(w, http.StatusOK, map[string]string{"message": req.Message, "signature": signature})
}

func handleVerify(w http.ResponseWriter, r *http.Request) {
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
	valid, err := svc.Verify(req.Message, req.Signature)
	if err != nil {
		log.Printf("verify error: %v", err)
		respond(w, http.StatusInternalServerError, map[string]string{"error": "verification failed"})
		return
	}
	respond(w, http.StatusOK, map[string]bool{"valid": valid})
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]string{"message": "CIA Triad API running (Go)"})
}

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

func main() {
	http.HandleFunc("/confidentiality/encrypt", cors(rateLimited(handleEncrypt)))
	http.HandleFunc("/confidentiality/decrypt", cors(rateLimited(handleDecrypt)))
	http.HandleFunc("/integrity/sign", cors(rateLimited(handleSign)))
	http.HandleFunc("/integrity/verify", cors(rateLimited(handleVerify)))
	http.HandleFunc("/availability/status", cors(cluster.HandleStatus))
	http.HandleFunc("/availability/request", cors(cluster.HandleRequest))
	http.HandleFunc("/", cors(handleRoot))

	certFile := os.Getenv("TLS_CERT")
	keyFile := os.Getenv("TLS_KEY")

	if certFile != "" && keyFile != "" {
		log.Println("CIA API running on :8000 (TLS)")
		if err := http.ListenAndServeTLS(":8000", certFile, keyFile, nil); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Println("CIA API running on :8000 (no TLS — set TLS_CERT/TLS_KEY for production)")
		if err := http.ListenAndServe(":8000", nil); err != nil {
			log.Fatal(err)
		}
	}
}
