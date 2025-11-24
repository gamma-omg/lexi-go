package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gamma-omg/lexi-go/internal/pkg/router"
)

func Recover() router.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					slog.Error("internal server error",
						"error", err,
						"method", r.Method,
						"url", r.URL.String(),
						"remote_addr", r.RemoteAddr,
						"stack_trace", string(debug.Stack()),
					)

					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
