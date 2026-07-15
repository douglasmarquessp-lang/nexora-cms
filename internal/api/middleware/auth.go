package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"nexora/internal/modules/auth"
)

func RequireAuth(svc *auth.Service) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"authorization header required"}}`, http.StatusUnauthorized)
				return
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"invalid authorization format"}}`, http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == "" {
				http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"empty token"}}`, http.StatusUnauthorized)
				return
			}

			userID, err := svc.ValidateAccessToken(token)
			if err != nil {
				http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"invalid or expired token"}}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), auth.CtxUserID, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func OptionalAuth(svc *auth.Service) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				next.ServeHTTP(w, r)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			userID, err := svc.ValidateAccessToken(token)
			if err == nil {
				ctx := context.WithValue(r.Context(), auth.CtxUserID, userID)
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}

func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	return auth.GetUserIDFromCtx(ctx)
}
