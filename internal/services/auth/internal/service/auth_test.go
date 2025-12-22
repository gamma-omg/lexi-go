package service

import (
	"context"
	"testing"

	"github.com/gamma-omg/lexi-go/internal/pkg/serr"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/oauth"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/store"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAuthenticator struct {
	loginFunc    func(env oauth.Env, providerName string) (string, error)
	exchangeFunc func(ctx context.Context, env oauth.Env, providerName, code, state string) (oauth.User, error)
}

func (m *mockAuthenticator) LoginURL(env oauth.Env, providerName string) (string, error) {
	return m.loginFunc(env, providerName)
}

func (m *mockAuthenticator) Exchange(ctx context.Context, env oauth.Env, providerName, code, state string) (oauth.User, error) {
	return m.exchangeFunc(ctx, env, providerName, code, state)
}

type mockStore struct {
	getUserIdentityFunc    func(ctx context.Context, r store.GetUserIdentityRequest) (store.Identity, error)
	createUserFunc         func(ctx context.Context) (int64, error)
	createUserIdentityFunc func(ctx context.Context, r store.CreateUserIdentityRequest) (string, error)
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
	issueFunc func(claims token.UserClaims) (string, error)
}

func (m *mockTokenIssuer) Issue(claims token.UserClaims) (string, error) {
	return m.issueFunc(claims)
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

func TestAuth_LoginURL(t *testing.T) {
	srv := NewAuth(
		WithAuthenticator(&mockAuthenticator{
			loginFunc: func(env oauth.Env, providerName string) (string, error) {
				return "http://example.com/login", nil
			},
		}),
		WithStore(&mockStore{}),
		WithAccessToken(&mockTokenIssuer{}),
		WithRefreshToken(&mockTokenIssuer{}),
	)

	url, err := srv.LoginURL("google", newMockEnv())
	require.NoError(t, err)
	require.Equal(t, "http://example.com/login", url)
}

func TestAuth_LoginURL_ProviderNotFound(t *testing.T) {
	srv := NewAuth(
		WithAuthenticator(&mockAuthenticator{
			loginFunc: func(env oauth.Env, providerName string) (string, error) {
				return "", oauth.ErrProviderNotFound
			},
		}),
		WithStore(&mockStore{}),
		WithAccessToken(&mockTokenIssuer{}),
		WithRefreshToken(&mockTokenIssuer{}),
	)

	_, err := srv.LoginURL("unknown_provider", newMockEnv())
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
					ID:    "user123",
					Email: "test@example.com",
				}, nil
			},
		}),
		WithStore(&mockStore{
			getUserIdentityFunc: func(ctx context.Context, r store.GetUserIdentityRequest) (store.Identity, error) {
				return store.Identity{
					ID:       r.ID,
					Provider: r.Provider,
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
	)

	resp, err := srv.AuthCallback(context.Background(), newMockEnv(), AuthCallbackRequest{
		Provider: "google",
		Code:     "auth_code_123",
		State:    "state_123",
	})
	require.NoError(t, err)

	assert.Equal(t, "access_token", resp.AccessToken)
	assert.Equal(t, "refresh_token", resp.RefreshToken)
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
			getUserIdentityFunc: func(ctx context.Context, r store.GetUserIdentityRequest) (store.Identity, error) {
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
	)

	resp, err := srv.AuthCallback(context.Background(), newMockEnv(), AuthCallbackRequest{
		Provider: "google",
		Code:     "auth_code",
		State:    "valid_state",
	})
	require.NoError(t, err)

	assert.Equal(t, "access_token_new", resp.AccessToken)
	assert.Equal(t, "refresh_token_new", resp.RefreshToken)

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
			getUserIdentityFunc: func(ctx context.Context, r store.GetUserIdentityRequest) (store.Identity, error) {
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
