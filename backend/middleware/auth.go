// Package middleware provides HTTP middleware for auth, rate limiting, and validation.
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
)

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
