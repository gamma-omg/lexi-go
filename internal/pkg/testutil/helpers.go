package testutil

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type TestFile struct {
	Name      string
	FieldName string
	Content   io.Reader
}

func SendFile(t testing.TB, h http.Handler, method, path string, file TestFile) *httptest.ResponseRecorder {
	t.Helper()

	var bodyRW strings.Builder
	writer := multipart.NewWriter(&bodyRW)

	part, err := writer.CreateFormFile(file.FieldName, file.Name)
	require.NoError(t, err)

	_, err = io.Copy(part, file.Content)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	req, err := http.NewRequest(method, path, strings.NewReader(bodyRW.String()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	return rec
}

func SendRequest(t testing.TB, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var bodyRW strings.Builder
	enc := json.NewEncoder(&bodyRW)
	err := enc.Encode(body)
	require.NoError(t, err)

	req, err := http.NewRequest(method, path, strings.NewReader(bodyRW.String()))
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	return rec
}

func ParseResponse[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()

	dec := json.NewDecoder(rec.Body)
	var resp T
	err := dec.Decode(&resp)
	require.NoError(t, err)

	return resp
}

func WaitFor(t testing.TB, ctx context.Context, interval time.Duration, condition func() bool) bool {
	t.Helper()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if condition() {
				return true
			}
		}
	}
}
