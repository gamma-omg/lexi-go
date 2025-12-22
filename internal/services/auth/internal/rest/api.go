package rest

import (
	"encoding/json"
	"net/http"

	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/oauth"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/service"
)

type API struct {
	srv *service.Auth
	mux *http.ServeMux
}

func NewAPI(srv *service.Auth) *API {
	api := &API{
		srv: srv,
		mux: http.NewServeMux(),
	}
	api.mount()
	return api
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}

func (a *API) mount() {
	a.mux.HandleFunc("/auth/{provider}/login", a.handleLogin)
	a.mux.HandleFunc("/auth/{provider}/callback", a.handleCallback)
	a.mux.HandleFunc("/refresh", a.handleRefresh)
}

func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	p := r.PathValue("provider")
	url, err := a.srv.LoginURL(p, oauth.NewHTTPEnv(w, r))
	if err != nil {
		handleError(w, err)
		return
	}

	http.Redirect(w, r, url, http.StatusFound)
}

type callbackResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (a *API) handleCallback(w http.ResponseWriter, r *http.Request) {
	p := r.PathValue("provider")
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	resp, err := a.srv.AuthCallback(r.Context(), oauth.NewHTTPEnv(w, r), service.AuthCallbackRequest{
		Provider: p,
		Code:     code,
		State:    state,
	})
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, callbackResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
	})
}

func (a *API) handleRefresh(w http.ResponseWriter, r *http.Request) {
	// TODO: implement token refresh
}

func handleError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		handleError(w, err)
	}
}
