package token

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTIssuer(t *testing.T) {
	secret := NewSecretString("test_secret")
	issuer := NewJWTIssuer(JwtConfig{
		Issuer:    "test-issuer",
		Secret:    secret,
		Algorithm: jwt.SigningMethodHS256.Name,
		TTL:       time.Hour,
	})

	claims := UserClaims{
		Type:     TypeAccess,
		ID:       "user-123",
		Email:    "test@example.com",
		Provider: "google",
		Name:     "Test User",
		Picture:  "http://example.com/pic.jpg",
	}

	tokenStr, err := issuer.Issue(claims)
	require.NoError(t, err)
	require.NotEmpty(t, tokenStr)

	claims, err = issuer.Validate(tokenStr)
	require.NoError(t, err)

	assert.Equal(t, TypeAccess, claims.Type)
	assert.Equal(t, "user-123", claims.ID)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.Equal(t, "google", claims.Provider)
	assert.Equal(t, "Test User", claims.Name)
	assert.Equal(t, "http://example.com/pic.jpg", claims.Picture)
}
