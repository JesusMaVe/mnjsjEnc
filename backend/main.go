package main

import (
	"log"
	"net/http"
	"strings"

	"securemessage/config"
	"securemessage/database"
	"securemessage/handlers"
	"securemessage/repository"
	"securemessage/services"
)

func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
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

	if err := ciaService.HealthCheck(); err != nil {
		log.Printf("WARNING: CIA API not reachable at http://127.0.0.1:8000 - %v", err)
		log.Printf("Make sure to start it: cd cia-api && go run main.go")
	}

	wsHandler := handlers.NewWebSocketHandler(roomRepo, messageRepo, ciaService)
	msgHandler := handlers.NewMessageHandler(messageRepo, ciaService)

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

	http.HandleFunc("/decrypt", cors(msgHandler.Decrypt))

	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := http.ListenAndServe(":"+cfg.ServerPort, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
