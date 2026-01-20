package rest

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gamma-omg/lexi-go/internal/pkg/serr"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/oauth"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/service"
	"github.com/stretchr/testify/assert"
)

type mockAuthService struct {
	loginURLFunc     func(env oauth.Env, r service.LoginRequest) (string, error)
	authCallbackFunc func(ctx context.Context, env oauth.Env, r service.AuthCallbackRequest) (service.AuthCallbackResponse, error)
	refreshFunc      func(ctx context.Context, refreshToken string) (string, error)
	redeemFunc       func(ctx context.Context, code string) (service.TokenPair, error)
}

func (m *mockAuthService) LoginURL(env oauth.Env, r service.LoginRequest) (string, error) {
	return m.loginURLFunc(env, r)
}

func (m *mockAuthService) AuthCallback(ctx context.Context, env oauth.Env, r service.AuthCallbackRequest) (service.AuthCallbackResponse, error) {
	return m.authCallbackFunc(ctx, env, r)
}

func (m *mockAuthService) Refresh(ctx context.Context, refreshToken string) (string, error) {
	return m.refreshFunc(ctx, refreshToken)
}

func (m *mockAuthService) RedeemCode(ctx context.Context, code string) (service.TokenPair, error) {
	return m.redeemFunc(ctx, code)
}

func TestAPI_HandleLogin(t *testing.T) {
	srv := &mockAuthService{
		loginURLFunc: func(env oauth.Env, r service.LoginRequest) (string, error) {
			return "http://example.com/login", nil
		},
	}
	api := NewAPI(srv)

	req := httptest.NewRequest("GET", "/google/login?redirect_url=/redirect", nil)
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	resp := rec.Result()
	assert.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Equal(t, "http://example.com/login", resp.Header.Get("Location"))
}

func TestAPI_HandleLogin_ProviderNotFound(t *testing.T) {
	srv := &mockAuthService{
		loginURLFunc: func(env oauth.Env, r service.LoginRequest) (string, error) {
			return "", serr.NewServiceError(errors.New("test error"), http.StatusNotFound, "not found")
		},
	}

	api := NewAPI(srv)

	req := httptest.NewRequest("GET", "/unknown/login?redirect_url=/redirect", nil)
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	resp := rec.Result()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestAPI_Callback(t *testing.T) {
	srv := &mockAuthService{
		authCallbackFunc: func(ctx context.Context, env oauth.Env, req service.AuthCallbackRequest) (service.AuthCallbackResponse, error) {
			return service.AuthCallbackResponse{
				RedirectURL: "http://example.com/redirect",
			}, nil
		},
	}
	api := NewAPI(srv)

	req := httptest.NewRequest("GET", "/google/callback?code=test_code&state=test_state", nil)
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	resp := rec.Result()
	assert.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Equal(t, "http://example.com/redirect", resp.Header.Get("Location"))
}

func TestAPI_Callback_AuthFailed(t *testing.T) {
	srv := &mockAuthService{
		authCallbackFunc: func(ctx context.Context, env oauth.Env, req service.AuthCallbackRequest) (service.AuthCallbackResponse, error) {
			return service.AuthCallbackResponse{},
				serr.NewServiceError(errors.New("auth failed"), http.StatusUnauthorized, "authentication failed")
		},
	}
	api := NewAPI(srv)

	req := httptest.NewRequest("GET", "/google/callback?code=invalid_code&state=invalid_state", nil)
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	resp := rec.Result()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAPI_HandleRefresh(t *testing.T) {
	srv := &mockAuthService{
		refreshFunc: func(ctx context.Context, refreshToken string) (string, error) {
			return "new_access_token_value", nil
		},
	}
	api := NewAPI(srv)

	req := httptest.NewRequest("POST", "/refresh", strings.NewReader(`{"refresh_token":"valid_refresh_token"}`))
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	resp := rec.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t,
		`{
			"access_token":"new_access_token_value"
		}`,
		rec.Body.String(),
	)
}

func TestAPI_HandleRefresh_Unauthorized(t *testing.T) {
	srv := &mockAuthService{
		refreshFunc: func(ctx context.Context, refreshToken string) (string, error) {
			return "", serr.NewServiceError(errors.New("invalid token"), http.StatusUnauthorized, "unauthorized")
		},
	}
	api := NewAPI(srv)

	req := httptest.NewRequest("POST", "/refresh", strings.NewReader(`{"refresh_token":"invalid_refresh_token"}`))
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	resp := rec.Result()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAPI_HandleRedeemCode(t *testing.T) {
	srv := &mockAuthService{
		redeemFunc: func(ctx context.Context, code string) (service.TokenPair, error) {
			return service.TokenPair{
				AccessToken:  "access_token_value",
				RefreshToken: "refresh_token_value",
			}, nil
		},
	}
	api := NewAPI(srv)

	req := httptest.NewRequest("POST", "/internal/redeem", strings.NewReader(`{"code":"valid_code"}`))
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	resp := rec.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t,
		`{
			"access_token":"access_token_value",
			"refresh_token":"refresh_token_value"
		}`,
		rec.Body.String(),
	)
}
