package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gamma-omg/lexi-go/internal/services/words/internal/model"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockWordsService struct {
	AddWordFunc          func(ctx context.Context, r service.AddWordRequest) (int64, error)
	DeleteWordFunc       func(ctx context.Context, wordID int64) error
	PickWordFunc         func(ctx context.Context, r service.PickWoardRequest) (int64, error)
	UnpickWordFunc       func(ctx context.Context, pickID int64) error
	GetUserPicksFunc     func(ctx context.Context, r service.GetUserPicksRequest) (service.GetUserPicksResponse, error)
	RemoveTagsFunc       func(ctx context.Context, r service.RemoveTagsRequest) error
	CreateDefinitionFunc func(ctx context.Context, r service.CreateDefinitionRequest) (int64, error)
}

func (m *mockWordsService) AddWord(ctx context.Context, r service.AddWordRequest) (int64, error) {
	return m.AddWordFunc(ctx, r)
}

func (m *mockWordsService) DeleteWord(ctx context.Context, wordID int64) error {
	return m.DeleteWordFunc(ctx, wordID)
}

func (m *mockWordsService) PickWord(ctx context.Context, r service.PickWoardRequest) (int64, error) {
	return m.PickWordFunc(ctx, r)
}

func (m *mockWordsService) UnpickWord(ctx context.Context, pickID int64) error {
	return m.UnpickWordFunc(ctx, pickID)
}

func (m *mockWordsService) GetUserPicks(ctx context.Context, r service.GetUserPicksRequest) (service.GetUserPicksResponse, error) {
	return m.GetUserPicksFunc(ctx, r)
}

func (m *mockWordsService) RemoveTags(ctx context.Context, r service.RemoveTagsRequest) error {
	return m.RemoveTagsFunc(ctx, r)
}

func (m *mockWordsService) CreateDefinition(ctx context.Context, r service.CreateDefinitionRequest) (int64, error) {
	return m.CreateDefinitionFunc(ctx, r)
}

func sendRequest(t *testing.T, mux *http.ServeMux, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var bodyRW strings.Builder
	enc := json.NewEncoder(&bodyRW)
	err := enc.Encode(body)
	require.NoError(t, err)

	req, err := http.NewRequest(method, path, strings.NewReader(bodyRW.String()))
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	return rec
}

func parseResponse[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()

	dec := json.NewDecoder(rec.Body)
	var resp T
	err := dec.Decode(&resp)
	require.NoError(t, err)

	return resp
}

func TestPUTWord(t *testing.T) {
	req := addWordRequest{
		Lemma: "test",
		Lang:  "en",
		Class: "noun",
	}
	api := &API{
		srv: &mockWordsService{
			AddWordFunc: func(ctx context.Context, r service.AddWordRequest) (int64, error) {
				if r.Lemma == req.Lemma && r.Lang == model.Lang(req.Lang) && r.Class == "noun" {
					return 42, nil
				}

				return 0, errors.New("unexpected request")
			},
		},
	}

	mux := http.NewServeMux()
	api.Register(mux)

	rec := sendRequest(t, mux, "PUT", "/words", req)
	assert.Equal(t, http.StatusCreated, rec.Code)

	resp := parseResponse[addWordResponse](t, rec)
	assert.Equal(t, int64(42), resp.ID)
}

func TestPUTWord_BadRequest(t *testing.T) {
	api := &API{
		srv: &mockWordsService{},
	}

	mux := http.NewServeMux()
	api.Register(mux)

	rec := sendRequest(t, mux, "PUT", "/words", "invalid json")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDELETEWord(t *testing.T) {
	api := &API{
		srv: &mockWordsService{
			DeleteWordFunc: func(ctx context.Context, wordID int64) error {
				if wordID == 123 {
					return nil
				}

				return errors.New("unexpected word ID")
			},
		},
	}

	mux := http.NewServeMux()
	api.Register(mux)

	rec := sendRequest(t, mux, "DELETE", "/words/123", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestDELETEWord_BadRequest(t *testing.T) {
	api := &API{
		srv: &mockWordsService{},
	}

	mux := http.NewServeMux()
	api.Register(mux)

	rec := sendRequest(t, mux, "DELETE", "/words/invalid-id", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPUTPick(t *testing.T) {
	req := pickWordRequest{
		WordID: 123,
		DefID:  456,
	}
	api := &API{
		srv: &mockWordsService{
			PickWordFunc: func(ctx context.Context, r service.PickWoardRequest) (int64, error) {
				if r.WordID == req.WordID && r.DefID == req.DefID {
					return 42, nil
				}

				return 0, errors.New("unexpected request")
			},
		},
	}

	mux := http.NewServeMux()
	api.Register(mux)

	rec := sendRequest(t, mux, "PUT", "/picks", req)
	assert.Equal(t, http.StatusCreated, rec.Code)

	resp := parseResponse[pickWordResponse](t, rec)
	assert.Equal(t, int64(42), resp.PickID)
}

func TestPUTPicks_BadRequest(t *testing.T) {
	api := &API{
		srv: &mockWordsService{},
	}

	mux := http.NewServeMux()
	api.Register(mux)

	rec := sendRequest(t, mux, "PUT", "/picks", "invalid json")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDELETEPick(t *testing.T) {
	api := &API{
		srv: &mockWordsService{
			UnpickWordFunc: func(ctx context.Context, pickID int64) error {
				if pickID == 123 {
					return nil
				}

				return errors.New("unexpected pick ID")
			},
		},
	}

	mux := http.NewServeMux()
	api.Register(mux)

	rec := sendRequest(t, mux, "DELETE", "/picks/123", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestDELETEPick_BadRequest(t *testing.T) {
	api := &API{
		srv: &mockWordsService{},
	}

	mux := http.NewServeMux()
	api.Register(mux)

	rec := sendRequest(t, mux, "DELETE", "/picks/invalid-id", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGETPicks(t *testing.T) {
	req := getPicksRequest{
		UserID: "user-123",
	}
	api := &API{
		srv: &mockWordsService{
			GetUserPicksFunc: func(ctx context.Context, r service.GetUserPicksRequest) (service.GetUserPicksResponse, error) {
				if r.UserID == req.UserID {
					return service.GetUserPicksResponse{
						Picks: []service.UserPick{
							{
								ID:     1,
								UserID: "user-123",
								Word:   "test",
								Def:    "A test definition",
								Tags:   []string{"tag1", "tag2"},
								Lang:   model.Lang("en"),
								Class:  model.WordClass("noun"),
							},
						},
						NextCursor: "",
					}, nil
				}

				return service.GetUserPicksResponse{}, errors.New("unexpected user ID")
			},
		},
	}

	mux := http.NewServeMux()
	api.Register(mux)

	rec := sendRequest(t, mux, "GET", "/picks", req)
	assert.Equal(t, http.StatusOK, rec.Code)

	resp := parseResponse[getPicksResponse](t, rec)
	assert.Len(t, resp.Picks, 1)
	assert.Equal(t, int64(1), resp.Picks[0].ID)
	assert.Equal(t, "user-123", resp.Picks[0].UserID)
	assert.Equal(t, "test", resp.Picks[0].Word)
	assert.Equal(t, "en", resp.Picks[0].Lang)
	assert.Equal(t, "noun", resp.Picks[0].Class)
	assert.Equal(t, "A test definition", resp.Picks[0].Def)
	assert.Equal(t, []string{"tag1", "tag2"}, resp.Picks[0].Tags)
}

func TestGETPicks_BadRequest(t *testing.T) {
	api := &API{
		srv: &mockWordsService{},
	}

	mux := http.NewServeMux()
	api.Register(mux)

	rec := sendRequest(t, mux, "GET", "/picks", "invalid json")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDELETETag(t *testing.T) {
	req := deleteTagRequest{
		PickID: 123,
		Tags:   []string{"tag1", "tag2"},
	}
	api := &API{
		srv: &mockWordsService{
			RemoveTagsFunc: func(ctx context.Context, r service.RemoveTagsRequest) error {
				if r.PickID == req.PickID && len(r.Tags) == len(req.Tags) &&
					r.Tags[0] == req.Tags[0] &&
					r.Tags[1] == req.Tags[1] {
					return nil
				}

				return errors.New("unexpected request")
			},
		},
	}

	mux := http.NewServeMux()
	api.Register(mux)

	rec := sendRequest(t, mux, "DELETE", "/tags", req)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestDELETETag_BadRequest(t *testing.T) {
	api := &API{
		srv: &mockWordsService{},
	}

	mux := http.NewServeMux()
	api.Register(mux)

	rec := sendRequest(t, mux, "DELETE", "/tags", "invalid json")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPUTDefinition(t *testing.T) {
	req := createDefinitionRequest{
		WordID: 123,
		Def:    "Test definition",
		Rarity: 456,
	}
	api := &API{
		srv: &mockWordsService{
			CreateDefinitionFunc: func(ctx context.Context, r service.CreateDefinitionRequest) (int64, error) {
				if r.WordID == req.WordID && r.Text == req.Def && r.Rarity == req.Rarity {
					return 42, nil
				}

				return 0, errors.New("unexpected request")
			},
		},
	}

	mux := http.NewServeMux()
	api.Register(mux)

	rec := sendRequest(t, mux, "PUT", "/definitions", req)
	assert.Equal(t, http.StatusCreated, rec.Code)

	resp := parseResponse[createDefinitionResponse](t, rec)
	assert.Equal(t, int64(42), resp.ID)
}

func TestPUTDefinition_BadRequest(t *testing.T) {
	api := &API{
		srv: &mockWordsService{},
	}

	mux := http.NewServeMux()
	api.Register(mux)

	rec := sendRequest(t, mux, "PUT", "/definitions", "invalid json")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
