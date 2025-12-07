package rest

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gamma-omg/lexi-go/internal/pkg/middleware"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/fn"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/model"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/service"
)

type mux interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

type wordsService interface {
	AddWord(ctx context.Context, r service.AddWordRequest) (int64, error)
	DeleteWord(ctx context.Context, wordID int64) error
	PickWord(ctx context.Context, r service.PickWoardRequest) (int64, error)
	UnpickWord(ctx context.Context, pickID int64) error
	GetUserPicks(ctx context.Context, r service.GetUserPicksRequest) (service.GetUserPicksResponse, error)
	RemoveTags(ctx context.Context, r service.RemoveTagsRequest) error
	CreateDefinition(ctx context.Context, r service.CreateDefinitionRequest) (int64, error)
	AttachImage(ctx context.Context, r service.AttachImageRequest) (int64, error)
}

type API struct {
	srv wordsService
}

func NewAPI(srv wordsService) *API {
	return &API{srv: srv}
}

func (api *API) Register(m mux) {
	m.HandleFunc("PUT /words", api.handleAddWord)
	m.HandleFunc("DELETE /words/{word_id}", api.handleDeleteWord)
	m.HandleFunc("PUT /picks", api.handlePickWord)
	m.HandleFunc("DELETE /picks/{pick_id}", api.handleDeletePick)
	m.HandleFunc("GET /picks", api.handleGetPicks)
	m.HandleFunc("DELETE /tags", api.handleDeleteTag)
	m.HandleFunc("PUT /definitions", api.handleCreateDefinition)
	m.HandleFunc("PUT /images", api.handleAttachImage)
}

type addWordRequest struct {
	Lemma string `json:"lemma"`
	Lang  string `json:"lang"`
	Class string `json:"class"`
}

type addWordResponse struct {
	ID int64 `json:"id"`
}

func (api *API) handleAddWord(w http.ResponseWriter, r *http.Request) {
	req, err := parseRequest[addWordRequest](r)
	if err != nil {
		handleErr(w, r, service.NewServiceError(err, http.StatusBadRequest, "invalid request body"))
		return
	}

	id, err := api.srv.AddWord(r.Context(), service.AddWordRequest{
		Lemma: req.Lemma,
		Lang:  model.Lang(req.Lang),
		Class: model.WordClass(req.Class),
	})
	if err != nil {
		handleErr(w, r, err)
		return
	}

	writeResponse(w, http.StatusCreated, addWordResponse{ID: id})
}

func (api *API) handleDeleteWord(w http.ResponseWriter, r *http.Request) {
	wordID, err := idFromRequest(r, "word_id")
	if err != nil {
		handleErr(w, r, err)
		return
	}

	err = api.srv.DeleteWord(r.Context(), wordID)
	if err != nil {
		handleErr(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type pickWordRequest struct {
	UserID string   `json:"user_id"`
	WordID int64    `json:"word_id"`
	DefID  int64    `json:"def_id"`
	Tags   []string `json:"tags"`
}

type pickWordResponse struct {
	PickID int64 `json:"pick_id"`
}

func (api *API) handlePickWord(w http.ResponseWriter, r *http.Request) {
	req, err := parseRequest[pickWordRequest](r)
	if err != nil {
		handleErr(w, r, service.NewServiceError(err, http.StatusBadRequest, "invalid request body"))
		return
	}

	pickID, err := api.srv.PickWord(r.Context(), service.PickWoardRequest{
		UserID: middleware.UserIDFromContext(r.Context()),
		WordID: req.WordID,
		DefID:  req.DefID,
		Tags:   req.Tags,
	})
	if err != nil {
		handleErr(w, r, err)
		return
	}

	writeResponse(w, http.StatusCreated, pickWordResponse{PickID: pickID})
}

func (api *API) handleDeletePick(w http.ResponseWriter, r *http.Request) {
	pickID, err := idFromRequest(r, "pick_id")
	if err != nil {
		handleErr(w, r, err)
		return
	}

	err = api.srv.UnpickWord(r.Context(), pickID)
	if err != nil {
		handleErr(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type getPicksRequest struct {
	UserID      string   `json:"user_id"`
	WithTags    []string `json:"with_tags"`
	WithoutTags []string `json:"without_tags"`
	PageSize    int      `json:"page_size"`
	NextCursor  string   `json:"next_cursor"`
}

type getPicksResponse struct {
	Picks      []userPickResponse `json:"picks"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

type userPickResponse struct {
	ID     int64    `json:"id"`
	UserID string   `json:"user_id"`
	Word   string   `json:"word"`
	Lang   string   `json:"lang"`
	Class  string   `json:"class"`
	Def    string   `json:"def"`
	Tags   []string `json:"tags"`
}

func (api *API) handleGetPicks(w http.ResponseWriter, r *http.Request) {
	req, err := parseRequest[getPicksRequest](r)
	if err != nil {
		handleErr(w, r, service.NewServiceError(err, http.StatusBadRequest, "invalid request body"))
		return
	}

	resp, err := api.srv.GetUserPicks(r.Context(), service.GetUserPicksRequest{
		UserID:      req.UserID,
		WithTags:    req.WithTags,
		WithoutTags: req.WithoutTags,
		PageSize:    req.PageSize,
		NextCursor:  req.NextCursor,
	})
	if err != nil {
		handleErr(w, r, err)
		return
	}

	err = writeResponse(w, http.StatusOK, getPicksResponse{
		Picks: fn.Map(resp.Picks, func(pick service.UserPick) userPickResponse {
			return userPickResponse{
				ID:     pick.ID,
				UserID: pick.UserID,
				Word:   pick.Word,
				Lang:   string(pick.Lang),
				Class:  string(pick.Class),
				Def:    pick.Def,
				Tags:   pick.Tags,
			}
		}),
		NextCursor: resp.NextCursor,
	})
	if err != nil {
		handleErr(w, r, err)
		return
	}
}

type deleteTagRequest struct {
	PickID int64    `json:"pick_id"`
	Tags   []string `json:"tags"`
}

func (api *API) handleDeleteTag(w http.ResponseWriter, r *http.Request) {
	req, err := parseRequest[deleteTagRequest](r)
	if err != nil {
		handleErr(w, r, service.NewServiceError(err, http.StatusBadRequest, "invalid request body"))
		return
	}

	err = api.srv.RemoveTags(r.Context(), service.RemoveTagsRequest{
		PickID: req.PickID,
		Tags:   req.Tags,
	})
	if err != nil {
		handleErr(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type createDefinitionRequest struct {
	WordID int64   `json:"word_id"`
	Def    string  `json:"def"`
	Rarity float32 `json:"rarity"`
	Source string  `json:"source"`
}

type createDefinitionResponse struct {
	ID int64 `json:"id"`
}

func (api *API) handleCreateDefinition(w http.ResponseWriter, r *http.Request) {
	req, err := parseRequest[createDefinitionRequest](r)
	if err != nil {
		handleErr(w, r, service.NewServiceError(err, http.StatusBadRequest, "invalid request body"))
		return
	}

	defID, err := api.srv.CreateDefinition(r.Context(), service.CreateDefinitionRequest{
		WordID: req.WordID,
		Text:   req.Def,
		Rarity: float32(req.Rarity),
		Source: model.DataSource(req.Source),
	})
	if err != nil {
		handleErr(w, r, err)
		return
	}

	writeResponse(w, http.StatusCreated, createDefinitionResponse{ID: defID})
}

type attachImageRequest struct {
	DefID    int64  `json:"def_id"`
	ImageURL string `json:"image_url"`
	Source   string `json:"source"`
}

type attachImageResponse struct {
	ImageID int64 `json:"image_id"`
}

func (api *API) handleAttachImage(w http.ResponseWriter, r *http.Request) {
	req, err := parseRequest[attachImageRequest](r)
	if err != nil {
		handleErr(w, r, service.NewServiceError(err, http.StatusBadRequest, "invalid request body"))
		return
	}

	imageID, err := api.srv.AttachImage(r.Context(), service.AttachImageRequest{
		DefID:    req.DefID,
		ImageURL: req.ImageURL,
		Source:   model.DataSource(req.Source),
	})
	if err != nil {
		handleErr(w, r, err)
		return
	}

	writeResponse(w, http.StatusCreated, attachImageResponse{ImageID: imageID})
}

func idFromRequest(r *http.Request, param string) (int64, error) {
	idStr := r.PathValue(param)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, service.NewServiceError(err, http.StatusBadRequest, "invalid id parameter")
	}

	return int64(id), nil
}

func parseRequest[T any](r *http.Request) (T, error) {
	var req T
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&req)
	return req, err
}

func writeResponse[T any](w http.ResponseWriter, status int, resp T) error {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	return enc.Encode(resp)
}

func handleErr(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request error",
		"error", err,
		"method", r.Method,
		"url", r.URL.String(),
		"remote_addr", r.RemoteAddr,
	)

	var se *service.ServiceError
	if errors.As(err, &se) {
		http.Error(w, se.Msg, se.StatusCode)
		return
	}

	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
