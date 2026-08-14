// Package handlers implements HTTP and WebSocket request handlers.
package handlers

import (
	"encoding/json"
	"net/http"
	"securemessage/repository"
	"securemessage/services"
)

type MessageHandler struct {
	messageRepo *repository.MessageRepository
	ciaService  *services.CIAService
}

func NewMessageHandler(messageRepo *repository.MessageRepository, ciaService *services.CIAService) *MessageHandler {
	return &MessageHandler{messageRepo: messageRepo, ciaService: ciaService}
}

// GetMessages retorna los mensajes de una sala (cifrados + firma + status)
func (h *MessageHandler) GetMessages(w http.ResponseWriter, r *http.Request, roomName string) {
	messages, err := h.messageRepo.GetMessagesByRoom(roomName)
	if err != nil {
		http.Error(w, "Error fetching messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(messages)
}

// Decrypt descifra un ciphertext via CIA API (proxy seguro)
func (h *MessageHandler) Decrypt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Ciphertext string `json:"ciphertext"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	plaintext, err := h.ciaService.Decrypt(req.Ciphertext)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid ciphertext"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": plaintext})
}
