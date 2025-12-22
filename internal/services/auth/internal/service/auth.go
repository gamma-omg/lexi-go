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
}

type Auth struct {
	auth         *oauth.Authenticator
	store        store.Store
	accessToken  tokenIssuer
	refreshToken tokenIssuer
}

func (s *Auth) LoginURL(providerName string, env oauth.Env) (string, error) {
	url, err := s.auth.LoginURL(env, providerName)
	if err != nil {
		if errors.Is(err, oauth.ErrProviderNotFound) {
			return "", serr.NewServiceError(err, http.StatusNotFound, "oauth provider not found: %s", providerName)
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
			return nil, serr.NewServiceError(err, http.StatusNotFound, "provider not found: %s", err)
		}

		if errors.Is(err, oauth.ErrAuthFailed) {
			return nil, serr.NewServiceError(err, http.StatusUnauthorized, "authentication failed")
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

func (s *Auth) getOrCreateUser(ctx context.Context, provider string, usr oauth.User) (store.Identity, error) {
	id, err := s.store.GetUserIdentity(ctx, store.GetUserIdentityRequest{
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

			id, err = tx.GetUserIdentity(ctx, store.GetUserIdentityRequest{
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
