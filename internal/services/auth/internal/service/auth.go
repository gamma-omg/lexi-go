package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"

	"github.com/gamma-omg/lexi-go/internal/pkg/serr"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/oauth"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/store"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/token"
)

// TokenPair represents a pair of access and refresh tokens
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// tokenIssuer defines the interface for issuing and validating tokens
type tokenIssuer interface {
	Issue(claims token.UserClaims) (string, error)
	Validate(token string) (token.UserClaims, error)
}

// authenticator defines the interface for OAuth authentication flow management
type authenticator interface {
	LoginURL(env oauth.Env, providerName string, state, nonce string) (string, error)
	Exchange(ctx context.Context, env oauth.Env, providerName, code string, state string) (oauth.User, error)
}

// tokenExchanger defines the interface for storing and redeeming token exchange codes
type oneTimeCodeProvider interface {
	CreateCode(ctx context.Context, ts TokenPair) (string, error)
	RedeemCode(ctx context.Context, code string) (TokenPair, error)
}

// Auth handles OAuth authentication and token management
type Auth struct {
	auth         authenticator
	store        store.Store
	accessToken  tokenIssuer
	refreshToken tokenIssuer
	otc          oneTimeCodeProvider
}

// AuthOption defines a functional option for configuring the Auth service
type AuthOption func(*Auth) *Auth

func WithAuthenticator(a authenticator) AuthOption {
	return func(s *Auth) *Auth {
		s.auth = a
		return s
	}
}

func WithStore(st store.Store) AuthOption {
	return func(s *Auth) *Auth {
		s.store = st
		return s
	}
}

func WithAccessToken(iss tokenIssuer) AuthOption {
	return func(s *Auth) *Auth {
		s.accessToken = iss
		return s
	}
}

func WithRefreshToken(iss tokenIssuer) AuthOption {
	return func(s *Auth) *Auth {
		s.refreshToken = iss
		return s
	}
}

func WithOTC(ex oneTimeCodeProvider) AuthOption {
	return func(s *Auth) *Auth {
		s.otc = ex
		return s
	}
}

// NewAuth creates a new Auth service with the provided options
func NewAuth(opts ...AuthOption) *Auth {
	s := &Auth{}
	for _, opt := range opts {
		s = opt(s)
	}

	if s.auth == nil {
		panic("oauth authenticator is required")
	}

	if s.store == nil {
		panic("store is required")
	}

	if s.accessToken == nil {
		panic("access token issuer is required")
	}

	if s.refreshToken == nil {
		panic("refresh token issuer is required")
	}

	if s.otc == nil {
		panic("token exchanger is required")
	}

	return s
}

type LoginRequest struct {
	Provider    string
	RedirectURL string
}

// LoginURL generates a login URL for the specified provider
func (s *Auth) LoginURL(env oauth.Env, r LoginRequest) (string, error) {
	state := randString(32)
	nonce := randString(32)

	err := env.Save("redirect_url", r.RedirectURL)
	if err != nil {
		return "", fmt.Errorf("save redirect url: %w", err)
	}

	url, err := s.auth.LoginURL(env, r.Provider, state, nonce)
	if err != nil {
		if errors.Is(err, oauth.ErrProviderNotFound) {
			sErr := serr.NewServiceError(err, http.StatusNotFound, "oauth provider not found")
			sErr.Env["provider"] = r.Provider
			return "", sErr
		}

		return "", fmt.Errorf("login url: %w", err)
	}

	return url, nil
}

type AuthCallbackRequest struct {
	Provider string
	Code     string
	State    string
}

type AuthCallbackResponse struct {
	UID          string
	Name         string
	Email        string
	Picture      string
	AccessToken  string
	RefreshToken string
	RedirectURL  string
	OTC          string
}

// AuthCallback handles the OAuth callback, exchanges the code for user info, and issues tokens
func (s *Auth) AuthCallback(ctx context.Context, env oauth.Env, r AuthCallbackRequest) (resp AuthCallbackResponse, err error) {
	usr, err := s.auth.Exchange(ctx, env, r.Provider, r.Code, r.State)
	if err != nil {
		if errors.Is(err, oauth.ErrProviderNotFound) {
			sErr := serr.NewServiceError(err, http.StatusNotFound, "provider not found")
			sErr.Env["provider"] = r.Provider
			err = sErr
			return
		}

		if errors.Is(err, oauth.ErrAuthFailed) {
			sErr := serr.NewServiceError(err, http.StatusUnauthorized, "authentication failed")
			sErr.Env["provider"] = r.Provider
			err = sErr
			return
		}

		err = fmt.Errorf("exchange: %w", err)
		return
	}

	id, err := s.getOrCreateUser(ctx, r.Provider, usr)
	if err != nil {
		err = fmt.Errorf("get or create user: %w", err)
		return
	}

	at, atErr := s.accessToken.Issue(token.UserClaims{
		ID:       id.User.UID,
		Email:    id.Email,
		Provider: id.Provider,
		Name:     id.Name,
		Picture:  id.Picture,
	})
	if atErr != nil {
		err = fmt.Errorf("issue access token: %w", atErr)
		return
	}

	rt, rtErr := s.refreshToken.Issue(token.UserClaims{
		ID:   id.User.UID,
		Type: token.TypeRefresh,
	})
	if rtErr != nil {
		err = fmt.Errorf("issue refresh token: %w", rtErr)
		return
	}

	code, err := s.otc.CreateCode(ctx, TokenPair{
		AccessToken:  at,
		RefreshToken: rt,
	})
	if err != nil {
		err = fmt.Errorf("create exchange code: %w", err)
		return
	}

	redirectURL, err := env.Load("redirect_url")
	if err != nil {
		err = fmt.Errorf("load redirect url: %w", err)
		return
	}

	return AuthCallbackResponse{
		UID:          id.User.UID,
		Name:         id.Name,
		Email:        id.Email,
		Picture:      id.Picture,
		AccessToken:  at,
		RefreshToken: rt,
		RedirectURL:  fmt.Sprintf("%s?otc=%s", redirectURL, code),
		OTC:          code,
	}, nil
}

// Refresh issues a new access token using a valid refresh token
func (a *Auth) Refresh(ctx context.Context, refreshToken string) (string, error) {
	claims, err := a.refreshToken.Validate(refreshToken)
	if err != nil {
		return "", serr.NewServiceError(err, http.StatusUnauthorized, "invalid refresh token")
	}

	id, err := a.store.GetUserIdentity(ctx, store.GetUserIdentityRequest{
		UID:      claims.ID,
		Provider: claims.Provider,
	})
	if err != nil {
		return "", serr.NewServiceError(err, http.StatusUnauthorized, "invalid user identity")
	}

	at, atErr := a.accessToken.Issue(token.UserClaims{
		ID:       id.User.UID,
		Email:    id.Email,
		Provider: id.Provider,
		Name:     id.Name,
		Picture:  id.Picture,
	})
	if atErr != nil {
		return "", fmt.Errorf("issue access token: %w", atErr)
	}

	return at, nil
}

// RedeemCode redeems a code for a token set
func (a *Auth) RedeemCode(ctx context.Context, code string) (TokenPair, error) {
	ts, err := a.otc.RedeemCode(ctx, code)
	if err != nil {
		sErr := serr.NewServiceError(err, http.StatusInternalServerError, "failed to redeem code")
		sErr.Env["code"] = code
		return TokenPair{}, sErr
	}

	return ts, nil
}

// getOrCreateUser retrieves an existing user identity or creates a new user and identity
func (s *Auth) getOrCreateUser(ctx context.Context, provider string, usr oauth.User) (store.Identity, error) {
	id, err := s.store.GetIdentity(ctx, store.GetIdentityRequest{
		ID:       usr.ID,
		Provider: provider,
	})
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			return store.Identity{}, fmt.Errorf("get user identity: %w", err)
		}

		err = s.store.WithTx(ctx, func(tx store.Store) error {
			userID, err := s.store.CreateUser(ctx)
			if err != nil {
				return fmt.Errorf("create user: %w", err)
			}

			_, err = s.store.CreateUserIdentity(ctx, store.CreateUserIdentityRequest{
				UserID:   userID,
				ID:       usr.ID,
				Provider: provider,
				Email:    usr.VerifiedEmail(),
				Name:     usr.Name,
				Picture:  usr.Picture,
			})
			if err != nil {
				return fmt.Errorf("create user identity: %w", err)
			}

			id, err = tx.GetIdentity(ctx, store.GetIdentityRequest{
				ID:       usr.ID,
				Provider: provider,
			})
			if err != nil {
				return fmt.Errorf("get user identity after create: %w", err)
			}

			return nil
		})

		if err != nil {
			return store.Identity{}, fmt.Errorf("with tx: %w", err)
		}

	}

	return id, nil
}

// randString generates a random state string of the specified size
func randString(size int) string {
	b := make([]byte, size)

	// rand.Read never returns an error
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
