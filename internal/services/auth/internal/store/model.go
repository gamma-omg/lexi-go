package store

import "time"

type Model struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

type User struct {
	Model
	ID  int64
	UID string
}

type Identity struct {
	Model
	User     User
	ID       string
	Provider string
	Email    string
	Name     string
	Picture  string
}
