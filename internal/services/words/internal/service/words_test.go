package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gamma-omg/lexi-go/internal/services/words/internal/model"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/store"
	"github.com/stretchr/testify/require"
)

type mockStore struct {
	insertWord         func(ctx context.Context, r store.WordInsertRequest) (int64, error)
	deleteWord         func(ctx context.Context, r store.WordDeleteRequest) error
	CreateUserPickFunc func(ctx context.Context, r store.UserPickCreateRequest) (int64, error)
	DeleteUserPickFunc func(ctx context.Context, r store.UserPickDeleteRequest) error
	CreateTagsFunc     func(ctx context.Context, r store.TagsCreateRequest) (model.TagIDMap, error)
	GetTagsFunc        func(ctx context.Context, r store.TagsGetRequest) (model.TagIDMap, error)
	AddTagsFunc        func(ctx context.Context, r store.TagsAddRequest) error
	RemoveTagsFunc     func(ctx context.Context, r store.TagsRemoveRequest) error
}

func (m *mockStore) InsertWord(ctx context.Context, r store.WordInsertRequest) (int64, error) {
	return m.insertWord(ctx, r)
}

func (m *mockStore) DeleteWord(ctx context.Context, r store.WordDeleteRequest) error {
	return m.deleteWord(ctx, r)
}

func (m *mockStore) CreateUserPick(ctx context.Context, r store.UserPickCreateRequest) (int64, error) {
	return m.CreateUserPickFunc(ctx, r)
}

func (m *mockStore) DeleteUserPick(ctx context.Context, r store.UserPickDeleteRequest) error {
	return m.DeleteUserPickFunc(ctx, r)
}

func (m *mockStore) CreateTags(ctx context.Context, r store.TagsCreateRequest) (model.TagIDMap, error) {
	return m.CreateTagsFunc(ctx, r)
}

func (m *mockStore) GetTags(ctx context.Context, r store.TagsGetRequest) (model.TagIDMap, error) {
	return m.GetTagsFunc(ctx, r)
}

func (m *mockStore) AddTags(ctx context.Context, r store.TagsAddRequest) error {
	return m.AddTagsFunc(ctx, r)
}

func (m *mockStore) RemoveTags(ctx context.Context, r store.TagsRemoveRequest) error {
	return m.RemoveTagsFunc(ctx, r)
}

func (m *mockStore) WithinTx(ctx context.Context, fn func(tx store.DataStore) error) error {
	return fn(m)
}

func TestAddWord(t *testing.T) {
	var insertedWords []store.WordInsertRequest
	mockStore := &mockStore{
		insertWord: func(ctx context.Context, r store.WordInsertRequest) (int64, error) {
			insertedWords = append(insertedWords, r)
			return 1, nil
		},
	}

	service := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})
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

	service := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})
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

	service := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})
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

	service := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})
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
	var addedTags []store.TagsAddRequest
	userStore := &mockStore{
		CreateTagsFunc: func(ctx context.Context, r store.TagsCreateRequest) (model.TagIDMap, error) {
			return model.TagIDMap{
				"important": 100,
				"review":    200,
			}, nil
		},
		GetTagsFunc: func(ctx context.Context, r store.TagsGetRequest) (model.TagIDMap, error) {
			return model.TagIDMap{}, nil
		},
		AddTagsFunc: func(ctx context.Context, r store.TagsAddRequest) error {
			addedTags = append(addedTags, r)
			return nil
		},
		CreateUserPickFunc: func(ctx context.Context, r store.UserPickCreateRequest) (int64, error) {
			createdPicks = append(createdPicks, r)
			return int64(len(createdPicks)), nil
		},
	}

	srv := NewWordsService(userStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})

	err := srv.PickWord(context.Background(), UserPickWordRequest{
		UserID: "user-123",
		WordID: 456,
		DefID:  789,
		Tags:   []string{"important", "review"},
	})
	require.NoError(t, err)

	require.Len(t, createdPicks, 1)
	require.Contains(t, createdPicks, store.UserPickCreateRequest{
		UserID: "user-123",
		DefID:  789,
	})

	require.Len(t, addedTags, 1)
	require.Contains(t, addedTags, store.TagsAddRequest{
		PickID: 1,
		TagIDs: []int64{100, 200},
	})
}

func TestPickWord_Exists(t *testing.T) {
	userStore := &mockStore{
		CreateUserPickFunc: func(ctx context.Context, r store.UserPickCreateRequest) (int64, error) {
			return 0, store.ErrExists
		},
	}

	srv := NewWordsService(userStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})

	err := srv.PickWord(context.Background(), UserPickWordRequest{
		UserID: "user-123",
		WordID: 456,
		DefID:  789,
	})
	require.Error(t, err)

	var se *ServiceError
	require.True(t, errors.As(err, &se))
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

	srv := NewWordsService(userStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})

	err := srv.UnpickWord(context.Background(), 456)
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

	srv := NewWordsService(userStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})

	err := srv.UnpickWord(context.Background(), 456)
	require.Error(t, err)

	var se *ServiceError
	require.True(t, errors.As(err, &se))
	require.Equal(t, http.StatusNotFound, se.StatusCode)
	require.Equal(t, "456", se.Env["pick_id"])
}

func TestAddTag(t *testing.T) {
	var addedTags []store.TagsAddRequest
	mockStore := &mockStore{
		CreateTagsFunc: func(ctx context.Context, r store.TagsCreateRequest) (model.TagIDMap, error) {
			return model.TagIDMap{
				"important": 789,
				"review":    790,
			}, nil
		},
		GetTagsFunc: func(ctx context.Context, r store.TagsGetRequest) (model.TagIDMap, error) {
			return model.TagIDMap{}, nil
		},
		AddTagsFunc: func(ctx context.Context, r store.TagsAddRequest) error {
			addedTags = append(addedTags, r)
			return nil
		},
	}

	srv := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})
	req := UserPickAddTagRequest{
		PickID: 456,
		Tags:   []string{"important", "review"},
	}

	err := srv.AddTags(context.Background(), req)
	require.NoError(t, err)

	require.Len(t, addedTags, 1)
	require.Contains(t, addedTags, store.TagsAddRequest{
		PickID: 456,
		TagIDs: []int64{789, 790},
	})
}

func TestAddTag_PickNotFound(t *testing.T) {
	mockStore := &mockStore{
		CreateTagsFunc: func(ctx context.Context, r store.TagsCreateRequest) (model.TagIDMap, error) {
			return model.TagIDMap{}, nil
		},
		GetTagsFunc: func(ctx context.Context, r store.TagsGetRequest) (model.TagIDMap, error) {
			return model.TagIDMap{}, nil
		},
		AddTagsFunc: func(ctx context.Context, r store.TagsAddRequest) error {
			return store.ErrNotFound
		},
	}

	srv := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})
	req := UserPickAddTagRequest{
		PickID: 456,
		Tags:   []string{"important"},
	}

	err := srv.AddTags(context.Background(), req)
	require.Error(t, err)

	var se *ServiceError
	require.True(t, errors.As(err, &se))
	require.Equal(t, http.StatusNotFound, se.StatusCode)
	require.Equal(t, "456", se.Env["pick_id"])
}

func TestRemoveTag(t *testing.T) {
	var removedTags []store.TagsRemoveRequest
	mockStore := &mockStore{
		RemoveTagsFunc: func(ctx context.Context, r store.TagsRemoveRequest) error {
			removedTags = append(removedTags, r)
			return nil
		},
	}

	srv := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})
	req := UserPickRemoveTagRequest{
		PickID: 456,
		TagID:  789,
	}

	err := srv.RemoveTag(context.Background(), req)
	require.NoError(t, err)

	require.Len(t, removedTags, 1)
	require.Contains(t, removedTags, store.TagsRemoveRequest{
		PickID: 456,
		TagIDs: []int64{789},
	})
}

func TestRemoveTag_PickNotFound(t *testing.T) {
	mockStore := &mockStore{
		RemoveTagsFunc: func(ctx context.Context, r store.TagsRemoveRequest) error {
			return store.ErrNotFound
		},
	}

	srv := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})
	req := UserPickRemoveTagRequest{
		PickID: 456,
		TagID:  789,
	}

	err := srv.RemoveTag(context.Background(), req)
	require.Error(t, err)

	var se *ServiceError
	require.True(t, errors.As(err, &se))
	require.Equal(t, http.StatusNotFound, se.StatusCode)
	require.Equal(t, "456", se.Env["pick_id"])
	require.Equal(t, "789", se.Env["tag_id"])
}
