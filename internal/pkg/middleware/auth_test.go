package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gamma-omg/lexi-go/internal/pkg/router"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth_WithoutToken(t *testing.T) {
	r := router.New()
	r.Use(Auth([]byte("test-api-key")))

	r.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_InvalidToken(t *testing.T) {
	r := router.New()
	r.Use(Auth([]byte("test-api-key")))

	r.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("X-API-Key", "invalid-token")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_ValidToken(t *testing.T) {
	key := []byte("test-api-key")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{Subject: "user-123"})
	signed, err := token.SignedString([]byte(key))
	require.NoError(t, err)

	r := router.New()
	r.Use(Auth(key))

	r.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		uid := UserIDFromContext(r.Context())
		fmt.Fprintln(w, uid)
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", signed)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "user-123\n", rec.Body.String())
}

func TestAuth_ValidToken_NoUser(t *testing.T) {
	key := []byte("test-api-key")
	token := jwt.New(jwt.SigningMethodHS256)
	signed, err := token.SignedString([]byte(key))
	require.NoError(t, err)

	r := router.New()
	r.Use(Auth(key))

	r.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", signed)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
