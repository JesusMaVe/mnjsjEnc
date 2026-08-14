package repository

import (
	"database/sql"
	"securemessage/models"
)

type MessageRepository struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) CreateMessage(msg *models.Message, integrity *models.MessageIntegrity) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO messages (id, room_id, sender_username, content_encrypted) VALUES ($1, $2, $3, $4)`,
		msg.ID, msg.RoomID, msg.SenderUsername, msg.ContentEncrypted,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		`INSERT INTO message_integrity (message_id, signature, status) VALUES ($1, $2, $3)`,
		integrity.MessageID, integrity.Signature, integrity.Status,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *MessageRepository) GetMessagesByRoom(roomName string) ([]models.MessageResponse, error) {
	rows, err := r.db.Query(
		`SELECT m.id, m.sender_username, m.content_encrypted, mi.status, m.created_at
		 FROM messages m
		 JOIN message_integrity mi ON m.id = mi.message_id
		 JOIN rooms r ON m.room_id = r.id
		 WHERE r.name = $1
		 ORDER BY m.created_at ASC`,
		roomName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.MessageResponse
	for rows.Next() {
		var msg models.MessageResponse
		err := rows.Scan(&msg.ID, &msg.Sender, &msg.Content, &msg.Status, &msg.CreatedAt)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (r *MessageRepository) GetMessageByID(messageID string) (*models.Message, *models.MessageIntegrity, error) {
	var msg models.Message
	var integrity models.MessageIntegrity

	err := r.db.QueryRow(
		`SELECT id, room_id, sender_username, content_encrypted FROM messages WHERE id = $1`,
		messageID,
	).Scan(&msg.ID, &msg.RoomID, &msg.SenderUsername, &msg.ContentEncrypted)
	if err != nil {
		return nil, nil, err
	}

	err = r.db.QueryRow(
		`SELECT message_id, signature, status FROM message_integrity WHERE message_id = $1`,
		messageID,
	).Scan(&integrity.MessageID, &integrity.Signature, &integrity.Status)
	if err != nil {
		return nil, nil, err
	}

	return &msg, &integrity, nil
}

func (r *MessageRepository) UpdateMessageStatus(messageID, status string) error {
	_, err := r.db.Exec(
		`UPDATE message_integrity SET status = $1, validated_at = NOW() WHERE message_id = $2`,
		status, messageID,
	)
	return err
}

func (r *MessageRepository) UpdateSignature(messageID, signature string) error {
	_, err := r.db.Exec(
		`UPDATE message_integrity SET signature = $1 WHERE message_id = $2`,
		signature, messageID,
	)
	return err
}
