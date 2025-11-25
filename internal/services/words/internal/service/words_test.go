package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/gamma-omg/lexi-go/internal/services/words/internal/model"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/store"
	"github.com/stretchr/testify/require"
)

type mockStore struct {
	insertWord         func(ctx context.Context, r store.WordInsertRequest) (int64, error)
	deleteWord         func(ctx context.Context, r store.WordDeleteRequest) error
	CreateUserPickFunc func(ctx context.Context, r store.UserPickCreateRequest) error
	DeleteUserPickFunc func(ctx context.Context, r store.UserPickDeleteRequest) error
	AddTagFunc         func(ctx context.Context, r store.UserPickAddTagRequest) error
	RemoveTagFunc      func(ctx context.Context, r store.UserPickRemoveTagRequest) error
	GetOrCreateTagFunc func(ctx context.Context, tag string) (int64, error)
}

func (m *mockStore) InsertWord(ctx context.Context, r store.WordInsertRequest) (int64, error) {
	return m.insertWord(ctx, r)
}

func (m *mockStore) DeleteWord(ctx context.Context, r store.WordDeleteRequest) error {
	return m.deleteWord(ctx, r)
}

func (m *mockStore) CreateUserPick(ctx context.Context, r store.UserPickCreateRequest) error {
	return m.CreateUserPickFunc(ctx, r)
}

func (m *mockStore) DeleteUserPick(ctx context.Context, r store.UserPickDeleteRequest) error {
	return m.DeleteUserPickFunc(ctx, r)
}

func (m *mockStore) AddTag(ctx context.Context, r store.UserPickAddTagRequest) error {
	return m.AddTagFunc(ctx, r)
}

func (m *mockStore) RemoveTag(ctx context.Context, r store.UserPickRemoveTagRequest) error {
	return m.RemoveTagFunc(ctx, r)
}

func (m *mockStore) GetOrCreateTag(ctx context.Context, tag string) (int64, error) {
	return m.GetOrCreateTagFunc(ctx, tag)
}

func TestAddWord(t *testing.T) {
	var insertedWords []store.WordInsertRequest
	mockStore := &mockStore{
		insertWord: func(ctx context.Context, r store.WordInsertRequest) (int64, error) {
			insertedWords = append(insertedWords, r)
			return 1, nil
		},
	}

	service := &WordsService{store: mockStore}
	req := WordAddRequest{
		Lemma: "example",
		Lang:  "en",
		Class: model.Noun,
	}

	_, err := service.AddWord(context.Background(), req)
	require.NoError(t, err)

	require.Len(t, insertedWords, 1)
	require.Contains(t, insertedWords, store.WordInsertRequest{
		Lemma: req.Lemma,
		Lang:  req.Lang,
		Class: req.Class,
	})
}

func TestAddWord_Exists(t *testing.T) {
	mockStore := &mockStore{
		insertWord: func(ctx context.Context, r store.WordInsertRequest) (int64, error) {
			return 0, store.ErrExists
		},
	}

	service := &WordsService{store: mockStore}
	req := WordAddRequest{
		Lemma: "example",
		Lang:  "en",
		Class: model.Noun,
	}

	_, err := service.AddWord(context.Background(), req)
	require.Error(t, err)

	se, ok := err.(*ServiceError)
	require.True(t, ok)
	require.Equal(t, http.StatusConflict, se.StatusCode)
	require.Equal(t, req.Lemma, se.Env["lemma"])
	require.Equal(t, string(req.Lang), se.Env["lang"])
	require.Equal(t, string(req.Class), se.Env["class"])
}

func TestDeleteWord(t *testing.T) {
	var deletedWords []store.WordDeleteRequest
	mockStore := &mockStore{
		deleteWord: func(ctx context.Context, r store.WordDeleteRequest) error {
			deletedWords = append(deletedWords, r)
			return nil
		},
	}

	service := &WordsService{store: mockStore}
	wordID := int64(12345)

	err := service.DeleteWord(context.Background(), wordID)
	require.NoError(t, err)

	require.Len(t, deletedWords, 1)
	require.Contains(t, deletedWords, store.WordDeleteRequest{
		ID: wordID,
	})
}

func TestDeleteWord_NotFound(t *testing.T) {
	mockStore := &mockStore{
		deleteWord: func(ctx context.Context, r store.WordDeleteRequest) error {
			return store.ErrNotFound
		},
	}

	service := &WordsService{store: mockStore}
	wordID := int64(12345)

	err := service.DeleteWord(context.Background(), wordID)
	require.Error(t, err)

	se, ok := err.(*ServiceError)
	require.True(t, ok)
	require.Equal(t, http.StatusNotFound, se.StatusCode)
	require.Equal(t, "12345", se.Env["word_id"])
}

func TestPickWord(t *testing.T) {
	var createdPicks []store.UserPickCreateRequest
	userStore := &mockStore{
		CreateUserPickFunc: func(ctx context.Context, r store.UserPickCreateRequest) error {
			createdPicks = append(createdPicks, r)
			return nil
		},
	}

	srv := &WordsService{store: userStore}

	err := srv.PickWord(context.Background(), UserPickWordRequest{
		UserID: "user-123",
		WordID: 456,
		DefID:  789,
	})
	require.NoError(t, err)

	require.Len(t, createdPicks, 1)
	require.Contains(t, createdPicks, store.UserPickCreateRequest{
		UserID: "user-123",
		DefID:  789,
	})
}

func TestPickWord_Exists(t *testing.T) {
	userStore := &mockStore{
		CreateUserPickFunc: func(ctx context.Context, r store.UserPickCreateRequest) error {
			return store.ErrExists
		},
	}

	userService := &WordsService{store: userStore}

	err := userService.PickWord(context.Background(), UserPickWordRequest{
		UserID: "user-123",
		WordID: 456,
		DefID:  789,
	})
	require.Error(t, err)

	se, ok := err.(*ServiceError)
	require.True(t, ok)
	require.Equal(t, http.StatusConflict, se.StatusCode)
	require.Equal(t, "user-123", se.Env["user_id"])
	require.Equal(t, "456", se.Env["word_id"])
	require.Equal(t, "789", se.Env["def_id"])
}

func TestUnpickWord(t *testing.T) {
	var deletedPicks []store.UserPickDeleteRequest
	userStore := &mockStore{
		DeleteUserPickFunc: func(ctx context.Context, r store.UserPickDeleteRequest) error {
			deletedPicks = append(deletedPicks, r)
			return nil
		},
	}

	userService := &WordsService{store: userStore}

	err := userService.UnpickWord(context.Background(), 456)
	require.NoError(t, err)

	require.Len(t, deletedPicks, 1)
	require.Contains(t, deletedPicks, store.UserPickDeleteRequest{PickID: 456})
}

func TestUnpickWord_NotFound(t *testing.T) {
	userStore := &mockStore{
		DeleteUserPickFunc: func(ctx context.Context, r store.UserPickDeleteRequest) error {
			return store.ErrNotFound
		},
	}

	userService := &WordsService{store: userStore}

	err := userService.UnpickWord(context.Background(), 456)
	require.Error(t, err)

	se, ok := err.(*ServiceError)
	require.True(t, ok)
	require.Equal(t, http.StatusNotFound, se.StatusCode)
	require.Equal(t, "456", se.Env["pick_id"])
}

func TestAddTag(t *testing.T) {
	var addedTags []store.UserPickAddTagRequest
	mockStore := &mockStore{
		GetOrCreateTagFunc: func(ctx context.Context, tag string) (int64, error) {
			return 789, nil
		},
		AddTagFunc: func(ctx context.Context, r store.UserPickAddTagRequest) error {
			addedTags = append(addedTags, r)
			return nil
		},
	}

	service := &WordsService{store: mockStore}
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
	mockStore := &mockStore{
		GetOrCreateTagFunc: func(ctx context.Context, tag string) (int64, error) {
			return 789, nil
		},
		AddTagFunc: func(ctx context.Context, r store.UserPickAddTagRequest) error {
			return store.ErrNotFound
		},
	}

	service := &WordsService{store: mockStore}
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
	mockStore := &mockStore{
		RemoveTagFunc: func(ctx context.Context, r store.UserPickRemoveTagRequest) error {
			removedTags = append(removedTags, r)
			return nil
		},
	}

	service := &WordsService{store: mockStore}
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
	mockStore := &mockStore{
		RemoveTagFunc: func(ctx context.Context, r store.UserPickRemoveTagRequest) error {
			return store.ErrNotFound
		},
	}

	service := &WordsService{store: mockStore}
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
