-- SecureMessage Database Schema

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE rooms (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE room_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    room_id UUID REFERENCES rooms(id) ON DELETE CASCADE,
    username VARCHAR(50) NOT NULL,
    public_key TEXT NOT NULL,
    joined_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(room_id, username)
);

CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    room_id UUID REFERENCES rooms(id) ON DELETE CASCADE,
    sender_username VARCHAR(50) NOT NULL,
    content_original TEXT NOT NULL,        -- texto plano para firma/verificación
    content_encrypted TEXT NOT NULL,       -- ciphertext de CIA API
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE message_integrity (
    message_id UUID PRIMARY KEY REFERENCES messages(id) ON DELETE CASCADE,
    signature TEXT NOT NULL,               -- firma de CIA API
    status VARCHAR(20) DEFAULT 'no_verificado'
        CHECK (status IN ('no_verificado', 'verificado')),
    validated_at TIMESTAMP
);
