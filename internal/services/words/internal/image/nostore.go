package image

import (
	"errors"
	"io"
	"net/url"
)

type NoStore struct{}

func (s *NoStore) SaveImage(img io.Reader) (*url.URL, error) {
	return nil, errors.New("image store is not supported")
}
