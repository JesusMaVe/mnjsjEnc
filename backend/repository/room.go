package repository

import (
	"database/sql"
	"securemessage/models"
)

type RoomRepository struct {
	db *sql.DB
}

func NewRoomRepository(db *sql.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

func (r *RoomRepository) GetOrCreateRoom(name string) (*models.Room, error) {
	var room models.Room

	err := r.db.QueryRow(
		`SELECT id, name, created_at FROM rooms WHERE name = $1`, name,
	).Scan(&room.ID, &room.Name, &room.CreatedAt)
	if err == nil {
		return &room, nil
	}

	err = r.db.QueryRow(
		`INSERT INTO rooms (name) VALUES ($1) RETURNING id, name, created_at`, name,
	).Scan(&room.ID, &room.Name, &room.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &room, nil
}

func (r *RoomRepository) AddUserToRoom(roomID, username, publicKey string) (*models.RoomUser, error) {
	var user models.RoomUser

	err := r.db.QueryRow(
		`INSERT INTO room_users (room_id, username, public_key) VALUES ($1, $2, $3)
		 ON CONFLICT (room_id, username) DO UPDATE SET public_key = $3
		 RETURNING id, room_id, username, public_key, joined_at`,
		roomID, username, publicKey,
	).Scan(&user.ID, &user.RoomID, &user.Username, &user.PublicKey, &user.JoinedAt)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *RoomRepository) GetUserByUsername(roomID, username string) (*models.RoomUser, error) {
	var user models.RoomUser

	err := r.db.QueryRow(
		`SELECT id, room_id, username, public_key, joined_at
		 FROM room_users WHERE room_id = $1 AND username = $2`,
		roomID, username,
	).Scan(&user.ID, &user.RoomID, &user.Username, &user.PublicKey, &user.JoinedAt)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *RoomRepository) GetUsersInRoom(roomID string) ([]models.RoomUser, error) {
	rows, err := r.db.Query(
		`SELECT id, room_id, username, public_key, joined_at
		 FROM room_users WHERE room_id = $1`,
		roomID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.RoomUser
	for rows.Next() {
		var user models.RoomUser
		err := rows.Scan(&user.ID, &user.RoomID, &user.Username, &user.PublicKey, &user.JoinedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}
