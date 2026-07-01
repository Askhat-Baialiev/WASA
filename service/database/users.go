package database

import (
	"fmt"

	"github.com/gofrs/uuid"
)

func (db *appdbimpl) GetOrCreateUser(name string) (User, error) {
	var u User
	err := db.c.QueryRow(`SELECT id, name, photo FROM users WHERE name = ?`, name).
		Scan(&u.ID, &u.Name, &u.Photo)
	if err == nil {
		return u, nil
	}

	id, err := uuid.NewV4()
	if err != nil {
		return User{}, fmt.Errorf("uuid: %w", err)
	}
	u = User{ID: id.String(), Name: name, Photo: ""}
	if _, err := db.c.Exec(`INSERT INTO users (id, name, photo) VALUES (?, ?, ?)`,
		u.ID, u.Name, u.Photo); err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (db *appdbimpl) GetUser(userID string) (User, error) {
	var u User
	err := db.c.QueryRow(`SELECT id, name, photo FROM users WHERE id = ?`, userID).
		Scan(&u.ID, &u.Name, &u.Photo)
	if err != nil {
		return User{}, fmt.Errorf("user not found: %w", err)
	}
	return u, nil
}

// GetUserByName looks up a user by their (unique) display name.
func (db *appdbimpl) GetUserByName(name string) (User, error) {
	var u User
	err := db.c.QueryRow(`SELECT id, name, photo FROM users WHERE name = ?`, name).
		Scan(&u.ID, &u.Name, &u.Photo)
	if err != nil {
		return User{}, fmt.Errorf("user not found: %w", err)
	}
	return u, nil
}

func (db *appdbimpl) SetUserName(userID, name string) error {
	_, err := db.c.Exec(`UPDATE users SET name = ? WHERE id = ?`, name, userID)
	return err
}

func (db *appdbimpl) SetUserPhoto(userID, photo string) error {
	_, err := db.c.Exec(`UPDATE users SET photo = ? WHERE id = ?`, photo, userID)
	return err
}

func (db *appdbimpl) SearchUsers(query string) ([]User, error) {
	rows, err := db.c.Query(`SELECT id, name, photo FROM users WHERE name LIKE ? ORDER BY name`, "%"+query+"%")
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Photo); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
