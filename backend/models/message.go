// Package models defines the domain types for SecureMessage.
package models

import "time"

type Message struct {
	ID               string    `json:"id"`
	RoomID           string    `json:"roomId"`
	SenderUsername   string    `json:"sender"`
	ContentEncrypted string    `json:"contentEncrypted"`
	CreatedAt        time.Time `json:"createdAt"`
}

type MessageIntegrity struct {
	MessageID   string     `json:"messageId"`
	Signature   string     `json:"signature"`
	Status      string     `json:"status"`
	ValidatedAt *time.Time `json:"validatedAt,omitempty"`
}

type MessageResponse struct {
	ID        string    `json:"id"`
	Sender    string    `json:"sender"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}
