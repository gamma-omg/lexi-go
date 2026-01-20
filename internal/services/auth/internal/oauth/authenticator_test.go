package oauth

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockIdentityProvider struct {
	loginFunc    func(state, nonce string) (string, error)
	exchangeFunc func(ctx context.Context, code string) (User, error)
}

func (m *mockIdentityProvider) LoginURL(state, nonce string) (string, error) {
	return m.loginFunc(state, nonce)
}

func (m *mockIdentityProvider) Exchange(ctx context.Context, code string) (User, error) {
	return m.exchangeFunc(ctx, code)
}

type memEnv struct {
	store map[string]string
}

func newMemEnv() *memEnv {
	return &memEnv{
		store: make(map[string]string),
	}
}

func (m *memEnv) Save(key, val string) error {
	m.store[key] = val
	return nil
}

func (m *memEnv) Load(key string) (string, error) {
	val, ok := m.store[key]
	if !ok {
		return "", errors.New("key not found")
	}
	return val, nil
}

type mockEnv struct {
	saveFunc func(key, val string) error
	loadFunc func(key string) (string, error)
}

func (m *mockEnv) Save(key, val string) error {
	return m.saveFunc(key, val)
}

func (m *mockEnv) Load(key string) (string, error) {
	return m.loadFunc(key)
}

func TestAuthenticator_LoginURL(t *testing.T) {
	a := NewAuthenticator()
	a.Use("test", &mockIdentityProvider{
		loginFunc: func(state, nonce string) (string, error) {
			return "test_url", nil
		},
		exchangeFunc: func(ctx context.Context, code string) (User, error) {
			return User{}, nil
		},
	})

	url, err := a.LoginURL(newMemEnv(), "test", "some_state", "some_nonce")
	require.NoError(t, err)
	require.Equal(t, "test_url", url)
}

func TestAuthenticator_LoginURL_ProviderNotFound(t *testing.T) {
	a := NewAuthenticator()

	_, err := a.LoginURL(newMemEnv(), "non_existent", "state", "nonce")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrProviderNotFound))
}

func TestAuthenticator_LoginURL_EnvSaveError(t *testing.T) {
	a := NewAuthenticator()
	a.Use("test", &mockIdentityProvider{
		loginFunc: func(state, nonce string) (string, error) {
			return "test_url", nil
		},
		exchangeFunc: func(ctx context.Context, code string) (User, error) {
			return User{}, nil
		},
	})

	brokenEnv := &mockEnv{
		saveFunc: func(key, val string) error {
			return errors.New("save error")
		},
		loadFunc: func(key string) (string, error) {
			return "", nil
		},
	}

	_, err := a.LoginURL(brokenEnv, "test", "state", "nonce")
	require.Error(t, err)
}

func TestAuthenticator_LoginURL_ProviderLoginError(t *testing.T) {
	a := NewAuthenticator()
	a.Use("test", &mockIdentityProvider{
		loginFunc: func(state, nonce string) (string, error) {
			return "", errors.New("login error")
		},
		exchangeFunc: func(ctx context.Context, code string) (User, error) {
			return User{}, nil
		},
	})

	_, err := a.LoginURL(newMemEnv(), "test", "some_state", "some_nonce")
	require.Error(t, err)
}

func TestAuthenticator_Exchange(t *testing.T) {
	a := NewAuthenticator()
	a.Use("test", &mockIdentityProvider{
		loginFunc: func(state, nonce string) (string, error) {
			return "", nil
		},
		exchangeFunc: func(ctx context.Context, code string) (User, error) {
			return User{
				Nonce:         "valid_nonce",
				ID:            "user123",
				Email:         "test@example.com",
				Name:          "Test User",
				Picture:       "http://example.com/user.png",
				EmailVerified: true,
			}, nil
		},
	})

	env := newMemEnv()
	err := errors.Join(
		env.Save("state", "valid_state"),
		env.Save("nonce", "valid_nonce"),
	)
	require.NoError(t, err)

	usr, err := a.Exchange(context.Background(), env, "test", "auth_code_123", "valid_state")
	require.NoError(t, err)
	require.Equal(t, "user123", usr.ID)
	require.Equal(t, "test@example.com", usr.Email)
	require.Equal(t, "Test User", usr.Name)
	require.Equal(t, "http://example.com/user.png", usr.Picture)
	require.True(t, usr.EmailVerified)
}

func TestAuthenticator_Exchange_ProviderNotFound(t *testing.T) {
	a := NewAuthenticator()

	_, err := a.Exchange(context.Background(), newMemEnv(), "non_existent", "code", "state")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrProviderNotFound))
}

func TestAuthenticator_Exchange_EnvLoadError(t *testing.T) {
	a := NewAuthenticator()
	a.Use("test", &mockIdentityProvider{
		loginFunc: func(state, nonce string) (string, error) {
			return "", nil
		},
		exchangeFunc: func(ctx context.Context, code string) (User, error) {
			return User{}, nil
		},
	})

	brokenEnv := &mockEnv{
		saveFunc: func(key, val string) error {
			return nil
		},
		loadFunc: func(key string) (string, error) {
			return "", errors.New("load error")
		},
	}

	_, err := a.Exchange(context.Background(), brokenEnv, "test", "code", "state")
	require.Error(t, err)
}

func TestAuthenticator_Exchange_StateMismatch(t *testing.T) {
	a := NewAuthenticator()
	a.Use("test", &mockIdentityProvider{
		loginFunc: func(state, nonce string) (string, error) {
			return "", nil
		},
		exchangeFunc: func(ctx context.Context, code string) (User, error) {
			return User{}, nil
		},
	})

	env := newMemEnv()
	err := errors.Join(
		env.Save("state", "expected_state"),
		env.Save("nonce", "expected_nonce"),
	)
	require.NoError(t, err)

	_, err = a.Exchange(context.Background(), env, "test", "code", "wrong_state")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrAuthFailed))
}

func TestAuthenticator_Exchange_ProviderMissingNonce(t *testing.T) {
	a := NewAuthenticator()
	a.Use("test", &mockIdentityProvider{
		loginFunc: func(state, nonce string) (string, error) {
			return "", nil
		},
		exchangeFunc: func(ctx context.Context, code string) (User, error) {
			return User{
				Nonce: "",
			}, nil
		},
	})

	env := newMemEnv()
	err := errors.Join(
		env.Save("state", "valid_state"),
		env.Save("nonce", "valid_nonce"),
	)
	require.NoError(t, err)

	_, err = a.Exchange(context.Background(), env, "test", "code", "valid_state")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrAuthFailed))
}

func TestAuthenticator_Exchange_EnvLoadNonceError(t *testing.T) {
	a := NewAuthenticator()
	a.Use("test", &mockIdentityProvider{
		loginFunc: func(state, nonce string) (string, error) {
			return "", nil
		},
		exchangeFunc: func(ctx context.Context, code string) (User, error) {
			return User{
				Nonce: "some_nonce",
			}, nil
		},
	})

	brokenEnv := &mockEnv{
		saveFunc: func(key, val string) error {
			return nil
		},
		loadFunc: func(key string) (string, error) {
			if key == "nonce" {
				return "", errors.New("load nonce error")
			}
			return "some_state", nil
		},
	}

	_, err := a.Exchange(context.Background(), brokenEnv, "test", "code", "some_state")
	require.Error(t, err)
}

func TestAutheticator_Exchange_InvalidNonce(t *testing.T) {
	a := NewAuthenticator()
	a.Use("test", &mockIdentityProvider{
		loginFunc: func(state, nonce string) (string, error) {
			return "", nil
		},
		exchangeFunc: func(ctx context.Context, code string) (User, error) {
			return User{
				Nonce: "invalid_nonce",
			}, nil
		},
	})

	env := newMemEnv()
	err := errors.Join(
		env.Save("state", "valid_state"),
		env.Save("nonce", "expected_nonce"),
	)
	require.NoError(t, err)

	_, err = a.Exchange(context.Background(), env, "test", "code", "valid_state")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrAuthFailed))
}

func TestAuthenticator_Exchange_ProviderNonceMismatch(t *testing.T) {
	a := NewAuthenticator()
	a.Use("test", &mockIdentityProvider{
		loginFunc: func(state, nonce string) (string, error) {
			return "", nil
		},
		exchangeFunc: func(ctx context.Context, code string) (User, error) {
			return User{
				Nonce: "wrong_nonce",
			}, nil
		},
	})

	env := newMemEnv()
	err := errors.Join(
		env.Save("state", "valid_state"),
		env.Save("nonce", "valid_nonce"),
	)
	require.NoError(t, err)

	_, err = a.Exchange(context.Background(), env, "test", "code", "valid_state")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrAuthFailed))
}

func TestAuthenticator_Exchange_ProviderExchangeError(t *testing.T) {
	a := NewAuthenticator()
	a.Use("test", &mockIdentityProvider{
		loginFunc: func(state, nonce string) (string, error) {
			return "", nil
		},
		exchangeFunc: func(ctx context.Context, code string) (User, error) {
			return User{}, errors.New("exchange error")
		},
	})

	env := newMemEnv()
	err := errors.Join(
		env.Save("state", "valid_state"),
		env.Save("nonce", "valid_nonce"),
	)
	require.NoError(t, err)

	_, err = a.Exchange(context.Background(), env, "test", "code", "valid_state")
	require.Error(t, err)
}
