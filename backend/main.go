package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"securemessage/config"
	"securemessage/database"
	"securemessage/handlers"
	"securemessage/middleware"
	"securemessage/repository"
	"securemessage/services"
)

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
	tokenStore := middleware.NewTokenStore()

	if err := ciaService.HealthCheck(); err != nil {
		log.Printf("WARNING: CIA API not reachable at http://127.0.0.1:8000 - %v", err)
		log.Printf("Make sure to start it: cd cia-api && go run main.go")
	}

	wsHandler := handlers.NewWebSocketHandler(roomRepo, messageRepo, ciaService, tokenStore)
	msgHandler := handlers.NewMessageHandler(messageRepo, ciaService)
	limiter := middleware.NewRateLimiter(100, time.Minute) // 100 req/min per IP

	// Token endpoint — generates a room access token
	http.HandleFunc("/token", cors(limiter.Middleware(func(w http.ResponseWriter, r *http.Request) {
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
		json.NewEncoder(w).Encode(map[string]string{"token": token, "room": roomName})
	})))

	http.HandleFunc("/ws", wsHandler.HandleWebSocket)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/messages/", cors(limiter.Middleware(func(w http.ResponseWriter, r *http.Request) {
		roomName := strings.TrimPrefix(r.URL.Path, "/messages/")
		if roomName == "" {
			http.Error(w, "roomName required", http.StatusBadRequest)
			return
		}
		msgHandler.GetMessages(w, r, roomName)
	})))

	http.HandleFunc("/decrypt", cors(limiter.Middleware(middleware.MaxBytes(msgHandler.Decrypt, 1<<20))))

	// Graceful shutdown
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
