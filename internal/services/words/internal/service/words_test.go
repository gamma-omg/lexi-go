package service

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"

	"github.com/gamma-omg/lexi-go/internal/services/words/internal/model"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockStore struct {
	insertWord         func(ctx context.Context, r store.InsertWordRequst) (int64, error)
	deleteWord         func(ctx context.Context, r store.DeleteWordRequest) error
	CreateUserPickFunc func(ctx context.Context, r store.CreateUserPickRequest) (int64, error)
	GetUserPicksFunc   func(ctx context.Context, r store.GetUserPicksRequest) (store.GetUserPicksResponse, error)
	DeleteUserPickFunc func(ctx context.Context, r store.DeleteUserPickRequest) error
	CreateTagsFunc     func(ctx context.Context, r store.CreateTagsRequest) (model.TagIDMap, error)
	GetTagsFunc        func(ctx context.Context, r store.GetTagsRequest) (model.TagIDMap, error)
	AddTagsFunc        func(ctx context.Context, r store.AddTagsRequest) error
	RemoveTagsFunc     func(ctx context.Context, r store.RemoveTagsRequest) error
}

func (m *mockStore) InsertWord(ctx context.Context, r store.InsertWordRequst) (int64, error) {
	return m.insertWord(ctx, r)
}

func (m *mockStore) DeleteWord(ctx context.Context, r store.DeleteWordRequest) error {
	return m.deleteWord(ctx, r)
}

func (m *mockStore) CreateUserPick(ctx context.Context, r store.CreateUserPickRequest) (int64, error) {
	return m.CreateUserPickFunc(ctx, r)
}

func (m *mockStore) GetUserPicks(ctx context.Context, r store.GetUserPicksRequest) (store.GetUserPicksResponse, error) {
	return m.GetUserPicksFunc(ctx, r)
}

func (m *mockStore) DeleteUserPick(ctx context.Context, r store.DeleteUserPickRequest) error {
	return m.DeleteUserPickFunc(ctx, r)
}

func (m *mockStore) CreateTags(ctx context.Context, r store.CreateTagsRequest) (model.TagIDMap, error) {
	return m.CreateTagsFunc(ctx, r)
}

func (m *mockStore) GetTags(ctx context.Context, r store.GetTagsRequest) (model.TagIDMap, error) {
	return m.GetTagsFunc(ctx, r)
}

func (m *mockStore) AddTags(ctx context.Context, r store.AddTagsRequest) error {
	return m.AddTagsFunc(ctx, r)
}

func (m *mockStore) RemoveTags(ctx context.Context, r store.RemoveTagsRequest) error {
	return m.RemoveTagsFunc(ctx, r)
}

func (m *mockStore) WithinTx(ctx context.Context, fn func(tx store.DataStore) error) error {
	return fn(m)
}

func TestAddWord(t *testing.T) {
	var insertedWords []store.InsertWordRequst
	mockStore := &mockStore{
		insertWord: func(ctx context.Context, r store.InsertWordRequst) (int64, error) {
			insertedWords = append(insertedWords, r)
			return 1, nil
		},
	}

	service := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})
	req := AddWordRequest{
		Lemma: "example",
		Lang:  "en",
		Class: model.Noun,
	}

	_, err := service.AddWord(context.Background(), req)
	require.NoError(t, err)

	require.Len(t, insertedWords, 1)
	require.Contains(t, insertedWords, store.InsertWordRequst{
		Lemma: req.Lemma,
		Lang:  req.Lang,
		Class: req.Class,
	})
}

func TestAddWord_Exists(t *testing.T) {
	mockStore := &mockStore{
		insertWord: func(ctx context.Context, r store.InsertWordRequst) (int64, error) {
			return 0, store.ErrExists
		},
	}

	service := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})
	req := AddWordRequest{
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
	var deletedWords []store.DeleteWordRequest
	mockStore := &mockStore{
		deleteWord: func(ctx context.Context, r store.DeleteWordRequest) error {
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
	require.Contains(t, deletedWords, store.DeleteWordRequest{
		ID: wordID,
	})
}

func TestDeleteWord_NotFound(t *testing.T) {
	mockStore := &mockStore{
		deleteWord: func(ctx context.Context, r store.DeleteWordRequest) error {
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
	var createdPicks []store.CreateUserPickRequest
	var addedTags []store.AddTagsRequest
	mockStore := &mockStore{
		CreateTagsFunc: func(ctx context.Context, r store.CreateTagsRequest) (model.TagIDMap, error) {
			return model.TagIDMap{
				"important": 100,
				"review":    200,
			}, nil
		},
		GetTagsFunc: func(ctx context.Context, r store.GetTagsRequest) (model.TagIDMap, error) {
			return model.TagIDMap{}, nil
		},
		AddTagsFunc: func(ctx context.Context, r store.AddTagsRequest) error {
			addedTags = append(addedTags, r)
			return nil
		},
		CreateUserPickFunc: func(ctx context.Context, r store.CreateUserPickRequest) (int64, error) {
			createdPicks = append(createdPicks, r)
			return int64(len(createdPicks)), nil
		},
	}

	srv := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})

	err := srv.PickWord(context.Background(), PickWoardRequest{
		UserID: "user-123",
		WordID: 456,
		DefID:  789,
		Tags:   []string{"important", "review"},
	})
	require.NoError(t, err)

	require.Len(t, createdPicks, 1)
	require.Contains(t, createdPicks, store.CreateUserPickRequest{
		UserID: "user-123",
		DefID:  789,
	})

	require.Len(t, addedTags, 1)
	require.Contains(t, addedTags, store.AddTagsRequest{
		PickID: 1,
		TagIDs: []int64{100, 200},
	})
}

func TestPickWord_Exists(t *testing.T) {
	mockStore := &mockStore{
		CreateUserPickFunc: func(ctx context.Context, r store.CreateUserPickRequest) (int64, error) {
			return 0, store.ErrExists
		},
	}

	srv := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})

	err := srv.PickWord(context.Background(), PickWoardRequest{
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
	var deletedPicks []store.DeleteUserPickRequest
	mockStore := &mockStore{
		DeleteUserPickFunc: func(ctx context.Context, r store.DeleteUserPickRequest) error {
			deletedPicks = append(deletedPicks, r)
			return nil
		},
	}

	srv := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})

	err := srv.UnpickWord(context.Background(), 456)
	require.NoError(t, err)

	require.Len(t, deletedPicks, 1)
	require.Contains(t, deletedPicks, store.DeleteUserPickRequest{PickID: 456})
}

func TestGetUserPicks(t *testing.T) {
	var requests []store.GetUserPicksRequest
	tagIDs := map[string]int64{
		"tag1": 1,
		"tag2": 2,
		"tag3": 3,
	}
	mockStore := &mockStore{
		GetUserPicksFunc: func(ctx context.Context, r store.GetUserPicksRequest) (store.GetUserPicksResponse, error) {
			slices.Sort(r.WithTags)
			slices.Sort(r.WithoutTags)
			requests = append(requests, r)
			return store.GetUserPicksResponse{}, nil
		},
		GetTagsFunc: func(ctx context.Context, r store.GetTagsRequest) (model.TagIDMap, error) {
			ret := make(model.TagIDMap)
			for _, tag := range r.Tags {
				ret[tag] = tagIDs[tag]
			}
			return ret, nil
		},
	}

	srv := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})

	_, err := srv.GetUserPicks(context.Background(), GetUserPicksRequest{
		UserID:      "user-123",
		WithTags:    []string{"tag1", "tag2"},
		WithoutTags: []string{"tag3"},
		PageSize:    10,
	})
	require.NoError(t, err)

	require.Len(t, requests, 1)
	require.Contains(t, requests, store.GetUserPicksRequest{
		UserID:      "user-123",
		WithTags:    []int64{1, 2},
		WithoutTags: []int64{3},
		PageSize:    10,
	})
}

func TestGetUserPicks_MissingTag(t *testing.T) {
	mockStore := &mockStore{
		GetTagsFunc: func(ctx context.Context, r store.GetTagsRequest) (model.TagIDMap, error) {
			return model.TagIDMap{
				"tag1": 1,
			}, nil
		},
	}

	srv := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})

	resp, err := srv.GetUserPicks(context.Background(), GetUserPicksRequest{
		UserID:      "user-123",
		WithTags:    []string{"tag1", "tag2"},
		WithoutTags: []string{},
		PageSize:    10,
	})
	require.NoError(t, err)
	require.Empty(t, resp.Picks)
}

func TestGetUserPicks_InvalidExcludeTag(t *testing.T) {
	var requests []store.GetUserPicksRequest
	mockStore := &mockStore{
		GetTagsFunc: func(ctx context.Context, r store.GetTagsRequest) (model.TagIDMap, error) {
			return model.TagIDMap{}, store.ErrNotFound
		},
		GetUserPicksFunc: func(ctx context.Context, r store.GetUserPicksRequest) (store.GetUserPicksResponse, error) {
			slices.Sort(r.WithTags)
			slices.Sort(r.WithoutTags)
			requests = append(requests, r)
			return store.GetUserPicksResponse{}, nil
		},
	}

	srv := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})

	_, err := srv.GetUserPicks(context.Background(), GetUserPicksRequest{
		UserID:      "user-123",
		WithTags:    []string{},
		WithoutTags: []string{"tag2"},
		PageSize:    10,
	})
	require.NoError(t, err)
	require.Len(t, requests, 1)
	assert.Contains(t, requests, store.GetUserPicksRequest{
		UserID:      "user-123",
		WithTags:    []int64{},
		WithoutTags: []int64{},
		PageSize:    10,
	})
}

func TestGetUserPicks_InvalidPaginationCursor(t *testing.T) {
	mockStore := &mockStore{}

	srv := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})

	_, err := srv.GetUserPicks(context.Background(), GetUserPicksRequest{
		UserID:     "user-123",
		NextCursor: "invalid-cursor",
		PageSize:   10,
	})
	require.Error(t, err)

	var se *ServiceError
	require.True(t, errors.As(err, &se))
	assert.Equal(t, http.StatusBadRequest, se.StatusCode)
	assert.Equal(t, "invalid-cursor", se.Env["cursor"])
}

func TestUnpickWord_NotFound(t *testing.T) {
	mockStore := &mockStore{
		DeleteUserPickFunc: func(ctx context.Context, r store.DeleteUserPickRequest) error {
			return store.ErrNotFound
		},
	}

	srv := NewWordsService(mockStore, WordsServiceConfig{
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
	var addedTags []store.AddTagsRequest
	mockStore := &mockStore{
		CreateTagsFunc: func(ctx context.Context, r store.CreateTagsRequest) (model.TagIDMap, error) {
			return model.TagIDMap{
				"important": 789,
				"review":    790,
			}, nil
		},
		GetTagsFunc: func(ctx context.Context, r store.GetTagsRequest) (model.TagIDMap, error) {
			return model.TagIDMap{}, nil
		},
		AddTagsFunc: func(ctx context.Context, r store.AddTagsRequest) error {
			addedTags = append(addedTags, r)
			return nil
		},
	}

	srv := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})
	req := AddTagsRequest{
		PickID: 456,
		Tags:   []string{"important", "review"},
	}

	err := srv.AddTags(context.Background(), req)
	require.NoError(t, err)

	require.Len(t, addedTags, 1)
	require.Contains(t, addedTags, store.AddTagsRequest{
		PickID: 456,
		TagIDs: []int64{789, 790},
	})
}

func TestAddTag_PickNotFound(t *testing.T) {
	mockStore := &mockStore{
		CreateTagsFunc: func(ctx context.Context, r store.CreateTagsRequest) (model.TagIDMap, error) {
			return model.TagIDMap{}, nil
		},
		GetTagsFunc: func(ctx context.Context, r store.GetTagsRequest) (model.TagIDMap, error) {
			return model.TagIDMap{}, nil
		},
		AddTagsFunc: func(ctx context.Context, r store.AddTagsRequest) error {
			return store.ErrNotFound
		},
	}

	srv := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})
	req := AddTagsRequest{
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
	var removedTags []store.RemoveTagsRequest
	mockStore := &mockStore{
		RemoveTagsFunc: func(ctx context.Context, r store.RemoveTagsRequest) error {
			removedTags = append(removedTags, r)
			return nil
		},
	}

	srv := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})
	req := RemoveTagRequest{
		PickID: 456,
		TagID:  789,
	}

	err := srv.RemoveTag(context.Background(), req)
	require.NoError(t, err)

	require.Len(t, removedTags, 1)
	require.Contains(t, removedTags, store.RemoveTagsRequest{
		PickID: 456,
		TagIDs: []int64{789},
	})
}

func TestRemoveTag_PickNotFound(t *testing.T) {
	mockStore := &mockStore{
		RemoveTagsFunc: func(ctx context.Context, r store.RemoveTagsRequest) error {
			return store.ErrNotFound
		},
	}

	srv := NewWordsService(mockStore, WordsServiceConfig{
		TagsCacheSize: 100,
		TagsMaxCost:   100,
	})
	req := RemoveTagRequest{
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
