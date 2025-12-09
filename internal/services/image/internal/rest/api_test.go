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
	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"
)

type mockImageService struct {
	UploadFunc func(img io.Reader) (*url.URL, error)
}

func (m *mockImageService) Upload(img io.Reader) (*url.URL, error) {
	return m.UploadFunc(img)
}

func TestPOSTUpload(t *testing.T) {
	mux := http.NewServeMux()
	srv := &mockImageService{
		UploadFunc: func(img io.Reader) (*url.URL, error) {
			return &url.URL{Scheme: "https", Host: "images.example.com", Path: "/image123.jpg"}, nil
		},
	}
	api := NewAPI(srv, 10<<20, t.TempDir())
	api.Register(mux)

	rec := testutil.SendFile(t, mux, "POST", "/upload", testutil.TestFile{
		Name:      "test.jpg",
		FieldName: "image",
		Content:   strings.NewReader("test image content"),
	})

	assert.Equal(t, http.StatusCreated, rec.Code)

	resp := testutil.ParseResponse[uploadImageResponse](t, rec)
	assert.Equal(t, resp.ImageURL, "https://images.example.com/image123.jpg")
}

func TestGETImage(t *testing.T) {
	mux := http.NewServeMux()
	root := t.TempDir()
	api := NewAPI(&mockImageService{}, 10<<20, root)
	api.Register(mux)

	err := os.WriteFile(filepath.Join(root, "test.jpg"), []byte("test image content"), 0644)
	require.NoError(t, err)

	req, err := http.NewRequest("GET", "/image/test.jpg", nil)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, rec.Code, http.StatusOK)
	assert.Equal(t, rec.Body.String(), "test image content")
}
