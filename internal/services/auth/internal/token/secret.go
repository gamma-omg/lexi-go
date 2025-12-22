package token

type secretProvider interface {
	Get() []byte
}

type SecretString struct {
	secret []byte
}

func NewSecretString(secret string) *SecretString {
	return &SecretString{
		secret: []byte(secret),
	}
}

func (s *SecretString) Get() []byte {
	return s.secret
}
