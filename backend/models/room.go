package models

import "time"

type Room struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type RoomUser struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"roomId"`
	Username  string    `json:"username"`
	PublicKey string    `json:"publicKey"`
	JoinedAt time.Time `json:"joinedAt"`
}
