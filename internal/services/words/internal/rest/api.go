package rest

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gamma-omg/lexi-go/internal/pkg/middleware"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/model"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/service"
)

type mux interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

type API struct {
	svc *service.WordsService
}

func NewAPI(svc *service.WordsService) *API {
	return &API{svc: svc}
}

func (api *API) Register(m mux) {
	m.HandleFunc("PUT /words/{lang}/{class}/{lemma}", api.handleAddWord)
	m.HandleFunc("DELETE /words/{word_id}", api.handleDeleteWord)
	m.HandleFunc("PUT /picks/{word_id}/{def_id}", api.handlePickWord)
	m.HandleFunc("DELETE /picks/{pick_id}", api.handleDeletePick)
	m.HandleFunc("PUT /tags/{pick_id}/{tag}", api.handleAddTag)
	m.HandleFunc("DELETE /tags/{pick_id}/{tag_id}", api.handleDeleteTag)
}

func (api *API) handleAddWord(w http.ResponseWriter, r *http.Request) {
	lang := r.PathValue("lang")
	class := r.PathValue("class")
	lemma := r.PathValue("lemma")

	_, err := api.svc.AddWord(r.Context(), service.WordAddRequest{
		Lemma: lemma,
		Lang:  model.Lang(lang),
		Class: model.WordClass(class),
	})
	if err != nil {
		handleErr(w, r, err)
		return
	}
}

func (api *API) handleDeleteWord(w http.ResponseWriter, r *http.Request) {
	wordID, err := idFromRequest(r, "word_id")
	if err != nil {
		handleErr(w, r, err)
		return
	}

	err = api.svc.DeleteWord(r.Context(), wordID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
}

func (api *API) handlePickWord(w http.ResponseWriter, r *http.Request) {
	wordID, err := idFromRequest(r, "word_id")
	if err != nil {
		handleErr(w, r, err)
		return
	}

	defID, err := idFromRequest(r, "def_id")
	if err != nil {
		handleErr(w, r, err)
		return
	}

	err = api.svc.PickWord(r.Context(), service.UserPickWordRequest{
		UserID: middleware.UserIDFromContext(r.Context()),
		WordID: int64(wordID),
		DefID:  int64(defID),
	})
	if err != nil {
		handleErr(w, r, err)
		return
	}
}

func (api *API) handleDeletePick(w http.ResponseWriter, r *http.Request) {
	pickID, err := idFromRequest(r, "pick_id")
	if err != nil {
		handleErr(w, r, err)
		return
	}

	err = api.svc.UnpickWord(r.Context(), pickID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
}

func (api *API) handleAddTag(w http.ResponseWriter, r *http.Request) {
	pickID, err := idFromRequest(r, "pick_id")
	if err != nil {
		handleErr(w, r, err)
		return
	}

	tag := r.PathValue("tag")
	err = api.svc.AddTag(r.Context(), service.UserPickAddTagRequest{
		PickID: pickID,
		Tag:    tag,
	})
	if err != nil {
		handleErr(w, r, err)
		return
	}
}

func (api *API) handleDeleteTag(w http.ResponseWriter, r *http.Request) {
	pickID, err := idFromRequest(r, "pick_id")
	if err != nil {
		handleErr(w, r, err)
		return
	}

	tagID, err := idFromRequest(r, "tag_id")
	if err != nil {
		handleErr(w, r, err)
		return
	}

	err = api.svc.RemoveTag(r.Context(), service.UserPickRemoveTagRequest{
		PickID: pickID,
		TagID:  tagID,
	})
	if err != nil {
		handleErr(w, r, err)
		return
	}
}

func idFromRequest(r *http.Request, param string) (int64, error) {
	idStr := r.PathValue(param)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, service.NewServiceError(err, http.StatusBadRequest, "invalid id parameter")
	}

	return int64(id), nil
}

func handleErr(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request error",
		"error", err,
		"method", r.Method,
		"url", r.URL.String(),
		"remote_addr", r.RemoteAddr,
	)

	se, ok := err.(*service.ServiceError)
	if !ok {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Error(w, se.Msg, se.StatusCode)
}
