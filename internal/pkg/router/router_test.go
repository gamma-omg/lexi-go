package router_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gamma-omg/lexi-go/internal/pkg/router"
	"github.com/stretchr/testify/assert"
)

func TestHandle(t *testing.T) {
	tbl := []struct {
		method     string
		mountPoint string
		url        string
		status     int
	}{
		{"GET", "/hello", "/hello", http.StatusOK},
		{"GET", "/notfound", "/notfound", http.StatusNotFound},
		{"POST", "/hello", "/hello", http.StatusConflict},
		{"PUT", "/hello", "/hello", http.StatusCreated},
		{"DELETE", "/hello", "/hello", http.StatusForbidden},
		{"GET", "/", "/", http.StatusOK},
		{"GET", "/long/path", "/long/path", http.StatusOK},
		{"POST", "/long/path/", "/long/path/child", http.StatusOK},
		{"POST", "POST /api/v1/", "/api/v1/method", http.StatusOK},
		{"POST", "GET /api/v1/", "/api/v1/method", http.StatusMethodNotAllowed},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			r := router.New()

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(c.method, c.url, http.NoBody)

			r.Handle(c.mountPoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(c.status)
			}))
			r.ServeHTTP(rec, req)

			assert.Equal(t, c.status, rec.Code)
		})
	}
}

func TestHandleFunc(t *testing.T) {
	tbl := []struct {
		method     string
		mountPoint string
		url        string
		status     int
	}{
		{"GET", "/hello", "/hello", http.StatusOK},
		{"GET", "/notfound", "/notfound", http.StatusNotFound},
		{"POST", "/hello", "/hello", http.StatusMethodNotAllowed},
		{"PUT", "/hello", "/hello", http.StatusMethodNotAllowed},
		{"DELETE", "/hello", "/hello", http.StatusForbidden},
		{"GET", "/", "/", http.StatusOK},
		{"GET", "/long/path", "/long/path", http.StatusOK},
		{"POST", "/long/path/", "/long/path/child", http.StatusOK},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			r := router.New()

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(c.method, c.url, http.NoBody)

			r.HandleFunc(c.mountPoint, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(c.status)
			})

			r.ServeHTTP(rec, req)

			assert.Equal(t, c.status, rec.Code)
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
		{"POST", "/v1", "/hello/", "/v1/hello/world", "hello from subrouter", http.StatusForbidden},
		{"POST", "/long/prefix", "/hello", "/long/prefix/hello", "", http.StatusConflict},
		{"GET", "/api/v1", "/", "/api/v1/method", "", http.StatusOK},
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
