package token

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
)

type Type string

const (
	TypeAccess  Type = "access"
	TypeRefresh Type = "refresh"
)

type UserClaims struct {
	jwt.StandardClaims
	Email    string `json:"email"`
	Provider string `json:"provider"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
	Type     Type   `json:"typ"`
}

type TokenIssuer struct {
	secret    secretProvider
	algorithm string
	issuer    string
	ttl       time.Duration
}

type TokenIssuerConfig struct {
	Secret    secretProvider
	Algorithm string
	Issuer    string
	TTL       time.Duration
}

func NewTokenIssuer(cfg TokenIssuerConfig) *TokenIssuer {
	return &TokenIssuer{
		secret:    cfg.Secret,
		algorithm: cfg.Algorithm,
		issuer:    cfg.Issuer,
		ttl:       cfg.TTL,
	}
}

func (ti *TokenIssuer) Issue(claims UserClaims) (string, error) {
	claims.Issuer = ti.issuer
	claims.Type = TypeAccess
	claims.IssuedAt = time.Now().Unix()
	claims.ExpiresAt = time.Now().Add(ti.ttl).Unix()

	tk, err := jwt.NewWithClaims(jwt.GetSigningMethod(ti.algorithm), claims).SignedString(ti.secret.Get())
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return tk, nil
}
