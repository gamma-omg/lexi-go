package rest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gamma-omg/lexi-go/internal/pkg/httpx"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/oauth"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/service"
)

type authService interface {
	LoginURL(provider string, env oauth.Env) (string, error)
	AuthCallback(ctx context.Context, env oauth.Env, req service.AuthCallbackRequest) (service.AuthCallbackResponse, error)
	Refresh(ctx context.Context, refreshToken string) (string, error)
}

type API struct {
	srv authService
	mux *http.ServeMux
}

func NewAPI(srv authService) *API {
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
	a.mux.HandleFunc("/{provider}/login", a.handleLogin)
	a.mux.HandleFunc("/{provider}/callback", a.handleCallback)
	a.mux.HandleFunc("POST /refresh", a.handleRefresh)
}

func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	p := r.PathValue("provider")
	url, err := a.srv.LoginURL(p, oauth.NewHTTPEnv(w, r))
	if err != nil {
		httpx.HandleErr(w, r, err)
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
		httpx.HandleErr(w, r, err)
		return
	}

	err = httpx.WriteJSON(w, http.StatusOK, callbackResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
	})
	if err != nil {
		httpx.HandleErr(w, r, err)
		return
	}
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshResponse struct {
	AccessToken string `json:"access_token"`
}

func (a *API) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.HandleErr(w, r, fmt.Errorf("read request json: %w", err))
		return
	}

	accessToken, err := a.srv.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		httpx.HandleErr(w, r, err)
		return
	}

	err = httpx.WriteJSON(w, http.StatusOK, refreshResponse{
		AccessToken: accessToken,
	})
	if err != nil {
		httpx.HandleErr(w, r, fmt.Errorf("write response json: %w", err))
		return
	}
}
