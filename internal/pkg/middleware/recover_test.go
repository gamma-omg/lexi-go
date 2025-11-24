package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gamma-omg/lexi-go/internal/pkg/router"
	"github.com/stretchr/testify/assert"
)

func TestRecover_Panic(t *testing.T) {
	slog.SetDefault(slog.New(slog.DiscardHandler))

	r := router.New()
	r.Use(Recover())

	r.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/panic", nil)

	assert.NotPanics(t, func() {
		r.ServeHTTP(rec, req)
	})
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRecover_NoPanic(t *testing.T) {
	r := router.New()
	r.Use(Recover())

	r.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ok", nil)

	assert.NotPanics(t, func() {
		r.ServeHTTP(rec, req)
	})
	assert.Equal(t, http.StatusOK, rec.Code)
}
