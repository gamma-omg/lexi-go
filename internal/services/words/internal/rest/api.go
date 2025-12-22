package rest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gamma-omg/lexi-go/internal/pkg/httpx"
	"github.com/gamma-omg/lexi-go/internal/pkg/middleware"
	"github.com/gamma-omg/lexi-go/internal/pkg/serr"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/fn"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/model"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/service"
)

type wordsService interface {
	AddWord(ctx context.Context, r service.AddWordRequest) (int64, error)
	DeleteWord(ctx context.Context, wordID int64) error
	PickWord(ctx context.Context, r service.PickWoardRequest) (int64, error)
	UnpickWord(ctx context.Context, pickID int64) error
	GetUserPicks(ctx context.Context, r service.GetUserPicksRequest) (service.GetUserPicksResponse, error)
	RemoveTags(ctx context.Context, r service.RemoveTagsRequest) error
	CreateDefinition(ctx context.Context, r service.CreateDefinitionRequest) (int64, error)
	AttachImage(ctx context.Context, r service.AttachImageRequest) (service.AttachImageResponse, error)
}

type imageStore interface {
	SaveImage(ctx context.Context, img io.Reader) (*url.URL, error)
}

type API struct {
	srv      wordsService
	imgStore imageStore
	mux      http.ServeMux
}

func NewAPI(srv wordsService, imgStore imageStore) *API {
	api := &API{
		srv:      srv,
		imgStore: imgStore,
		mux:      *http.NewServeMux(),
	}

	api.mount()
	return api
}

func (api *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.mux.ServeHTTP(w, r)
}

func (api *API) mount() {
	api.mux.HandleFunc("PUT /words", api.handleAddWord)
	api.mux.HandleFunc("DELETE /words/{word_id}", api.handleDeleteWord)
	api.mux.HandleFunc("PUT /picks", api.handlePickWord)
	api.mux.HandleFunc("DELETE /picks/{pick_id}", api.handleDeletePick)
	api.mux.HandleFunc("GET /picks", api.handleGetPicks)
	api.mux.HandleFunc("DELETE /tags", api.handleDeleteTag)
	api.mux.HandleFunc("PUT /definitions", api.handleCreateDefinition)
	api.mux.HandleFunc("PUT /images/{def_id}/{source}", api.handleAttachImage)
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
	var req addWordRequest
	err := httpx.ReadJSON(r, &req)
	if err != nil {
		httpx.HandleErr(w, r, serr.NewServiceError(err, http.StatusBadRequest, "invalid request body"))
		return
	}

	id, err := api.srv.AddWord(r.Context(), service.AddWordRequest{
		Lemma: req.Lemma,
		Lang:  model.Lang(req.Lang),
		Class: model.WordClass(req.Class),
	})
	if err != nil {
		httpx.HandleErr(w, r, err)
		return
	}

	err = httpx.WriteJSON(w, http.StatusCreated, addWordResponse{ID: id})
	if err != nil {
		httpx.HandleErr(w, r, err)
		return
	}
}

func (api *API) handleDeleteWord(w http.ResponseWriter, r *http.Request) {
	wordID, err := idFromRequest(r, "word_id")
	if err != nil {
		httpx.HandleErr(w, r, err)
		return
	}

	err = api.srv.DeleteWord(r.Context(), wordID)
	if err != nil {
		httpx.HandleErr(w, r, err)
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
	var req pickWordRequest
	err := httpx.ReadJSON(r, &req)
	if err != nil {
		httpx.HandleErr(w, r, serr.NewServiceError(err, http.StatusBadRequest, "invalid request body"))
		return
	}

	pickID, err := api.srv.PickWord(r.Context(), service.PickWoardRequest{
		UserID: middleware.UserIDFromContext(r.Context()),
		WordID: req.WordID,
		DefID:  req.DefID,
		Tags:   req.Tags,
	})
	if err != nil {
		httpx.HandleErr(w, r, err)
		return
	}

	err = httpx.WriteJSON(w, http.StatusCreated, pickWordResponse{PickID: pickID})
	if err != nil {
		httpx.HandleErr(w, r, err)
		return
	}
}

func (api *API) handleDeletePick(w http.ResponseWriter, r *http.Request) {
	pickID, err := idFromRequest(r, "pick_id")
	if err != nil {
		httpx.HandleErr(w, r, err)
		return
	}

	err = api.srv.UnpickWord(r.Context(), pickID)
	if err != nil {
		httpx.HandleErr(w, r, err)
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
	var req getPicksRequest
	err := httpx.ReadJSON(r, &req)
	if err != nil {
		httpx.HandleErr(w, r, serr.NewServiceError(err, http.StatusBadRequest, "invalid request body"))
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
		httpx.HandleErr(w, r, err)
		return
	}

	err = httpx.WriteJSON(w, http.StatusOK, getPicksResponse{
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
		httpx.HandleErr(w, r, err)
		return
	}
}

type deleteTagRequest struct {
	PickID int64    `json:"pick_id"`
	Tags   []string `json:"tags"`
}

func (api *API) handleDeleteTag(w http.ResponseWriter, r *http.Request) {
	var req deleteTagRequest
	err := httpx.ReadJSON(r, &req)
	if err != nil {
		httpx.HandleErr(w, r, serr.NewServiceError(err, http.StatusBadRequest, "invalid request body"))
		return
	}

	err = api.srv.RemoveTags(r.Context(), service.RemoveTagsRequest{
		PickID: req.PickID,
		Tags:   req.Tags,
	})
	if err != nil {
		httpx.HandleErr(w, r, err)
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
	var req createDefinitionRequest
	err := httpx.ReadJSON(r, &req)
	if err != nil {
		httpx.HandleErr(w, r, serr.NewServiceError(err, http.StatusBadRequest, "invalid request body"))
		return
	}

	defID, err := api.srv.CreateDefinition(r.Context(), service.CreateDefinitionRequest{
		WordID: req.WordID,
		Text:   req.Def,
		Rarity: float32(req.Rarity),
		Source: model.DataSource(req.Source),
	})
	if err != nil {
		httpx.HandleErr(w, r, err)
		return
	}

	err = httpx.WriteJSON(w, http.StatusCreated, createDefinitionResponse{ID: defID})
	if err != nil {
		httpx.HandleErr(w, r, err)
		return
	}
}

type attachImageResponse struct {
	ImageID  int64    `json:"image_id"`
	ImageURL *url.URL `json:"image_url"`
}

func (api *API) handleAttachImage(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("image")
	if err != nil {
		httpx.HandleErr(w, r, serr.NewServiceError(err, http.StatusBadRequest, "invalid image file"))
		return
	}
	defer file.Close()

	imgUrl, err := api.imgStore.SaveImage(r.Context(), file)
	if err != nil {
		httpx.HandleErr(w, r, fmt.Errorf("save image: %w", err))
		return
	}

	source := r.PathValue("source")
	defID, err := idFromRequest(r, "def_id")
	if err != nil {
		httpx.HandleErr(w, r, serr.NewServiceError(err, http.StatusBadRequest, "invalid def_id parameter"))
		return
	}

	resp, err := api.srv.AttachImage(r.Context(), service.AttachImageRequest{
		DefID:    defID,
		ImageURL: imgUrl,
		Source:   model.DataSource(source),
	})
	if err != nil {
		httpx.HandleErr(w, r, err)
		return
	}

	err = httpx.WriteJSON(w, http.StatusCreated, attachImageResponse{
		ImageID:  resp.ImageID,
		ImageURL: resp.ImageURL,
	})
	if err != nil {
		httpx.HandleErr(w, r, err)
		return
	}
}

func idFromRequest(r *http.Request, param string) (int64, error) {
	idStr := r.PathValue(param)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, serr.NewServiceError(err, http.StatusBadRequest, "invalid id parameter")
	}

	return int64(id), nil
}
