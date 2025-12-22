package oauth

import "net/http"

type HTTPEnv struct {
	w http.ResponseWriter
	r *http.Request
}

func NewHTTPEnv(w http.ResponseWriter, r *http.Request) *HTTPEnv {
	return &HTTPEnv{w: w, r: r}
}

func (e *HTTPEnv) Save(key, val string) error {
	http.SetCookie(e.w, &http.Cookie{
		Name:     key,
		Value:    val,
		HttpOnly: true,
	})
	return nil
}

func (e *HTTPEnv) Load(key string) (string, error) {
	c, err := e.r.Cookie(key)
	if err != nil {
		return "", err
	}

	return c.Value, nil
}
