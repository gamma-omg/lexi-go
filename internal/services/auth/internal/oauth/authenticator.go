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

// User represents an authenticated user from an identity provider
type User struct {
	ID            string
	Email         string
	EmailVerified bool
	Name          string
	Picture       string
	Nonce         string
}

// VerifiedEmail returns the user's email if it has been verified; otherwise, it returns an empty string
func (u *User) VerifiedEmail() string {
	if u.EmailVerified {
		return u.Email
	}
	return ""
}

// Env represents a storage mechanism for saving and loading state values
type Env interface {
	Save(key, val string) error
	Load(key string) (string, error)
}

// identityProvider defines the interface that each OAuth identity provider must implement
type identityProvider interface {
	LoginURL(state string, nonce string) (string, error)
	Exchange(ctx context.Context, code string) (User, error)
}

// Authenticator manages multiple OAuth identity providers and handles the authentication flow
type Authenticator struct {
	providers map[string]identityProvider
	mu        sync.RWMutex
}

// NewAuthenticator creates a new Authenticator instance
func NewAuthenticator() *Authenticator {
	return &Authenticator{
		providers: make(map[string]identityProvider),
	}
}

// Use registers a new identity provider with the given name
func (a *Authenticator) Use(name string, p identityProvider) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, ok := a.providers[name]; ok {
		return ErrProviderConflict
	}

	a.providers[name] = p
	return nil
}

// LoginURL generates a login URL for the specified provider and saves the state in the provided environment
func (a *Authenticator) LoginURL(env Env, provider string, state, nonce string) (string, error) {
	p, err := a.getProvider(provider)
	if err != nil {
		return "", fmt.Errorf("get provider: %w", err)
	}

	if err = env.Save("state", state); err != nil {
		return "", fmt.Errorf("save state: %w", err)
	}
	if err = env.Save("nonce", nonce); err != nil {
		return "", fmt.Errorf("save nonce: %w", err)
	}

	url, err := p.LoginURL(state, nonce)
	if err != nil {
		return "", fmt.Errorf("get login url: %w", err)
	}

	return url, nil
}

// Exchange exchanges the authorization code for user information after validating the state
func (a *Authenticator) Exchange(ctx context.Context, env Env, provider, code, state string) (User, error) {
	p, err := a.getProvider(provider)
	if err != nil {
		return User{}, fmt.Errorf("get provider: %w", err)
	}

	savedState, err := env.Load("state")
	if err != nil {
		return User{}, fmt.Errorf("load state: %w", err)
	}

	if savedState != state {
		return User{}, ErrAuthFailed
	}

	usr, err := p.Exchange(ctx, code)
	if err != nil {
		var rerr *oauth2.RetrieveError
		if errors.As(err, &rerr) && rerr.Response != nil {
			switch rerr.Response.StatusCode {
			case http.StatusBadRequest, http.StatusUnauthorized:
				return User{}, ErrAuthFailed
			}
		}

		return User{}, fmt.Errorf("exchange: %w", err)
	}

	savedNonce, err := env.Load("nonce")
	if err != nil {
		return User{}, fmt.Errorf("load nonce: %w", err)
	}

	if usr.Nonce == "" || usr.Nonce != savedNonce {
		return User{}, ErrAuthFailed
	}

	return usr, nil
}

// getProvider retrieves the identity provider by name
func (a *Authenticator) getProvider(name string) (identityProvider, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	p, ok := a.providers[name]
	if !ok {
		return nil, ErrProviderNotFound
	}

	return p, nil
}

// randString generates a random state string of the specified size
func randString(size int) string {
	b := make([]byte, size)

	// rand.Read never returns an error
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
