package store

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("not found")
)

type Store interface {
	GetIdentity(ctx context.Context, r GetIdentityRequest) (Identity, error)
	GetUserIdentity(ctx context.Context, r GetUserIdentityRequest) (Identity, error)
	CreateUser(ctx context.Context) (int64, error)
	CreateUserIdentity(ctx context.Context, r CreateUserIdentityRequest) (string, error)
	WithTx(ctx context.Context, fn func(tx Store) error) error
}

type GetIdentityRequest struct {
	ID       string
	Provider string
}

type GetUserIdentityRequest struct {
	UID      string
	Provider string
}

type CreateUserIdentityRequest struct {
	UserID   int64
	ID       string
	Provider string
	Email    string
	Name     string
	Picture  string
}
