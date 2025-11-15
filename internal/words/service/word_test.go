package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/gamma-omg/lexi-go.git/internal/words/model"
	"github.com/gamma-omg/lexi-go.git/internal/words/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockWordStore struct {
	insertWord func(ctx context.Context, r store.WordInsertRequest) (string, error)
	deleteWord func(ctx context.Context, r store.WordDeleteRequest) error
}

func (m *mockWordStore) InsertWord(ctx context.Context, r store.WordInsertRequest) (string, error) {
	return m.insertWord(ctx, r)
}

func (m *mockWordStore) DeleteWord(ctx context.Context, r store.WordDeleteRequest) error {
	return m.deleteWord(ctx, r)
}

func TestAddWord(t *testing.T) {
	var insertedWords []store.WordInsertRequest
	mockStore := &mockWordStore{
		insertWord: func(ctx context.Context, r store.WordInsertRequest) (string, error) {
			insertedWords = append(insertedWords, r)
			return uuid.NewString(), nil
		},
	}

	service := &WordService{store: mockStore}
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
	mockStore := &mockWordStore{
		insertWord: func(ctx context.Context, r store.WordInsertRequest) (string, error) {
			return "", store.ErrExists
		},
	}

	service := &WordService{store: mockStore}
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
	mockStore := &mockWordStore{
		deleteWord: func(ctx context.Context, r store.WordDeleteRequest) error {
			deletedWords = append(deletedWords, r)
			return nil
		},
	}

	service := &WordService{store: mockStore}
	wordID := uuid.NewString()

	err := service.DeleteWord(context.Background(), wordID)
	require.NoError(t, err)

	require.Len(t, deletedWords, 1)
	require.Contains(t, deletedWords, store.WordDeleteRequest{
		ID: wordID,
	})
}

func TestDeleteWord_NotFound(t *testing.T) {
	mockStore := &mockWordStore{
		deleteWord: func(ctx context.Context, r store.WordDeleteRequest) error {
			return store.ErrNotFound
		},
	}

	service := &WordService{store: mockStore}
	wordID := uuid.NewString()

	err := service.DeleteWord(context.Background(), wordID)
	require.Error(t, err)

	se, ok := err.(*ServiceError)
	require.True(t, ok)
	require.Equal(t, http.StatusNotFound, se.StatusCode)
	require.Equal(t, wordID, se.Env["word_id"])
}
