package oauth

import (
	"fmt"
	"net/http"
)

// HTTPEnv implements the Env interface using HTTP cookies
type HTTPEnv struct {
	scope string
	w     http.ResponseWriter
	r     *http.Request
}

// NewHTTPEnv creates a new HTTPEnv instance
func NewHTTPEnv(scope string, w http.ResponseWriter, r *http.Request) *HTTPEnv {
	return &HTTPEnv{scope: scope, w: w, r: r}
}

func (e *HTTPEnv) Save(key, val string) error {
	http.SetCookie(e.w, &http.Cookie{
		Name:     fmt.Sprintf("%s-%s", e.scope, key),
		Value:    val,
		HttpOnly: true,
	})
	return nil
}

func (e *HTTPEnv) Load(key string) (string, error) {
	c, err := e.r.Cookie(fmt.Sprintf("%s-%s", e.scope, key))
	if err != nil {
		return "", err
	}

	return c.Value, nil
}
