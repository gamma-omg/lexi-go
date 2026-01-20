package oauth

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPEnv(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/save", func(w http.ResponseWriter, r *http.Request) {
		env := NewHTTPEnv("test_scope", w, r)
		require.NoError(t, env.Save("test_key", "test_val"))
	})

	mux.HandleFunc("/load", func(w http.ResponseWriter, r *http.Request) {
		env := NewHTTPEnv("test_scope", w, r)
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
		env := NewHTTPEnv("test_scope", w, r)
		_, err := env.Load("non_existent_key")
		require.Error(t, err)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := &http.Client{}

	_, err := client.Get(fmt.Sprintf("%s/load", srv.URL))
	require.NoError(t, err)
}

func TestHTTPEnv_Scope(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/save", func(w http.ResponseWriter, r *http.Request) {
		e1 := NewHTTPEnv("s1", w, r)
		e2 := NewHTTPEnv("s2", w, r)

		require.NoError(t, e1.Save("val", "v1"))
		require.NoError(t, e2.Save("val", "v2"))
	})
	mux.HandleFunc("/load", func(w http.ResponseWriter, r *http.Request) {
		e1 := NewHTTPEnv("s1", w, r)
		e2 := NewHTTPEnv("s2", w, r)

		v1, err1 := e1.Load("val")
		v2, err2 := e2.Load("val")
		require.NoError(t, err1)
		require.NoError(t, err2)

		assert.Equal(t, "v1", v1)
		assert.Equal(t, "v2", v2)
	})

	srv := httptest.NewServer(mux)
	defer srv.Client()

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	client := http.Client{Jar: jar}

	_, err = client.Get(fmt.Sprintf("%s/save", srv.URL))
	require.NoError(t, err)

	_, err = client.Get(fmt.Sprintf("%s/load", srv.URL))
	require.NoError(t, err)
}
