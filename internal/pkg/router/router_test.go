package router_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gamma-omg/lexi-go/internal/pkg/router"
	"github.com/stretchr/testify/assert"
)

func TestHandle(t *testing.T) {
	tbl := []struct {
		method       string
		path         string
		requestBody  string
		responseBody string
		status       int
	}{
		{"GET", "/hello", "Hello, world!", "ok", http.StatusOK},
		{"GET", "/notfound", "Not Found", "", http.StatusNotFound},
		{"POST", "/hello", "Method Not Allowed", "Not Allowed", http.StatusMethodNotAllowed},
		{"PUT", "/hello", "", "", http.StatusMethodNotAllowed},
		{"DELETE", "/hello", "", "forbidden", http.StatusForbidden},
		{"GET", "/", "", "root hit", http.StatusOK},
		{"GET", "/long/path", "long", "", http.StatusOK},
		{"POST", "/long/path/", "", "long", http.StatusOK},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			r := router.New()

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.requestBody))

			r.Handle(c.path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(c.status)
				fmt.Fprint(w, c.responseBody)
			}))
			r.ServeHTTP(rec, req)

			assert.Equal(t, c.status, rec.Code)
			assert.Equal(t, c.responseBody, rec.Body.String())
		})
	}
}

func TestHandleFunc(t *testing.T) {
	tbl := []struct {
		method       string
		path         string
		requestBody  string
		responseBody string
		status       int
	}{
		{"GET", "/hello", "Hello, world!", "ok", http.StatusOK},
		{"GET", "/notfound", "Not Found", "", http.StatusNotFound},
		{"POST", "/hello", "Method Not Allowed", "Not Allowed", http.StatusMethodNotAllowed},
		{"PUT", "/hello", "", "", http.StatusMethodNotAllowed},
		{"DELETE", "/hello", "", "forbidden", http.StatusForbidden},
		{"GET", "/", "", "root hit", http.StatusOK},
		{"GET", "/long/path", "long", "", http.StatusOK},
		{"POST", "/long/path/", "", "long", http.StatusOK},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			r := router.New()

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.requestBody))

			r.HandleFunc(c.path, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(c.status)
				fmt.Fprint(w, c.responseBody)
			})

			r.ServeHTTP(rec, req)

			assert.Equal(t, c.status, rec.Code)
			assert.Equal(t, c.responseBody, rec.Body.String())
		})
	}
}

func TestSubRouter(t *testing.T) {
	tbl := []struct {
		method       string
		mountPoint   string
		relativePath string
		path         string
		responseBody string
		status       int
	}{
		{"GET", "/api", "/hello", "/api/hello", "hello from subrouter", http.StatusOK},
		{"POST", "v1", "/hello/", "/v1/hello/world", "hello from subrouter", http.StatusForbidden},
		{"POST", "/long/prefix", "hello", "/long/prefix/hello", "", http.StatusConflict},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			r := router.New()
			sub := r.SubRouter(c.mountPoint)

			sub.HandleFunc(c.relativePath, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(c.status)
				fmt.Fprint(w, c.responseBody)
			})

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(c.method, c.path, nil)

			r.ServeHTTP(rec, req)

			assert.Equal(t, c.status, rec.Code)
			assert.Equal(t, c.responseBody, rec.Body.String())
		})
	}
}

func TestSubRoute_PanicsWhenEmpty(t *testing.T) {
	r := router.New()
	assert.Panics(t, func() {
		r.SubRouter("")
	})
}

func TestMiddleware(t *testing.T) {
	r := router.New()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom-Header", "value-123")
			next.ServeHTTP(w, r)
		})
	})

	r.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "testing middleware")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "testing middleware", rec.Body.String())
	assert.Equal(t, "value-123", rec.Header().Get("X-Custom-Header"))
}

func TestMiddleware_Order(t *testing.T) {
	r := router.New()

	callOrder := make(chan int, 2)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callOrder <- 1
			next.ServeHTTP(w, r)
		})
	})
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callOrder <- 2
			next.ServeHTTP(w, r)
		})
	})

	r.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "testing middleware order")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "testing middleware order", rec.Body.String())

	close(callOrder)
	assert.Equal(t, 1, <-callOrder)
	assert.Equal(t, 2, <-callOrder)
}
