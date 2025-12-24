package token

// secretProvider defines the interface for providing secret keys
type secretProvider interface {
	Get() []byte
}

// SecretString implements the secretProvider interface using a string
type SecretString struct {
	secret []byte
}

// NewSecretString creates a new SecretString instance
func NewSecretString(secret string) *SecretString {
	return &SecretString{
		secret: []byte(secret),
	}
}

// Get returns the secret as a byte slice
func (s *SecretString) Get() []byte {
	return s.secret
}
