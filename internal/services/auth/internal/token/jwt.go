package token

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
)

type JwtIssuer struct {
	secret    secretProvider
	algorithm string
	issuer    string
	ttl       time.Duration
}

type JwtConfig struct {
	Secret    secretProvider
	Algorithm string
	Issuer    string
	TTL       time.Duration
}

type jwtClaims struct {
	jwt.StandardClaims
	Type     Type   `json:"typ"`
	Email    string `json:"email"`
	Provider string `json:"provider"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
}

func NewJWTIssuer(cfg JwtConfig) *JwtIssuer {
	return &JwtIssuer{
		secret:    cfg.Secret,
		algorithm: cfg.Algorithm,
		issuer:    cfg.Issuer,
		ttl:       cfg.TTL,
	}
}

func (ti *JwtIssuer) Issue(claims UserClaims) (string, error) {
	tk, err := jwt.NewWithClaims(jwt.GetSigningMethod(ti.algorithm), jwtClaims{
		StandardClaims: jwt.StandardClaims{
			Id:        claims.ID,
			Issuer:    ti.issuer,
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(ti.ttl).Unix(),
		},
		Type:     claims.Type,
		Email:    claims.Email,
		Provider: claims.Provider,
		Name:     claims.Name,
		Picture:  claims.Picture,
	}).SignedString(ti.secret.Get())

	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return tk, nil
}

func (ti *JwtIssuer) Validate(tokenStr string) (UserClaims, error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != ti.algorithm {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return ti.secret.Get(), nil
	})
	if err != nil {
		return UserClaims{}, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := tok.Claims.(*jwtClaims)
	if !ok || !tok.Valid {
		return UserClaims{}, fmt.Errorf("invalid token claims")
	}

	return UserClaims{
		Type:     claims.Type,
		ID:       claims.Id,
		Email:    claims.Email,
		Provider: claims.Provider,
		Name:     claims.Name,
		Picture:  claims.Picture,
	}, nil
}
