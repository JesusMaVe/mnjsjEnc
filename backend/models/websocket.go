package models

type WebSocketMessage struct {
	Type      string `json:"type"`
	Username  string `json:"username,omitempty"`
	Sender    string `json:"sender,omitempty"`
	RoomID    string `json:"roomId,omitempty"`
	Content   string `json:"content,omitempty"`
	MessageID string `json:"messageId,omitempty"`
	Error     string `json:"error,omitempty"`
}
