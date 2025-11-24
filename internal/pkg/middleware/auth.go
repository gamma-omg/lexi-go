package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gamma-omg/lexi-go/internal/pkg/router"
	"github.com/golang-jwt/jwt/v5"
)

type ctxKey struct{}

var userIDKey ctxKey

func Auth(key any) router.Middleware {
	return func(next http.Handler) http.Handler {
		return authMiddleware(next, key)
	}
}

func authMiddleware(next http.Handler, key any) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawToken := r.Header.Get("Authorization")
		if rawToken == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(rawToken, func(t *jwt.Token) (any, error) {
			return key, nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

		if err != nil {
			authError("failed to parse jwt", w, r, err)
			return
		}
		if !token.Valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			authError("invalid jwt claims type", w, r, nil)
			return
		}

		uid, ok := claims["sub"].(string)
		if uid == "" || !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func authError(msg string, w http.ResponseWriter, r *http.Request, err error) {
	slog.Error(msg,
		"error", err,
		"method", r.Method,
		"url", r.URL.String(),
		"remote_addr", r.RemoteAddr,
	)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

func UserIDFromContext(ctx context.Context) string {
	uid, _ := ctx.Value(userIDKey).(string)
	return uid
}
