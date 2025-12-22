package token

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecretString(t *testing.T) {
	secret := NewSecretString("hello world")
	assert.Equal(t, []byte("hello world"), secret.Get())
}
