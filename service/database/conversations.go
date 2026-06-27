package database

import (
	"fmt"

	"github.com/gofrs/uuid"
)

// GetConversations returns every conversation the user is a member of, sorted
// by most recent activity. For 1:1 chats it substitutes the other participant's
// name/photo into `name`/`photo` so the UI does not have to figure it out.
func (db *appdbimpl) GetConversations(userID string) ([]Conversation, error) {
	rows, err := db.c.Query(`
		SELECT c.id,
		  CASE WHEN c.is_group = 1 THEN c.name
		       ELSE COALESCE((
		         SELECT u.name FROM users u
		         JOIN members m2 ON u.id = m2.user_id
		         WHERE m2.conversation_id = c.id AND u.id != m.user_id
		         LIMIT 1
		       ), c.name)
		  END AS display_name,
		  CASE WHEN c.is_group = 1 THEN c.photo
		       ELSE COALESCE((
		         SELECT u.photo FROM users u
		         JOIN members m2 ON u.id = m2.user_id
		         WHERE m2.conversation_id = c.id AND u.id != m.user_id
		         LIMIT 1
		       ), '')
		  END AS display_photo,
		  c.is_group, c.last_message, c.last_time
		FROM conversations c
		JOIN members m ON c.id = m.conversation_id
		WHERE m.user_id = ?
		ORDER BY c.last_time DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting conversations: %w", err)
	}
	defer rows.Close()

	conversations := make([]Conversation, 0)
	for rows.Next() {
		var c Conversation
		var isGroup int
		if err := rows.Scan(&c.ID, &c.Name, &c.Photo, &isGroup, &c.LastMessage, &c.LastTime); err != nil {
			return nil, err
		}
		c.IsGroup = isGroup == 1
		conversations = append(conversations, c)
	}
	return conversations, rows.Err()
}

// GetConversation returns one conversation row (without smart name substitution).
func (db *appdbimpl) GetConversation(conversationID string) (Conversation, error) {
	var c Conversation
	var isGroup int
	err := db.c.QueryRow(`
		SELECT id, name, photo, is_group, last_message, last_time
		FROM conversations WHERE id = ?`, conversationID).
		Scan(&c.ID, &c.Name, &c.Photo, &isGroup, &c.LastMessage, &c.LastTime)
	if err != nil {
		return Conversation{}, fmt.Errorf("conversation not found: %w", err)
	}
	c.IsGroup = isGroup == 1
	return c, nil
}

// CreateConversation inserts a new conversation and links each member to it.
func (db *appdbimpl) CreateConversation(name string, isGroup bool, members []string) (Conversation, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return Conversation{}, err
	}

	isGroupInt := 0
	if isGroup {
		isGroupInt = 1
	}

	if _, err := db.c.Exec(
		`INSERT INTO conversations (id, name, is_group) VALUES (?, ?, ?)`,
		id.String(), name, isGroupInt,
	); err != nil {
		return Conversation{}, fmt.Errorf("error creating conversation: %w", err)
	}

	for _, memberID := range members {
		if _, err := db.c.Exec(
			`INSERT INTO members (conversation_id, user_id) VALUES (?, ?)`,
			id.String(), memberID,
		); err != nil {
			return Conversation{}, err
		}
	}

	return Conversation{ID: id.String(), Name: name, IsGroup: isGroup}, nil
}

// SetConversationName renames a conversation (group branding).
func (db *appdbimpl) SetConversationName(conversationID, name string) error {
	_, err := db.c.Exec(`UPDATE conversations SET name = ? WHERE id = ?`, name, conversationID)
	return err
}

// SetConversationPhoto updates a conversation's picture (group branding).
func (db *appdbimpl) SetConversationPhoto(conversationID, photo string) error {
	_, err := db.c.Exec(`UPDATE conversations SET photo = ? WHERE id = ?`, photo, conversationID)
	return err
}

// AddMember adds a user to a conversation (idempotent).
func (db *appdbimpl) AddMember(conversationID, userID string) error {
	_, err := db.c.Exec(
		`INSERT OR IGNORE INTO members (conversation_id, user_id) VALUES (?, ?)`,
		conversationID, userID,
	)
	return err
}

// RemoveMember removes a user from a conversation (used by leaveGroup).
func (db *appdbimpl) RemoveMember(conversationID, userID string) error {
	_, err := db.c.Exec(
		`DELETE FROM members WHERE conversation_id = ? AND user_id = ?`,
		conversationID, userID,
	)
	return err
}

// CountMembers returns how many users belong to a conversation.
func (db *appdbimpl) CountMembers(conversationID string) (int, error) {
	var n int
	err := db.c.QueryRow(`SELECT COUNT(*) FROM members WHERE conversation_id = ?`, conversationID).Scan(&n)
	return n, err
}

// GetMembers returns the users belonging to a conversation.
func (db *appdbimpl) GetMembers(conversationID string) ([]User, error) {
	rows, err := db.c.Query(`
		SELECT u.id, u.name, u.photo
		FROM users u JOIN members m ON u.id = m.user_id
		WHERE m.conversation_id = ?
		ORDER BY u.name`, conversationID)
	if err != nil {
		return nil, fmt.Errorf("error getting members: %w", err)
	}
	defer rows.Close()

	members := make([]User, 0)
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Photo); err != nil {
			return nil, err
		}
		members = append(members, u)
	}
	return members, rows.Err()
}

// IsMember reports whether user is part of this conversation.
func (db *appdbimpl) IsMember(conversationID, userID string) (bool, error) {
	var count int
	err := db.c.QueryRow(
		`SELECT COUNT(*) FROM members WHERE conversation_id = ? AND user_id = ?`,
		conversationID, userID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// FindDirectConversation returns the existing 1:1 conversation between two users, if any.
// Used to avoid creating duplicate DMs.
func (db *appdbimpl) FindDirectConversation(userA, userB string) (Conversation, error) {
	var c Conversation
	var isGroup int
	err := db.c.QueryRow(`
		SELECT c.id, c.name, c.photo, c.is_group, c.last_message, c.last_time
		FROM conversations c
		JOIN members m1 ON c.id = m1.conversation_id AND m1.user_id = ?
		JOIN members m2 ON c.id = m2.conversation_id AND m2.user_id = ?
		WHERE c.is_group = 0
		  AND (SELECT COUNT(*) FROM members WHERE conversation_id = c.id) = 2
		LIMIT 1`, userA, userB,
	).Scan(&c.ID, &c.Name, &c.Photo, &isGroup, &c.LastMessage, &c.LastTime)
	if err != nil {
		return Conversation{}, fmt.Errorf("not found: %w", err)
	}
	c.IsGroup = isGroup == 1
	return c, nil
}
