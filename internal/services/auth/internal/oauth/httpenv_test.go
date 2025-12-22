package oauth

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPEnv(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/save", func(w http.ResponseWriter, r *http.Request) {
		env := NewHTTPEnv(w, r)
		env.Save("test_key", "test_val")
	})

	mux.HandleFunc("/load", func(w http.ResponseWriter, r *http.Request) {
		env := NewHTTPEnv(w, r)
		val, err := env.Load("test_key")
		require.NoError(t, err)
		require.Equal(t, "test_val", val)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	client := &http.Client{Jar: jar}

	_, err = client.Get(fmt.Sprintf("%s/save", srv.URL))
	require.NoError(t, err)

	_, err = client.Get(fmt.Sprintf("%s/load", srv.URL))
	require.NoError(t, err)
}

func TestHTTPEnv_Load_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/load", func(w http.ResponseWriter, r *http.Request) {
		env := NewHTTPEnv(w, r)
		_, err := env.Load("non_existent_key")
		require.Error(t, err)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := &http.Client{}

	_, err := client.Get(fmt.Sprintf("%s/load", srv.URL))
	require.NoError(t, err)
}
