package database

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

// SendMessage inserts a new message and updates the conversation's "last message" preview.
func (db *appdbimpl) SendMessage(conversationID, senderID, text, photo, replyToID string) (Message, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return Message{}, err
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)

	if _, err := db.c.Exec(
		`INSERT INTO messages (id, conversation_id, sender_id, text, photo, timestamp, reply_to_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id.String(), conversationID, senderID, text, photo, timestamp, replyToID,
	); err != nil {
		return Message{}, fmt.Errorf("error sending message: %w", err)
	}

	// Update the chat list preview (left sidebar will show this string).
	preview := text
	if preview == "" && photo != "" {
		preview = "[Photo]"
	}
	if _, err := db.c.Exec(
		`UPDATE conversations SET last_message = ?, last_time = ? WHERE id = ?`,
		preview, timestamp, conversationID,
	); err != nil {
		return Message{}, err
	}

	return Message{
		ID:             id.String(),
		ConversationID: conversationID,
		SenderID:       senderID,
		Text:           text,
		Photo:          photo,
		Timestamp:      timestamp,
		Status:         "sent",
		ReplyToID:      replyToID,
		Comments:       []Comment{},
	}, nil
}

// GetMessages returns every message in the conversation, with delivery status
// computed from message_reads + the conversation's member list:
//   - sent      → no other members read it yet
//   - delivered → every other member exists (not us alone)
//   - read      → every other member has it in message_reads
//
// Sender name is joined in for display.
func (db *appdbimpl) GetMessages(conversationID string) ([]Message, error) {
	rows, err := db.c.Query(`
		SELECT
			m.id, m.conversation_id, m.sender_id,
			COALESCE(u.name, m.sender_id) AS sender_name,
			m.text, m.photo, m.timestamp,
			COALESCE(m.reply_to_id, '') AS reply_to_id,
			COALESCE(m.forwarded, 0) AS forwarded,
			CASE
				WHEN (SELECT COUNT(*) FROM message_reads r
				      WHERE r.message_id = m.id AND r.user_id != m.sender_id) >=
				     (SELECT COUNT(*) FROM members mem
				      WHERE mem.conversation_id = m.conversation_id AND mem.user_id != m.sender_id)
				     AND (SELECT COUNT(*) FROM members mem
				          WHERE mem.conversation_id = m.conversation_id AND mem.user_id != m.sender_id) > 0
				THEN 'read'
				WHEN (SELECT COUNT(*) FROM members mem
				      WHERE mem.conversation_id = m.conversation_id AND mem.user_id != m.sender_id) > 0
				THEN 'delivered'
				ELSE 'sent'
			END AS status
		FROM messages m
		LEFT JOIN users u ON m.sender_id = u.id
		WHERE m.conversation_id = ?
		ORDER BY m.timestamp ASC`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]Message, 0)
	idx := make(map[string]int)
	for rows.Next() {
		var m Message
		var forwarded int
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.SenderName,
			&m.Text, &m.Photo, &m.Timestamp, &m.ReplyToID, &forwarded, &m.Status); err != nil {
			return nil, err
		}
		m.Forwarded = forwarded == 1
		m.Comments = []Comment{}
		idx[m.ID] = len(messages)
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		return messages, nil
	}

	// Batch-fetch comments (emoji reactions) for all messages in one query.
	ids := make([]string, len(messages))
	for i, m := range messages {
		ids[i] = m.ID
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	crows, err := db.c.Query(
		`SELECT c.id, c.message_id, c.user_id, COALESCE(u.name, c.user_id), c.emoticon
		 FROM comments c LEFT JOIN users u ON c.user_id = u.id
		 WHERE c.message_id IN (`+placeholders+`)`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer crows.Close()

	for crows.Next() {
		var c Comment
		if err := crows.Scan(&c.ID, &c.MessageID, &c.UserID, &c.UserName, &c.Emoticon); err != nil {
			return nil, err
		}
		if i, ok := idx[c.MessageID]; ok {
			messages[i].Comments = append(messages[i].Comments, c)
		}
	}
	return messages, crows.Err()
}

// GetMessage returns a single message (used to check authorship before delete,
// and to read the source message when forwarding).
func (db *appdbimpl) GetMessage(messageID string) (Message, error) {
	var m Message
	var forwarded int
	err := db.c.QueryRow(`
		SELECT id, conversation_id, sender_id, text, photo, timestamp, status, COALESCE(reply_to_id, ''), COALESCE(forwarded, 0)
		FROM messages WHERE id = ?`, messageID).
		Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Text, &m.Photo, &m.Timestamp, &m.Status, &m.ReplyToID, &forwarded)
	if err != nil {
		return Message{}, fmt.Errorf("message not found: %w", err)
	}
	m.Forwarded = forwarded == 1
	m.Comments = []Comment{}
	return m, nil
}

// DeleteMessage removes a message and any reactions on it.
func (db *appdbimpl) DeleteMessage(messageID string) error {
	if _, err := db.c.Exec(`DELETE FROM comments WHERE message_id = ?`, messageID); err != nil {
		return err
	}
	_, err := db.c.Exec(`DELETE FROM messages WHERE id = ?`, messageID)
	return err
}

// ForwardMessage copies an existing message into another conversation, marking
// the copy as "forwarded". Text and photo are carried over; reactions and the
// reply link are not. The new message is attributed to senderID (the forwarder).
func (db *appdbimpl) ForwardMessage(sourceMessageID, targetConversationID, senderID string) (Message, error) {
	src, err := db.GetMessage(sourceMessageID)
	if err != nil {
		return Message{}, err
	}

	id, err := uuid.NewV4()
	if err != nil {
		return Message{}, err
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)

	if _, err := db.c.Exec(
		`INSERT INTO messages (id, conversation_id, sender_id, text, photo, timestamp, forwarded)
		 VALUES (?, ?, ?, ?, ?, ?, 1)`,
		id.String(), targetConversationID, senderID, src.Text, src.Photo, timestamp,
	); err != nil {
		return Message{}, fmt.Errorf("error forwarding message: %w", err)
	}

	preview := src.Text
	if preview == "" && src.Photo != "" {
		preview = "[Photo]"
	}
	if _, err := db.c.Exec(
		`UPDATE conversations SET last_message = ?, last_time = ? WHERE id = ?`,
		preview, timestamp, targetConversationID,
	); err != nil {
		return Message{}, err
	}

	return Message{
		ID:             id.String(),
		ConversationID: targetConversationID,
		SenderID:       senderID,
		Text:           src.Text,
		Photo:          src.Photo,
		Timestamp:      timestamp,
		Status:         "sent",
		Forwarded:      true,
		Comments:       []Comment{},
	}, nil
}

// AddReaction sets this user's single emoji reaction on a message. A user may
// have at most ONE reaction per message, so any previous one is replaced first.
// The returned Comment carries the author's display name.
func (db *appdbimpl) AddReaction(messageID, userID, emoticon string) (Comment, error) {
	if _, err := db.c.Exec(`DELETE FROM comments WHERE message_id = ? AND user_id = ?`, messageID, userID); err != nil {
		return Comment{}, err
	}

	id, err := uuid.NewV4()
	if err != nil {
		return Comment{}, err
	}
	if _, err := db.c.Exec(
		`INSERT INTO comments (id, message_id, user_id, emoticon) VALUES (?, ?, ?, ?)`,
		id.String(), messageID, userID, emoticon,
	); err != nil {
		return Comment{}, fmt.Errorf("error adding reaction: %w", err)
	}

	var name string
	if err := db.c.QueryRow(`SELECT COALESCE(name, ?) FROM users WHERE id = ?`, userID, userID).Scan(&name); err != nil {
		name = userID
	}
	return Comment{ID: id.String(), MessageID: messageID, UserID: userID, UserName: name, Emoticon: emoticon}, nil
}

// RemoveReaction deletes this user's reaction on a message (if any).
func (db *appdbimpl) RemoveReaction(messageID, userID string) error {
	_, err := db.c.Exec(`DELETE FROM comments WHERE message_id = ? AND user_id = ?`, messageID, userID)
	return err
}

// MarkConversationRead inserts a (message_id, user_id) row for every message
// in the conversation, so the sender sees the "read" status.
func (db *appdbimpl) MarkConversationRead(conversationID, userID string) error {
	_, err := db.c.Exec(`
		INSERT OR IGNORE INTO message_reads (message_id, user_id)
		SELECT id, ? FROM messages WHERE conversation_id = ?`,
		userID, conversationID)
	return err
}
