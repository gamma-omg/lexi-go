package service

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gamma-omg/lexi-go/internal/pkg/serr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpload(t *testing.T) {
	srv := NewImageService(ImageServiceConfig{
		ServeRoot: &url.URL{Scheme: "http", Host: "example.com"},
		Root:      t.TempDir(),
		MaxWidth:  100,
		MaxHeight: 100,
	})

	img := image.NewRGBA(image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: 90, Y: 90},
	})

	buf := &bytes.Buffer{}
	err := png.Encode(buf, img)
	require.NoError(t, err)

	imgURL, err := srv.Upload(buf)
	require.NoError(t, err)
	require.NotNil(t, imgURL)
}

func TestUpload_InvalidImage(t *testing.T) {
	srv := NewImageService(ImageServiceConfig{
		ServeRoot: &url.URL{Scheme: "http", Host: "example.com"},
		Root:      t.TempDir(),
		MaxWidth:  100,
		MaxHeight: 100,
	})

	img := image.NewRGBA(image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: 200, Y: 200},
	})

	buf := &bytes.Buffer{}
	err := png.Encode(buf, img)
	require.NoError(t, err)

	imgURL, err := srv.Upload(buf)

	var se *serr.ServiceError
	require.ErrorAs(t, err, &se)
	assert.Nil(t, imgURL)
	assert.Equal(t, se.StatusCode, http.StatusRequestEntityTooLarge)
}

func TestUpload_TooLarge(t *testing.T) {
	srv := NewImageService(ImageServiceConfig{
		ServeRoot: &url.URL{Scheme: "http", Host: "example.com"},
		Root:      t.TempDir(),
		MaxWidth:  100,
		MaxHeight: 100,
	})

	img := image.NewRGBA(image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: 50, Y: 50},
	})

	buf := &bytes.Buffer{}
	err := png.Encode(buf, img)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	rdr := http.MaxBytesReader(rec, io.NopCloser(buf), 10)

	imgURL, err := srv.Upload(rdr)

	var se *serr.ServiceError
	require.ErrorAs(t, err, &se)
	assert.Nil(t, imgURL)
	assert.Equal(t, se.StatusCode, http.StatusRequestEntityTooLarge)
}
