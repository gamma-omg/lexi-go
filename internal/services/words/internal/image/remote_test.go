package image

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoteSaveImage(t *testing.T) {
	srv := http.Server{
		Addr: ":9999",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			enc := json.NewEncoder(w)
			_ = enc.Encode(saveImageResponse{
				ImageURL: "http://localhost:9999/images/test.jpg",
			})
		}),
	}

	go srv.ListenAndServe()

	s := NewRemoteStore("http://localhost:9999/", "image", "test.jpg")

	imgURL, err := s.SaveImage(t.Context(), strings.NewReader("test image content"))
	require.NoError(t, err)

	require.Equal(t, "http://localhost:9999/images/test.jpg", imgURL.String())
}

func TestRemoteSaveImage_BadResponse(t *testing.T) {
	srv := http.Server{
		Addr: ":9998",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}),
	}

	go srv.ListenAndServe()

	s := NewRemoteStore("http://localhost:9998/", "image", "test.jpg")

	_, err := s.SaveImage(t.Context(), strings.NewReader("test image content"))
	require.Error(t, err)
}
