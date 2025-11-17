package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/gamma-omg/lexi-go/internal/services/words/internal/store"
	"github.com/stretchr/testify/require"
)

type mockUserStore struct {
	CreateUserPickFunc func(ctx context.Context, r store.UserPickCreateRequest) error
	DeleteUserPickFunc func(ctx context.Context, r store.UserPickDeleteRequest) error
	AddTagFunc         func(ctx context.Context, r store.UserPickAddTagRequest) error
	RemoveTagFunc      func(ctx context.Context, r store.UserPickRemoveTagRequest) error
	GetOrCreateTagFunc func(ctx context.Context, tag string) (int64, error)
}

func (m *mockUserStore) CreateUserPick(ctx context.Context, r store.UserPickCreateRequest) error {
	return m.CreateUserPickFunc(ctx, r)
}

func (m *mockUserStore) DeleteUserPick(ctx context.Context, r store.UserPickDeleteRequest) error {
	return m.DeleteUserPickFunc(ctx, r)
}

func (m *mockUserStore) AddTag(ctx context.Context, r store.UserPickAddTagRequest) error {
	return m.AddTagFunc(ctx, r)
}

func (m *mockUserStore) RemoveTag(ctx context.Context, r store.UserPickRemoveTagRequest) error {
	return m.RemoveTagFunc(ctx, r)
}

func (m *mockUserStore) GetOrCreateTag(ctx context.Context, tag string) (int64, error) {
	return m.GetOrCreateTagFunc(ctx, tag)
}

func TestPickWord(t *testing.T) {
	var createdPicks []store.UserPickCreateRequest
	userStore := &mockUserStore{
		CreateUserPickFunc: func(ctx context.Context, r store.UserPickCreateRequest) error {
			createdPicks = append(createdPicks, r)
			return nil
		},
	}

	userService := &UserService{
		UserID: 123,
		store:  userStore,
	}

	err := userService.PickWord(context.Background(), UserPickWordRequest{
		WordID: 456,
		DefID:  789,
	})
	require.NoError(t, err)

	require.Len(t, createdPicks, 1)
	require.Contains(t, createdPicks, store.UserPickCreateRequest{
		UserID: 123,
		DefID:  789,
	})
}

func TestPickWord_Exists(t *testing.T) {
	userStore := &mockUserStore{
		CreateUserPickFunc: func(ctx context.Context, r store.UserPickCreateRequest) error {
			return store.ErrExists
		},
	}

	userService := &UserService{
		UserID: 123,
		store:  userStore,
	}

	err := userService.PickWord(context.Background(), UserPickWordRequest{
		WordID: 456,
		DefID:  789,
	})
	require.Error(t, err)

	se, ok := err.(*ServiceError)
	require.True(t, ok)
	require.Equal(t, http.StatusConflict, se.StatusCode)
	require.Equal(t, "123", se.Env["user_id"])
	require.Equal(t, "456", se.Env["word_id"])
	require.Equal(t, "789", se.Env["def_id"])
}

func TestUnpickWord(t *testing.T) {
	var deletedPicks []store.UserPickDeleteRequest
	userStore := &mockUserStore{
		DeleteUserPickFunc: func(ctx context.Context, r store.UserPickDeleteRequest) error {
			deletedPicks = append(deletedPicks, r)
			return nil
		},
	}

	userService := &UserService{
		UserID: 123,
		store:  userStore,
	}

	err := userService.UnpickWord(context.Background(), 456)
	require.NoError(t, err)

	require.Len(t, deletedPicks, 1)
	require.Contains(t, deletedPicks, store.UserPickDeleteRequest{PickID: 456})
}

func TestUnpickWord_NotFound(t *testing.T) {
	userStore := &mockUserStore{
		DeleteUserPickFunc: func(ctx context.Context, r store.UserPickDeleteRequest) error {
			return store.ErrNotFound
		},
	}

	userService := &UserService{
		UserID: 123,
		store:  userStore,
	}

	err := userService.UnpickWord(context.Background(), 456)
	require.Error(t, err)

	se, ok := err.(*ServiceError)
	require.True(t, ok)
	require.Equal(t, http.StatusNotFound, se.StatusCode)
	require.Equal(t, "456", se.Env["pick_id"])
}

func TestAddTag(t *testing.T) {
	var addedTags []store.UserPickAddTagRequest
	mockStore := &mockUserStore{
		GetOrCreateTagFunc: func(ctx context.Context, tag string) (int64, error) {
			return 789, nil
		},
		AddTagFunc: func(ctx context.Context, r store.UserPickAddTagRequest) error {
			addedTags = append(addedTags, r)
			return nil
		},
	}

	service := &UserService{
		UserID: 123,
		store:  mockStore,
	}
	req := UserPickAddTagRequest{
		PickID: 456,
		Tag:    "important",
	}

	err := service.AddTag(context.Background(), req)
	require.NoError(t, err)

	require.Len(t, addedTags, 1)
	require.Contains(t, addedTags, store.UserPickAddTagRequest{
		PickID: 456,
		TagID:  789,
	})
}

func TestAddTag_PickNotFound(t *testing.T) {
	mockStore := &mockUserStore{
		GetOrCreateTagFunc: func(ctx context.Context, tag string) (int64, error) {
			return 789, nil
		},
		AddTagFunc: func(ctx context.Context, r store.UserPickAddTagRequest) error {
			return store.ErrNotFound
		},
	}

	service := &UserService{
		UserID: 123,
		store:  mockStore,
	}
	req := UserPickAddTagRequest{
		PickID: 456,
		Tag:    "important",
	}

	err := service.AddTag(context.Background(), req)
	require.Error(t, err)

	se, ok := err.(*ServiceError)
	require.True(t, ok)
	require.Equal(t, http.StatusNotFound, se.StatusCode)
	require.Equal(t, "456", se.Env["pick_id"])
}

func TestRemoveTag(t *testing.T) {
	var removedTags []store.UserPickRemoveTagRequest
	mockStore := &mockUserStore{
		RemoveTagFunc: func(ctx context.Context, r store.UserPickRemoveTagRequest) error {
			removedTags = append(removedTags, r)
			return nil
		},
	}

	service := &UserService{
		UserID: 123,
		store:  mockStore,
	}
	req := UserPickRemoveTagRequest{
		PickID: 456,
		TagID:  789,
	}

	err := service.RemoveTag(context.Background(), req)
	require.NoError(t, err)

	require.Len(t, removedTags, 1)
	require.Contains(t, removedTags, store.UserPickRemoveTagRequest{
		PickID: 456,
		TagID:  789,
	})
}

func TestRemoveTag_PickNotFound(t *testing.T) {
	mockStore := &mockUserStore{
		RemoveTagFunc: func(ctx context.Context, r store.UserPickRemoveTagRequest) error {
			return store.ErrNotFound
		},
	}

	service := &UserService{
		UserID: 123,
		store:  mockStore,
	}
	req := UserPickRemoveTagRequest{
		PickID: 456,
		TagID:  789,
	}

	err := service.RemoveTag(context.Background(), req)
	require.Error(t, err)

	se, ok := err.(*ServiceError)
	require.True(t, ok)
	require.Equal(t, http.StatusNotFound, se.StatusCode)
	require.Equal(t, "456", se.Env["pick_id"])
	require.Equal(t, "789", se.Env["tag_id"])
}
