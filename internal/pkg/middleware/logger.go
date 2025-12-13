package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gamma-omg/lexi-go/internal/pkg/router"
)

type httpStatusWriter struct {
	Status int
	inner  http.ResponseWriter
}

func (sw *httpStatusWriter) Header() http.Header {
	return sw.inner.Header()
}

func (sw *httpStatusWriter) WriteHeader(status int) {
	sw.Status = status
	sw.inner.WriteHeader(status)
}

func (sw *httpStatusWriter) Write(b []byte) (int, error) {
	return sw.inner.Write(b)
}

func Log() router.Middleware {
	return LogWith(slog.Default())
}

func LogWith(l *slog.Logger) router.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			statusWriter := &httpStatusWriter{inner: w}
			t := time.Now()

			next.ServeHTTP(statusWriter, r)
			l.Info("request received",
				"time", t,
				"method", r.Method,
				"url", r.URL.String(),
				"ip", r.RemoteAddr,
				"status", statusWriter.Status,
				"agent", r.UserAgent())
		})
	}
}
