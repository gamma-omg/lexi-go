package rest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gamma-omg/lexi-go/internal/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockImageService struct {
	UploadFunc func(img io.Reader) (*url.URL, error)
}

func (m *mockImageService) Upload(img io.Reader) (*url.URL, error) {
	return m.UploadFunc(img)
}

func TestPOSTUpload(t *testing.T) {
	srv := &mockImageService{
		UploadFunc: func(img io.Reader) (*url.URL, error) {
			return &url.URL{Scheme: "https", Host: "images.example.com", Path: "/image123.jpg"}, nil
		},
	}
	api := NewAPI(srv, 10<<20, t.TempDir())

	rec := testutil.SendFile(t, api, "POST", "/upload", testutil.TestFile{
		Name:      "test.jpg",
		FieldName: "image",
		Content:   strings.NewReader("test image content"),
	})

	assert.Equal(t, http.StatusCreated, rec.Code)

	resp := testutil.ParseResponse[uploadImageResponse](t, rec)
	assert.Equal(t, "https://images.example.com/image123.jpg", resp.ImageURL)
}

func TestGETImage(t *testing.T) {
	root := t.TempDir()
	api := NewAPI(&mockImageService{}, 10<<20, root)

	err := os.WriteFile(filepath.Join(root, "test.jpg"), []byte("test image content"), 0644)
	require.NoError(t, err)

	req, err := http.NewRequest("GET", "/image/test.jpg", nil)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "test image content", rec.Body.String())
}
