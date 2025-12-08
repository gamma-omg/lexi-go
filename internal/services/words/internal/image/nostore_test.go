package image

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNoStore_SaveImage(t *testing.T) {
	s := NoStore{}

	_, err := s.SaveImage(strings.NewReader("test image content"))
	require.Error(t, err)
}
