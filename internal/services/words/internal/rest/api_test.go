package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gamma-omg/lexi-go/internal/services/words/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockWordsService struct {
	AddWordFunc          func(ctx context.Context, r service.AddWordRequest) (int64, error)
	DeleteWordFunc       func(ctx context.Context, wordID int64) error
	PickWordFunc         func(ctx context.Context, r service.PickWoardRequest) error
	UnpickWordFunc       func(ctx context.Context, pickID int64) error
	GetUserPicksFunc     func(ctx context.Context, r service.GetUserPicksRequest) (service.GetUserPicksResponse, error)
	RemoveTagFunc        func(ctx context.Context, r service.RemoveTagRequest) error
	CreateDefinitionFunc func(ctx context.Context, r service.CreateDefinitionRequest) (int64, error)
}

func (m *mockWordsService) AddWord(ctx context.Context, r service.AddWordRequest) (int64, error) {
	return m.AddWordFunc(ctx, r)
}

func (m *mockWordsService) DeleteWord(ctx context.Context, wordID int64) error {
	return m.DeleteWordFunc(ctx, wordID)
}

func (m *mockWordsService) PickWord(ctx context.Context, r service.PickWoardRequest) error {
	return m.PickWordFunc(ctx, r)
}

func (m *mockWordsService) UnpickWord(ctx context.Context, pickID int64) error {
	return m.UnpickWordFunc(ctx, pickID)
}

func (m *mockWordsService) GetUserPicks(ctx context.Context, r service.GetUserPicksRequest) (service.GetUserPicksResponse, error) {
	return m.GetUserPicksFunc(ctx, r)
}

func (m *mockWordsService) RemoveTag(ctx context.Context, r service.RemoveTagRequest) error {
	return m.RemoveTagFunc(ctx, r)
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
	assert.Equal(t, http.StatusOK, rec.Code)

	dec := json.NewDecoder(rec.Body)

	var resp createDefinitionResponse
	err := dec.Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, int64(42), resp.ID)
}
