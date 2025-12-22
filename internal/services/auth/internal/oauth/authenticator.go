package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"golang.org/x/oauth2"
)

var (
	ErrProviderConflict = errors.New("provider already exists")
	ErrProviderNotFound = errors.New("provider not found")
	ErrAuthFailed       = errors.New("auth failed")
)

type User struct {
	ID            string
	Email         string
	EmailVerified bool
	Name          string
	Picture       string
}

func (u *User) VerifiedEmail() string {
	if u.EmailVerified {
		return u.Email
	}
	return ""
}

type Env interface {
	Save(key, val string) error
	Load(key string) (string, error)
}

type identityProvider interface {
	LoginURL(state string) (string, error)
	Exchange(ctx context.Context, code string) (User, error)
}

type Authenticator struct {
	providers map[string]identityProvider
	mu        sync.RWMutex
}

func NewAuthenticator() *Authenticator {
	return &Authenticator{
		providers: make(map[string]identityProvider),
	}
}

func (a *Authenticator) Use(name string, p identityProvider) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, ok := a.providers[name]; ok {
		return ErrProviderConflict
	}

	a.providers[name] = p
	return nil
}

func (a *Authenticator) LoginURL(env Env, provider string) (string, error) {
	p, err := a.getProvider(provider)
	if err != nil {
		return "", fmt.Errorf("get provider: %w", err)
	}

	state := randState(32)
	if err = env.Save(provider, state); err != nil {
		return "", fmt.Errorf("save state: %w", err)
	}

	url, err := p.LoginURL(state)
	if err != nil {
		return "", fmt.Errorf("get login url: %w", err)
	}

	return url, nil
}

func (a *Authenticator) Exchange(ctx context.Context, env Env, provider, code, state string) (User, error) {
	p, err := a.getProvider(provider)
	if err != nil {
		return User{}, fmt.Errorf("get provider: %w", err)
	}

	saved, err := env.Load(provider)
	if err != nil {
		return User{}, fmt.Errorf("load state: %w", err)
	}

	if saved != state {
		return User{}, ErrAuthFailed
	}

	usr, err := p.Exchange(ctx, code)
	if err != nil {
		var rerr *oauth2.RetrieveError
		if errors.As(err, &rerr) {
			if rerr.Response != nil {
				if rerr.Response.StatusCode == http.StatusBadRequest || rerr.Response.StatusCode == http.StatusUnauthorized {
					return User{}, ErrAuthFailed
				}
			}
		}

		return User{}, fmt.Errorf("exchange: %w", err)
	}

	return usr, nil
}

func (a *Authenticator) getProvider(name string) (identityProvider, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	p, ok := a.providers[name]
	if !ok {
		return nil, ErrProviderNotFound
	}

	return p, nil
}

func randState(size int) string {
	b := make([]byte, size)

	// rand.Read never returns an error
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
