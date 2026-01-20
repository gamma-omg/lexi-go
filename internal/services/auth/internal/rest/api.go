package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gamma-omg/lexi-go/internal/pkg/httpx"
	"github.com/gamma-omg/lexi-go/internal/pkg/serr"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/oauth"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/service"
)

type authService interface {
	LoginURL(env oauth.Env, r service.LoginRequest) (string, error)
	AuthCallback(ctx context.Context, env oauth.Env, r service.AuthCallbackRequest) (service.AuthCallbackResponse, error)
	Refresh(ctx context.Context, refreshToken string) (string, error)
	RedeemCode(ctx context.Context, code string) (service.TokenPair, error)
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
	a.mux.HandleFunc("POST /internal/redeem", a.handleRedeemCode)
}

func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	p := readProvider(r, "provider")
	redirectURL, err := readRedirectURL(r, "redirect_url")
	if err != nil {
		httpx.HandleErr(w, r, serr.NewServiceError(err, http.StatusBadRequest, "invalid redirect url"))
		return
	}

	url, err := a.srv.LoginURL(oauth.NewHTTPEnv(p, w, r), service.LoginRequest{
		Provider:    p,
		RedirectURL: redirectURL,
	})
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
	p := readProvider(r, "provider")
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	resp, err := a.srv.AuthCallback(r.Context(), oauth.NewHTTPEnv(p, w, r), service.AuthCallbackRequest{
		Provider: p,
		Code:     code,
		State:    state,
	})
	if err != nil {
		httpx.HandleErr(w, r, err)
		return
	}

	http.Redirect(w, r, resp.RedirectURL, http.StatusFound)
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

type redeemCodeRequest struct {
	Code string `json:"code"`
}

type redeemCodeResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (a *API) handleRedeemCode(w http.ResponseWriter, r *http.Request) {
	var req redeemCodeRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.HandleErr(w, r, serr.NewServiceError(err, http.StatusBadRequest, "invalid request json"))
		return
	}

	tokens, err := a.srv.RedeemCode(r.Context(), req.Code)
	if err != nil {
		httpx.HandleErr(w, r, err)
		return
	}

	err = httpx.WriteJSON(w, http.StatusOK, redeemCodeResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
	if err != nil {
		httpx.HandleErr(w, r, fmt.Errorf("write response json: %w", err))
		return
	}
}

func readProvider(r *http.Request, name string) string {
	p := r.PathValue(name)
	p = strings.TrimSpace(p)
	p = strings.ToLower(p)
	return p
}

func readRedirectURL(r *http.Request, name string) (string, error) {
	redirectURL := r.URL.Query().Get(name)
	if redirectURL == "" {
		return "", errors.New("redirect url must be relative")
	}

	return redirectURL, nil
}
