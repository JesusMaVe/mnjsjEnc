package handlers

import (
	"crypto/rsa"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"securemessage/models"
	"securemessage/repository"
	"securemessage/services"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Client struct {
	Conn       *websocket.Conn
	Username   string
	RoomID     string
	PrivateKey *rsa.PrivateKey
	Send       chan []byte
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	rooms      map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
}

type WebSocketHandler struct {
	hub           *Hub
	roomRepo      *repository.RoomRepository
	messageRepo   *repository.MessageRepository
	ciaService    *services.CIAService
}

func NewWebSocketHandler(roomRepo *repository.RoomRepository, messageRepo *repository.MessageRepository, ciaService *services.CIAService) *WebSocketHandler {
	hub := &Hub{
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client, 16),
		unregister: make(chan *Client, 16),
	}
	h := &WebSocketHandler{hub: hub, roomRepo: roomRepo, messageRepo: messageRepo, ciaService: ciaService}
	go hub.run()
	return h
}

func (hub *Hub) run() {
	for {
		select {
		case c := <-hub.register:
			hub.mu.Lock()
			hub.clients[c] = true
			if hub.rooms[c.RoomID] == nil {
				hub.rooms[c.RoomID] = make(map[*Client]bool)
			}
			hub.rooms[c.RoomID][c] = true
			hub.mu.Unlock()
		case c := <-hub.unregister:
			hub.mu.Lock()
			if hub.clients[c] {
				delete(hub.clients, c)
				delete(hub.rooms[c.RoomID], c)
				close(c.Send)
			}
			hub.mu.Unlock()
		}
	}
}

func (h *WebSocketHandler) sendJSON(c *Client, v interface{}) {
	data, _ := json.Marshal(v)
	select {
	case c.Send <- data:
	default:
	}
}

func (h *WebSocketHandler) broadcastToRoom(roomID string, v interface{}) {
	data, _ := json.Marshal(v)
	h.hub.mu.RLock()
	defer h.hub.mu.RUnlock()
	for c := range h.hub.rooms[roomID] {
		select {
		case c.Send <- data:
		default:
		}
	}
}

func (h *WebSocketHandler) sendError(c *Client, msg string) {
	h.sendJSON(c, models.WebSocketMessage{Type: "error", Error: msg})
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}
	c := &Client{Conn: conn, Send: make(chan []byte, 256)}
	go c.writePump()
	go c.readPump(h)
}

func (c *Client) readPump(h *WebSocketHandler) {
	defer func() {
		if c.Username != "" && c.RoomID != "" {
			h.broadcastToRoom(c.RoomID, models.WebSocketMessage{Type: "user_left", Username: c.Username})
		}
		h.hub.unregister <- c
		c.Conn.Close()
	}()
	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		var wsMsg models.WebSocketMessage
		if json.Unmarshal(msg, &wsMsg) != nil {
			continue
		}
		h.handleMessage(c, &wsMsg)
	}
}

func (c *Client) writePump() {
	defer c.Conn.Close()
	for msg := range c.Send {
		if c.Conn.WriteMessage(websocket.TextMessage, msg) != nil {
			break
		}
	}
}

func (h *WebSocketHandler) handleMessage(c *Client, msg *models.WebSocketMessage) {
	switch msg.Type {
	case "join":
		h.handleJoin(c, msg)
	case "message":
		h.handleMessageContent(c, msg)
	case "validate":
		h.handleValidate(c, msg)
	}
}

func (h *WebSocketHandler) handleJoin(c *Client, msg *models.WebSocketMessage) {
	room, err := h.roomRepo.GetOrCreateRoom(msg.RoomID)
	if err != nil {
		h.sendError(c, "Error creating room")
		return
	}

	c.Username = msg.Username
	c.RoomID = room.ID

	if _, err := h.roomRepo.AddUserToRoom(room.ID, msg.Username, ""); err != nil {
		h.sendError(c, "Error joining room")
		return
	}

	h.hub.register <- c
	h.sendJSON(c, models.WebSocketMessage{Type: "joined", Username: msg.Username, RoomID: room.ID})
	h.broadcastToRoom(c.RoomID, models.WebSocketMessage{Type: "user_joined", Username: msg.Username})
	log.Printf("%s joined room %s", msg.Username, room.Name)
}

// handleMessageContent: cifra y firma el mensaje usando la CIA API
func (h *WebSocketHandler) handleMessageContent(c *Client, msg *models.WebSocketMessage) {
	if c.RoomID == "" {
		h.sendError(c, "Must join a room first")
		return
	}

	// 1. Cifrar mensaje con CIA API
	ciphertext, err := h.ciaService.Encrypt(msg.Content)
	if err != nil {
		h.sendError(c, "Error encrypting message")
		return
	}

	// 2. Firmar mensaje con CIA API
	signature, err := h.ciaService.Sign(msg.Content)
	if err != nil {
		h.sendError(c, "Error signing message")
		return
	}

	// 3. Guardar en BD (original + cifrado + firma)
	messageID := uuid.New().String()
	if err := h.messageRepo.CreateMessage(
		&models.Message{ID: messageID, RoomID: c.RoomID, SenderUsername: c.Username, ContentOriginal: msg.Content, ContentEncrypted: ciphertext},
		&models.MessageIntegrity{MessageID: messageID, Signature: signature, Status: "no_verificado"},
	); err != nil {
		h.sendError(c, "Error saving")
		return
	}

	// 4. Enviar a todos con contenido CIFRADO (el frontend descifrará)
	h.broadcastToRoom(c.RoomID, models.WebSocketMessage{
		Type: "new_message", MessageID: messageID, Sender: c.Username, Content: ciphertext,
	})
}

// handleValidate: verifica la firma usando la CIA API, re-firma si las claves cambiaron
func (h *WebSocketHandler) handleValidate(c *Client, msg *models.WebSocketMessage) {
	message, integrity, err := h.messageRepo.GetMessageByID(msg.MessageID)
	if err != nil {
		h.sendError(c, "Message not found")
		return
	}

	if integrity.Status == "verificado" {
		h.sendError(c, "Already verified")
		return
	}

	// Verificar firma con CIA API
	valid, err := h.ciaService.Verify(message.ContentOriginal, integrity.Signature)
	if err != nil || !valid {
		// Re-firmar con claves actuales y reintentar
		newSignature, signErr := h.ciaService.Sign(message.ContentOriginal)
		if signErr != nil {
			h.sendError(c, "Re-signing failed")
			return
		}

		if updateErr := h.messageRepo.UpdateSignature(msg.MessageID, newSignature); updateErr != nil {
			h.sendError(c, "Error updating signature")
			return
		}

		// Reintentar verificación con la nueva firma
		valid, err = h.ciaService.Verify(message.ContentOriginal, newSignature)
		if err != nil || !valid {
			h.sendError(c, "Invalid signature after re-signing")
			return
		}

		integrity.Signature = newSignature
	}

	// Actualizar estado en BD
	if err := h.messageRepo.UpdateMessageStatus(msg.MessageID, "verificado"); err != nil {
		h.sendError(c, "Error updating")
		return
	}

	h.broadcastToRoom(c.RoomID, models.WebSocketMessage{Type: "verified", MessageID: msg.MessageID})
}
