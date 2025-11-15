package service

import (
	"context"
	"testing"

	"github.com/gamma-omg/lexi-go.git/internal/words/model"
	"github.com/gamma-omg/lexi-go.git/internal/words/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockWordStore struct {
	insertedWords []store.WordInsertRequest
	deletedWords  []store.WordDeleteRequest
}

func (m *mockWordStore) InsertWord(ctx context.Context, r store.WordInsertRequest) (string, error) {
	m.insertedWords = append(m.insertedWords, r)
	return uuid.New().String(), nil
}

func (m *mockWordStore) DeleteWord(ctx context.Context, r store.WordDeleteRequest) error {
	m.deletedWords = append(m.deletedWords, r)
	return nil
}

func Test_AddWord(t *testing.T) {
	mockStore := &mockWordStore{}
	service := &WordService{store: mockStore}

	req := WordAddRequest{
		Lemma: "example",
		Lang:  "en",
		Class: model.Noun,
	}

	_, err := service.AddWord(context.Background(), req)
	require.NoError(t, err)

	require.Len(t, mockStore.insertedWords, 1)
	require.Contains(t, mockStore.insertedWords, store.WordInsertRequest{
		Lemma: req.Lemma,
		Lang:  req.Lang,
		Class: req.Class,
	})
}

func Test_DeleteWord(t *testing.T) {
	mockStore := &mockWordStore{}
	service := &WordService{store: mockStore}

	wordID := uuid.New().String()

	err := service.DeleteWord(context.Background(), wordID)
	require.NoError(t, err)

	require.Len(t, mockStore.deletedWords, 1)
	require.Contains(t, mockStore.deletedWords, store.WordDeleteRequest{
		ID: wordID,
	})
}
