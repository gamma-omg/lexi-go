package provider

import (
	"context"
	"crypto/sha1"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/oauth"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

const (
	googleScopeEmail   string = "email"
	googleScopeProfile string = "profile"
)

// Google implements the identityProvider interface for Google OAuth
type Google struct {
	cfg      *oauth2.Config
	verifier *oidc.IDTokenVerifier
}

// GoogleConfig holds the configuration for the Google OAuth provider
type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type userClaims struct {
	Sub      string `json:"sub,omitempty"`
	Email    string `json:"email,omitempty"`
	Verified bool   `json:"email_verified,omitempty"`
	Name     string `json:"name,omitempty"`
	Picture  string `json:"picture,omitempty"`
}

// NewGoogle creates a new Google OAuth provider with the given configuration
func NewGoogle(ctx context.Context, google GoogleConfig) (*Google, error) {
	p, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return nil, fmt.Errorf("new oidc provider: %w", err)
	}

	return &Google{
		cfg: &oauth2.Config{
			ClientID:     google.ClientID,
			ClientSecret: google.ClientSecret,
			RedirectURL:  google.RedirectURL,
			Scopes:       []string{oidc.ScopeOpenID, googleScopeProfile, googleScopeEmail},
			Endpoint:     endpoints.Google,
		},
		verifier: p.Verifier(&oidc.Config{ClientID: google.ClientID}),
	}, nil
}

// LoginURL generates the Google OAuth login URL with the given state
func (g *Google) LoginURL(state, nonce string) (string, error) {
	return g.cfg.AuthCodeURL(state, oidc.Nonce(nonce)), nil
}

// Exchange exchanges the authorization code for an OAuth user
func (g *Google) Exchange(ctx context.Context, code string) (oauth.User, error) {
	tok, err := g.cfg.Exchange(ctx, code)
	if err != nil {
		return oauth.User{}, err
	}

	raw := tok.Extra("id_token").(string)
	idTok, err := g.verifier.Verify(ctx, raw)
	if err != nil {
		return oauth.User{}, fmt.Errorf("verify id token: %w", err)
	}

	var usr userClaims
	if err := idTok.Claims(&usr); err != nil {
		return oauth.User{}, fmt.Errorf("read claims: %w", err)
	}

	return oauth.User{
		Nonce:         idTok.Nonce,
		ID:            usr.Sub,
		Email:         usr.Email,
		EmailVerified: usr.Verified,
		Picture:       usr.Picture,
		Name:          nameOrDefault(usr.Name, defaultName(usr)),
	}, nil
}

// nameOrDefault returns the user's name if it's not empty; otherwise, it returns the default name
func nameOrDefault(name, def string) string {
	if name != "" {
		return name
	}
	return def
}

// defaultName generates a default name based on the user's subject identifier
func defaultName(usr userClaims) string {
	id := sha1.New().Sum([]byte(usr.Sub))[:8]
	return fmt.Sprintf("google_%x", id)
}
