package service

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/gamma-omg/lexi-go/internal/pkg/serr"
	"github.com/google/uuid"
)

// ImageService handles image uploads and validations.
type ImageService struct {
	serveRoot *url.URL
	root      string
	maxWidth  int
	maxHeight int
}

type ImageServiceConfig struct {
	ServeRoot *url.URL
	Root      string
	MaxWidth  int
	MaxHeight int
}

// NewImageService creates a new instance of ImageService with the given configuration.
func NewImageService(cfg ImageServiceConfig) *ImageService {
	return &ImageService{
		serveRoot: cfg.ServeRoot,
		root:      cfg.Root,
		maxWidth:  cfg.MaxWidth,
		maxHeight: cfg.MaxHeight,
	}
}

// Upload validates and saves the image, returning its accessible URL.
func (s *ImageService) Upload(img io.Reader) (*url.URL, error) {
	var buff bytes.Buffer
	tee := io.TeeReader(img, &buff)

	cfg, _, err := image.DecodeConfig(tee)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return nil, serr.NewServiceError(err, http.StatusRequestEntityTooLarge, "image size exceeded")
		}
		return nil, fmt.Errorf("decode image config: %w", err)
	}
	if cfg.Width > s.maxWidth || cfg.Height > s.maxHeight {
		return nil, serr.NewServiceError(err, http.StatusRequestEntityTooLarge, "image dimensions exceeded")
	}

	f, err := os.Create(filepath.Join(s.root, uuid.NewString()))
	if err != nil {
		return nil, fmt.Errorf("create image file: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, io.MultiReader(&buff, img))
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return nil, serr.NewServiceError(err, http.StatusRequestEntityTooLarge, "image size exceeded")
		}
		return nil, fmt.Errorf("save image file: %w", err)
	}

	relPath, err := filepath.Rel(s.root, f.Name())
	if err != nil {
		return nil, fmt.Errorf("get relative image path: %w", err)
	}

	return s.serveRoot.JoinPath(relPath), nil
}
