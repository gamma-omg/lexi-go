package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gamma-omg/lexi-go.git/internal/words/store"
)

var (
	ErrUserPickNotFound = errors.New("user pick not found")
)

type userStore interface {
	CreateUserPick(ctx context.Context, r store.UserPickCreateRequest) error
	DeleteUserPick(ctx context.Context, r store.UserPickDeleteRequest) error
	GetOrCreateTag(ctx context.Context, tag string) (string, error)
	AddTag(ctx context.Context, r store.UserPickAddTagRequest) error
	RemoveTag(ctx context.Context, r store.UserPickRemoveTagRequest) error
}

// UserService provides access to user-specific word operations such as picking words and managing tags.
type UserService struct {
	UserID string
	store  userStore
}

type UserPickWordRequest struct {
	WordID string
	DefID  string
}

type UserUnpickWordRequest struct {
	WordID string
	DefID  string
}

type UserPickAddTagRequest struct {
	PickID string
	Tag    string
}

type UserPickRemoveTagRequest struct {
	PickID string
	TagID  string
}

// PickWord allows a user to pick a word definition for learning. If the pick already exists, it returns a ServiceError
// with status code 409.
func (s *UserService) PickWord(ctx context.Context, r UserPickWordRequest) error {
	if err := s.store.CreateUserPick(ctx, store.UserPickCreateRequest{
		UserID: s.UserID,
		WordID: r.WordID,
		DefID:  r.DefID,
	}); err != nil {
		if errors.Is(err, store.ErrExists) {
			se := NewServiceError(err, http.StatusConflict, "user pick already exists")
			se.Env["user_id"] = s.UserID
			se.Env["word_id"] = r.WordID
			se.Env["def_id"] = r.DefID
			return se
		}
	}

	return nil
}

// UnpickWord allows a user to unpick a previously picked word definition. If the pick does not exist, it returns a ServiceError
// with status code 404.
func (s *UserService) UnpickWord(ctx context.Context, r UserUnpickWordRequest) error {
	if err := s.store.DeleteUserPick(ctx, store.UserPickDeleteRequest{
		UserID: s.UserID,
		WordID: r.WordID,
		DefID:  r.DefID,
	}); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			se := NewServiceError(err, http.StatusNotFound, "user pick was not found")
			se.Env["user_id"] = s.UserID
			se.Env["word_id"] = r.WordID
			se.Env["def_id"] = r.DefID
			return se
		}

		return fmt.Errorf("delete user pick: %w", err)
	}

	return nil
}

// AddTag adds a tag to a user's picked word. If the pick does not exist, it returns a ServiceError with status code 404.
func (s *UserService) AddTag(ctx context.Context, r UserPickAddTagRequest) error {
	tagID, err := s.store.GetOrCreateTag(ctx, r.Tag)
	if err != nil {
		return fmt.Errorf("get or create tag: %w", err)
	}

	if err := s.store.AddTag(ctx, store.UserPickAddTagRequest{
		PickID: r.PickID,
		TagID:  tagID,
	}); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			se := NewServiceError(err, http.StatusNotFound, "user pick was not found")
			se.Env["pick_id"] = r.PickID
			return se
		}

		return fmt.Errorf("add tag to pick: %w", err)
	}

	return nil
}

// RemoveTag removes a tag from a user's picked word. If the pick does not exist, it returns a ServiceError with status code 404.
func (s *UserService) RemoveTag(ctx context.Context, r UserPickRemoveTagRequest) error {
	if err := s.store.RemoveTag(ctx, store.UserPickRemoveTagRequest{
		PickID: r.PickID,
		TagID:  r.TagID,
	}); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			se := NewServiceError(err, http.StatusNotFound, "user pick was not found")
			se.Env["pick_id"] = r.PickID
			se.Env["tag_id"] = r.TagID
			return se
		}

		return fmt.Errorf("remove tag from pick: %w", err)
	}

	return nil
}
