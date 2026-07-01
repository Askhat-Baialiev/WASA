package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type User struct {
	ID    string `json:"identifier"`
	Name  string `json:"name"`
	Photo string `json:"photo"`
}

type Conversation struct {
	ID          string `json:"conversationId"`
	Name        string `json:"name"`
	Photo       string `json:"photo"`
	IsGroup     bool   `json:"isGroup"`
	LastMessage string `json:"lastMessage"`
	LastTime    string `json:"lastMessageTime"`
}

type Message struct {
	ID             string    `json:"messageId"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	SenderName     string    `json:"sender"`
	Text           string    `json:"text"`
	Photo          string    `json:"photo,omitempty"`
	Timestamp      string    `json:"timestamp"`
	Status         string    `json:"status"`
	ReplyToID      string    `json:"replyToId,omitempty"`
	Forwarded      bool      `json:"forwarded"`
	Comments       []Comment `json:"comments"`
}

type Comment struct {
	ID        string `json:"commentId"`
	MessageID string `json:"messageId"`
	UserID    string `json:"userId"`
	UserName  string `json:"userName"`
	Emoticon  string `json:"emoticon"`
}

type AppDatabase interface {
	GetOrCreateUser(name string) (User, error)
	GetUser(userID string) (User, error)
	GetUserByName(name string) (User, error)
	SetUserName(userID, name string) error
	SetUserPhoto(userID, photo string) error
	SearchUsers(query string) ([]User, error)

	GetConversations(userID string) ([]Conversation, error)
	GetConversation(conversationID string) (Conversation, error)
	CreateConversation(name string, isGroup bool, members []string) (Conversation, error)
	IsMember(conversationID, userID string) (bool, error)
	FindDirectConversation(userA, userB string) (Conversation, error)
	MarkConversationRead(conversationID, userID string) error
	SetConversationName(conversationID, name string) error
	SetConversationPhoto(conversationID, photo string) error
	AddMember(conversationID, userID string) error
	RemoveMember(conversationID, userID string) error
	CountMembers(conversationID string) (int, error)
	GetMembers(conversationID string) ([]User, error)

	SendMessage(conversationID, senderID, text, photo, replyToID string) (Message, error)
	GetMessages(conversationID string) ([]Message, error)
	GetMessage(messageID string) (Message, error)
	DeleteMessage(messageID string) error
	ForwardMessage(sourceMessageID, targetConversationID, senderID string) (Message, error)

	AddReaction(messageID, userID, emoticon string) (Comment, error)
	RemoveReaction(messageID, userID string) error

	Ping() error
}

type appdbimpl struct {
	c *sql.DB
}

func New(db *sql.DB) (AppDatabase, error) {
	if db == nil {
		return nil, errors.New("database is required when building AppDatabase")
	}

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		photo TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS conversations (
		id TEXT PRIMARY KEY,
		name TEXT DEFAULT '',
		photo TEXT DEFAULT '',
		is_group INTEGER DEFAULT 0,
		last_message TEXT DEFAULT '',
		last_time TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS members (
		conversation_id TEXT,
		user_id TEXT,
		PRIMARY KEY (conversation_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		conversation_id TEXT,
		sender_id TEXT,
		text TEXT DEFAULT '',
		photo TEXT DEFAULT '',
		timestamp TEXT,
		status TEXT DEFAULT 'sent',
		reply_to_id TEXT DEFAULT '',
		forwarded INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS comments (
		id TEXT PRIMARY KEY,
		message_id TEXT,
		user_id TEXT,
		emoticon TEXT,
		UNIQUE(message_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS message_reads (
		message_id TEXT,
		user_id TEXT,
		PRIMARY KEY (message_id, user_id)
	);
	`
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("error creating database structure: %w", err)
	}

	// Migration for databases created before the "forwarded" column existed.
	// Ignore the "duplicate column" error when it is already present.
	if _, err := db.Exec(`ALTER TABLE messages ADD COLUMN forwarded INTEGER DEFAULT 0`); err != nil &&
		!strings.Contains(err.Error(), "duplicate column") {
		return nil, fmt.Errorf("error migrating messages table: %w", err)
	}

	return &appdbimpl{c: db}, nil
}

func (db *appdbimpl) Ping() error {
	return db.c.Ping()
}
