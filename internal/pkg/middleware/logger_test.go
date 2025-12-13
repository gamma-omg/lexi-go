package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gamma-omg/lexi-go/internal/pkg/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type logEntry struct {
	Msg    string `json:"msg"`
	Level  string `json:"level"`
	URL    string `json:"url"`
	Agent  string `json:"agent"`
	Status int    `json:"status"`
	IP     string `json:"ip"`
	Method string `json:"method"`
}

func TestLogWith(t *testing.T) {
	b := bytes.Buffer{}
	l := slog.New(slog.NewJSONHandler(&b, &slog.HandlerOptions{}))
	m := middleware.LogWith(l)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	req := httptest.NewRequest("GET", "/test?id=123", nil)
	req.RemoteAddr = "1.2.3.4"
	req.Header.Set("User-Agent", "test-runner")

	rec := httptest.NewRecorder()
	m(next).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTeapot, rec.Code)

	var e logEntry
	err := json.Unmarshal(b.Bytes(), &e)
	require.NoError(t, err)

	assert.Equal(t, "request received", e.Msg)
	assert.Equal(t, "INFO", e.Level)
	assert.Equal(t, "/test?id=123", e.URL)
	assert.Equal(t, "test-runner", e.Agent)
	assert.Equal(t, 418, e.Status)
	assert.Equal(t, "1.2.3.4", e.IP)
	assert.Equal(t, "GET", e.Method)
}
