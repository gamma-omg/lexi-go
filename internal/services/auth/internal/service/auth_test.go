package service

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gamma-omg/lexi-go/internal/pkg/serr"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/oauth"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/store"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAuthenticator struct {
	loginFunc    func(env oauth.Env, providerName string, state, nonce string) (string, error)
	exchangeFunc func(ctx context.Context, env oauth.Env, providerName, code, state string) (oauth.User, error)
}

func (m *mockAuthenticator) LoginURL(env oauth.Env, providerName string, state, nonce string) (string, error) {
	return m.loginFunc(env, providerName, state, nonce)
}

func (m *mockAuthenticator) Exchange(ctx context.Context, env oauth.Env, providerName, code, state string) (oauth.User, error) {
	return m.exchangeFunc(ctx, env, providerName, code, state)
}

type mockStore struct {
	getIdentityFunc        func(ctx context.Context, r store.GetIdentityRequest) (store.Identity, error)
	getUserIdentityFunc    func(ctx context.Context, r store.GetUserIdentityRequest) (store.Identity, error)
	createUserFunc         func(ctx context.Context) (int64, error)
	createUserIdentityFunc func(ctx context.Context, r store.CreateUserIdentityRequest) (string, error)
}

func (m *mockStore) GetIdentity(ctx context.Context, r store.GetIdentityRequest) (store.Identity, error) {
	return m.getIdentityFunc(ctx, r)
}

func (m *mockStore) GetUserIdentity(ctx context.Context, r store.GetUserIdentityRequest) (store.Identity, error) {
	return m.getUserIdentityFunc(ctx, r)
}

func (m *mockStore) CreateUser(ctx context.Context) (int64, error) {
	return m.createUserFunc(ctx)
}

func (m *mockStore) CreateUserIdentity(ctx context.Context, r store.CreateUserIdentityRequest) (string, error) {
	return m.createUserIdentityFunc(ctx, r)
}

func (m *mockStore) WithTx(ctx context.Context, fn func(store.Store) error) error {
	return fn(m)
}

type mockTokenIssuer struct {
	issueFunc    func(claims token.UserClaims) (string, error)
	validateFunc func(token string) (token.UserClaims, error)
}

func (m *mockTokenIssuer) Issue(claims token.UserClaims) (string, error) {
	return m.issueFunc(claims)
}

func (m *mockTokenIssuer) Validate(token string) (token.UserClaims, error) {
	return m.validateFunc(token)
}

type mockEnv struct {
	saveFunc func(key, val string) error
	loadFunc func(key string) (string, error)
}

func newMockEnv() *mockEnv {
	return &mockEnv{
		saveFunc: func(key, val string) error {
			return nil
		},
		loadFunc: func(key string) (string, error) {
			return "", nil
		},
	}
}

func (m *mockEnv) Save(key, val string) error {
	return m.saveFunc(key, val)
}

func (m *mockEnv) Load(key string) (string, error) {
	return m.loadFunc(key)
}

type mockOTC struct {
	createCodeFunc func(ctx context.Context, ts TokenPair) (string, error)
	redeemCodeFunc func(ctx context.Context, code string) (TokenPair, error)
}

func (m *mockOTC) CreateCode(ctx context.Context, ts TokenPair) (string, error) {
	return m.createCodeFunc(ctx, ts)
}

func (m *mockOTC) RedeemCode(ctx context.Context, code string) (TokenPair, error) {
	return m.redeemCodeFunc(ctx, code)
}

func TestAuth_LoginURL(t *testing.T) {
	srv := NewAuth(
		WithAuthenticator(&mockAuthenticator{
			loginFunc: func(env oauth.Env, providerName string, state, nonce string) (string, error) {
				return "http://example.com/login", nil
			},
		}),
		WithStore(&mockStore{}),
		WithAccessToken(&mockTokenIssuer{}),
		WithRefreshToken(&mockTokenIssuer{}),
		WithOTC(&mockOTC{}),
	)

	url, err := srv.LoginURL(newMockEnv(), LoginRequest{
		Provider:    "google",
		RedirectURL: "/redirect",
	})
	require.NoError(t, err)
	require.Equal(t, "http://example.com/login", url)
}

func TestAuth_LoginURL_ProviderNotFound(t *testing.T) {
	srv := NewAuth(
		WithAuthenticator(&mockAuthenticator{
			loginFunc: func(env oauth.Env, providerName string, state, nonce string) (string, error) {
				return "", oauth.ErrProviderNotFound
			},
		}),
		WithStore(&mockStore{}),
		WithAccessToken(&mockTokenIssuer{}),
		WithRefreshToken(&mockTokenIssuer{}),
		WithOTC(&mockOTC{}),
	)

	_, err := srv.LoginURL(newMockEnv(), LoginRequest{
		Provider:    "unknown_provider",
		RedirectURL: "/redirect",
	})
	require.Error(t, err)

	var sErr *serr.ServiceError
	require.ErrorAs(t, err, &sErr)
	assert.Equal(t, 404, sErr.StatusCode)
	assert.Equal(t, "unknown_provider", sErr.Env["provider"])
}

func TestAuth_AuthCallback_UserExists(t *testing.T) {
	srv := NewAuth(
		WithAuthenticator(&mockAuthenticator{
			exchangeFunc: func(ctx context.Context, env oauth.Env, providerName, code, state string) (oauth.User, error) {
				return oauth.User{
					ID:            "user123",
					Email:         "test@example.com",
					Name:          "Test User",
					Picture:       "http://example.com/avatar.png",
					EmailVerified: true,
				}, nil
			},
		}),
		WithStore(&mockStore{
			getIdentityFunc: func(ctx context.Context, r store.GetIdentityRequest) (store.Identity, error) {
				return store.Identity{
					ID:       r.ID,
					Provider: r.Provider,
					Name:     "Test User",
					Email:    "test@example.com",
					Picture:  "http://example.com/avatar.png",
					User: store.User{
						ID:  1,
						UID: "uid-123",
					},
				}, nil
			},
		}),
		WithAccessToken(&mockTokenIssuer{
			issueFunc: func(claims token.UserClaims) (string, error) {
				return "access_token", nil
			},
		}),
		WithRefreshToken(&mockTokenIssuer{
			issueFunc: func(claims token.UserClaims) (string, error) {
				return "refresh_token", nil
			},
		}),
		WithOTC(&mockOTC{
			createCodeFunc: func(ctx context.Context, ts TokenPair) (string, error) {
				return "123", nil
			},
		}),
	)

	env := &mockEnv{
		loadFunc: func(key string) (string, error) {
			if key == "redirect_url" {
				return "/redirect", nil
			}
			return "", nil
		},
	}

	resp, err := srv.AuthCallback(context.Background(), env, AuthCallbackRequest{
		Provider: "google",
		Code:     "auth_code_123",
		State:    "valid_state",
	})
	require.NoError(t, err)

	assert.Equal(t, "uid-123", resp.UID)
	assert.Equal(t, "Test User", resp.Name)
	assert.Equal(t, "test@example.com", resp.Email)
	assert.Equal(t, "http://example.com/avatar.png", resp.Picture)
	assert.Equal(t, "access_token", resp.AccessToken)
	assert.Equal(t, "refresh_token", resp.RefreshToken)
	assert.Equal(t, "/redirect?otc=123", resp.RedirectURL)
	assert.Equal(t, "123", resp.OTC)
}

func TestAuth_AuthCallback_NewUser(t *testing.T) {
	identities := make(map[string]store.Identity)

	srv := NewAuth(
		WithAuthenticator(&mockAuthenticator{
			exchangeFunc: func(ctx context.Context, env oauth.Env, providerName, code, state string) (oauth.User, error) {
				return oauth.User{
					ID:            "user_identity_1",
					Name:          "Test User",
					Picture:       "http://example.com/avatar.png",
					Email:         "test@example.com",
					EmailVerified: true,
				}, nil
			},
		}),
		WithStore(&mockStore{
			getIdentityFunc: func(ctx context.Context, r store.GetIdentityRequest) (store.Identity, error) {
				id, ok := identities[r.ID]
				if !ok {
					return store.Identity{}, store.ErrNotFound
				}

				return id, nil
			},
			createUserFunc: func(ctx context.Context) (int64, error) {
				return 2, nil
			},
			createUserIdentityFunc: func(ctx context.Context, r store.CreateUserIdentityRequest) (string, error) {
				identities[r.ID] = store.Identity{
					ID:       r.ID,
					Provider: r.Provider,
					Email:    r.Email,
					Name:     r.Name,
					Picture:  r.Picture,
					User: store.User{
						ID:  r.UserID,
						UID: "uid-456",
					},
				}

				return r.ID, nil
			},
		}),
		WithAccessToken(&mockTokenIssuer{
			issueFunc: func(claims token.UserClaims) (string, error) {
				return "access_token_new", nil
			},
		}),
		WithRefreshToken(&mockTokenIssuer{
			issueFunc: func(claims token.UserClaims) (string, error) {
				return "refresh_token_new", nil
			},
		}),
		WithOTC(&mockOTC{
			createCodeFunc: func(ctx context.Context, ts TokenPair) (string, error) {
				return "456", nil
			},
		}),
	)

	env := &mockEnv{
		loadFunc: func(key string) (string, error) {
			if key == "redirect_url" {
				return "/redirect", nil
			}
			return "", nil
		},
	}

	resp, err := srv.AuthCallback(context.Background(), env, AuthCallbackRequest{
		Provider: "google",
		Code:     "auth_code",
		State:    "valid_state",
	})
	require.NoError(t, err)

	assert.Equal(t, "uid-456", resp.UID)
	assert.Equal(t, "Test User", resp.Name)
	assert.Equal(t, "http://example.com/avatar.png", resp.Picture)
	assert.Equal(t, "test@example.com", resp.Email)
	assert.Equal(t, "access_token_new", resp.AccessToken)
	assert.Equal(t, "refresh_token_new", resp.RefreshToken)
	assert.Equal(t, "/redirect?otc=456", resp.RedirectURL)
	assert.Equal(t, "456", resp.OTC)

	expectedID, ok := identities["user_identity_1"]
	require.True(t, ok)
	assert.Equal(t, "user_identity_1", expectedID.ID)
	assert.Equal(t, "google", expectedID.Provider)
	assert.Equal(t, "test@example.com", expectedID.Email)
	assert.Equal(t, "Test User", expectedID.Name)
	assert.Equal(t, "http://example.com/avatar.png", expectedID.Picture)
	assert.Equal(t, "uid-456", expectedID.User.UID)
	assert.Equal(t, int64(2), expectedID.User.ID)
}

func TestAuth_AuthCallback_ProviderNotFound(t *testing.T) {
	srv := NewAuth(
		WithAuthenticator(&mockAuthenticator{
			exchangeFunc: func(ctx context.Context, env oauth.Env, providerName, code, state string) (oauth.User, error) {
				return oauth.User{}, oauth.ErrProviderNotFound
			},
		}),
		WithStore(&mockStore{}),
		WithAccessToken(&mockTokenIssuer{}),
		WithRefreshToken(&mockTokenIssuer{}),
		WithOTC(&mockOTC{}),
	)

	_, err := srv.AuthCallback(context.Background(), newMockEnv(), AuthCallbackRequest{
		Provider: "unknown_provider",
		Code:     "auth_code_123",
		State:    "state_123",
	})
	require.Error(t, err)

	var sErr *serr.ServiceError
	require.ErrorAs(t, err, &sErr)
	assert.Equal(t, 404, sErr.StatusCode)
	assert.Equal(t, "unknown_provider", sErr.Env["provider"])
}

func TestAuth_AuthCallback_AuthFailed(t *testing.T) {
	srv := NewAuth(
		WithAuthenticator(&mockAuthenticator{
			exchangeFunc: func(ctx context.Context, env oauth.Env, providerName, code, state string) (oauth.User, error) {
				return oauth.User{}, oauth.ErrAuthFailed
			},
		}),
		WithStore(&mockStore{}),
		WithAccessToken(&mockTokenIssuer{}),
		WithRefreshToken(&mockTokenIssuer{}),
		WithOTC(&mockOTC{}),
	)

	_, err := srv.AuthCallback(context.Background(), newMockEnv(), AuthCallbackRequest{
		Provider: "google",
		Code:     "auth_code_123",
		State:    "state_123",
	})
	require.Error(t, err)

	var sErr *serr.ServiceError
	require.ErrorAs(t, err, &sErr)
	assert.Equal(t, 401, sErr.StatusCode)
	assert.Equal(t, "google", sErr.Env["provider"])
}

func TestAuth_AuthCallback_UnverifiedEmail(t *testing.T) {
	identities := make(map[string]store.Identity)

	srv := NewAuth(
		WithAuthenticator(&mockAuthenticator{
			exchangeFunc: func(ctx context.Context, env oauth.Env, providerName, code, state string) (oauth.User, error) {
				return oauth.User{
					ID:            "user123",
					Email:         "unverified@example.com",
					EmailVerified: false,
				}, nil
			},
		}),
		WithStore(&mockStore{
			getIdentityFunc: func(ctx context.Context, r store.GetIdentityRequest) (store.Identity, error) {
				id, ok := identities[r.ID]
				if !ok {
					return store.Identity{}, store.ErrNotFound
				}

				return id, nil
			},
			createUserFunc: func(ctx context.Context) (int64, error) {
				return 2, nil
			},
			createUserIdentityFunc: func(ctx context.Context, r store.CreateUserIdentityRequest) (string, error) {
				identities[r.ID] = store.Identity{
					ID:       r.ID,
					Provider: r.Provider,
					Email:    r.Email,
					Name:     r.Name,
					Picture:  r.Picture,
					User: store.User{
						ID:  r.UserID,
						UID: "uid-456",
					},
				}

				return r.ID, nil
			},
		}),
		WithAccessToken(&mockTokenIssuer{issueFunc: func(claims token.UserClaims) (string, error) {
			return "access_token", nil
		}}),
		WithRefreshToken(&mockTokenIssuer{issueFunc: func(claims token.UserClaims) (string, error) {
			return "refresh_token", nil
		}}),
		WithOTC(&mockOTC{
			createCodeFunc: func(ctx context.Context, ts TokenPair) (string, error) {
				return "456", nil
			},
		}),
	)

	_, err := srv.AuthCallback(context.Background(), newMockEnv(), AuthCallbackRequest{
		Provider: "google",
		Code:     "auth_code_123",
		State:    "state_123",
	})
	require.NoError(t, err)

	id, ok := identities["user123"]
	require.True(t, ok)

	assert.Empty(t, id.Email)
}

func TestAuth_Refresh(t *testing.T) {
	srv := NewAuth(
		WithAuthenticator(&mockAuthenticator{}),
		WithStore(&mockStore{
			getUserIdentityFunc: func(ctx context.Context, r store.GetUserIdentityRequest) (store.Identity, error) {
				return store.Identity{
					ID:       "identity-123",
					Provider: r.Provider,
					User: store.User{
						ID:  1,
						UID: r.UID,
					},
				}, nil
			},
		}),
		WithAccessToken(&mockTokenIssuer{
			issueFunc: func(claims token.UserClaims) (string, error) {
				return "new_access_token", nil
			},
		}),
		WithRefreshToken(&mockTokenIssuer{
			validateFunc: func(tokenStr string) (token.UserClaims, error) {
				return token.UserClaims{
					ID:   "uid-123",
					Type: token.TypeRefresh,
				}, nil
			},
		}),
		WithOTC(&mockOTC{}),
	)

	accessToken, err := srv.Refresh(context.Background(), "valid_refresh_token")
	require.NoError(t, err)
	require.Equal(t, "new_access_token", accessToken)
}

func TestAuth_Refresh_InvalidToken(t *testing.T) {
	srv := NewAuth(
		WithAuthenticator(&mockAuthenticator{}),
		WithStore(&mockStore{}),
		WithAccessToken(&mockTokenIssuer{}),
		WithOTC(&mockOTC{}),
		WithRefreshToken(&mockTokenIssuer{
			validateFunc: func(tokenStr string) (token.UserClaims, error) {
				return token.UserClaims{}, fmt.Errorf("invalid refresh token")
			},
		}),
	)

	_, err := srv.Refresh(context.Background(), "invalid_refresh_token")
	require.Error(t, err)

	var sErr *serr.ServiceError
	require.ErrorAs(t, err, &sErr)
	assert.Equal(t, http.StatusUnauthorized, sErr.StatusCode)
}
