-- Phase 5: Remove plaintext content from messages table
-- Verification now uses ciphertext as signed payload instead of plaintext

ALTER TABLE messages DROP COLUMN content_original;
