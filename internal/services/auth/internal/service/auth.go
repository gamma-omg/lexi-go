package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gamma-omg/lexi-go/internal/pkg/serr"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/oauth"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/store"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/token"
)

type tokenIssuer interface {
	Issue(claims token.UserClaims) (string, error)
	Validate(token string) (token.UserClaims, error)
}

type authenticator interface {
	LoginURL(env oauth.Env, providerName string) (string, error)
	Exchange(ctx context.Context, env oauth.Env, providerName, code, state string) (oauth.User, error)
}

type Auth struct {
	auth         authenticator
	store        store.Store
	accessToken  tokenIssuer
	refreshToken tokenIssuer
}

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

	return s
}

func (s *Auth) LoginURL(providerName string, env oauth.Env) (string, error) {
	url, err := s.auth.LoginURL(env, providerName)
	if err != nil {
		if errors.Is(err, oauth.ErrProviderNotFound) {
			sErr := serr.NewServiceError(err, http.StatusNotFound, "oauth provider not found")
			sErr.Env["provider"] = providerName
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
	AccessToken  string
	RefreshToken string
}

func (s *Auth) AuthCallback(ctx context.Context, env oauth.Env, r AuthCallbackRequest) (resp *AuthCallbackResponse, err error) {
	usr, err := s.auth.Exchange(ctx, env, r.Provider, r.Code, r.State)
	if err != nil {
		if errors.Is(err, oauth.ErrProviderNotFound) {
			sErr := serr.NewServiceError(err, http.StatusNotFound, "provider not found")
			sErr.Env["provider"] = r.Provider
			return nil, sErr
		}

		if errors.Is(err, oauth.ErrAuthFailed) {
			sErr := serr.NewServiceError(err, http.StatusUnauthorized, "authentication failed")
			sErr.Env["provider"] = r.Provider
			return nil, sErr
		}

		return nil, fmt.Errorf("exchange: %w", err)
	}

	id, err := s.getOrCreateUser(ctx, r.Provider, usr)
	if err != nil {
		return nil, fmt.Errorf("get or create user %w", err)
	}

	at, atErr := s.accessToken.Issue(token.UserClaims{
		ID:       id.User.UID,
		Email:    id.Email,
		Provider: id.Provider,
		Name:     id.Name,
		Picture:  id.Picture,
	})
	if atErr != nil {
		return nil, fmt.Errorf("issue access token: %w", atErr)
	}

	rt, rtErr := s.refreshToken.Issue(token.UserClaims{
		ID:   id.User.UID,
		Type: token.TypeRefresh,
	})
	if rtErr != nil {
		return nil, fmt.Errorf("issue refresh token: %w", rtErr)
	}

	resp = &AuthCallbackResponse{
		AccessToken:  at,
		RefreshToken: rt,
	}
	return
}

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
