package store

import "time"

// Model represents the base database model
type Model struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

// User represents a user in the system
type User struct {
	Model
	ID  int64
	UID string
}

// Identity represents a user's identity from an OAuth provider
type Identity struct {
	Model
	User     User
	ID       string
	Provider string
	Email    string
	Name     string
	Picture  string
}
