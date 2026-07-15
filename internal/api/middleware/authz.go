package middleware

import (
	"context"
	"net/http"

	"nexora/internal/modules/auth"
	casbinPkg "nexora/internal/pkg/casbin"
)

type roleCtxKey string

const CtxUserRole roleCtxKey = "user_role"

func GetUserRole(ctx context.Context) (string, bool) {
	v := ctx.Value(CtxUserRole)
	if v == nil {
		return "", false
	}
	role, ok := v.(string)
	return role, ok
}

func SetUserRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, CtxUserRole, role)
}

func RequirePermission(enforcer *casbinPkg.Enforcer, obj, act string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := auth.GetUserIDFromCtx(r.Context())
			if !ok {
				http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"not authenticated"}}`, http.StatusUnauthorized)
				return
			}

			role, ok := GetUserRole(r.Context())
			if !ok {
				role = "user"
			}

			siteID, _ := GetSiteID(r.Context())
			domain := siteID.String()

			allowed, err := enforcer.Enforce(role, domain, obj, act)
			if err != nil {
				http.Error(w, `{"error":{"code":"INTERNAL","message":"authorization check failed"}}`, http.StatusInternalServerError)
				return
			}

			if !allowed {
				http.Error(w, `{"error":{"code":"FORBIDDEN","message":"insufficient permissions"}}`, http.StatusForbidden)
				return
			}

			_ = userID
			next.ServeHTTP(w, r)
		})
	}
}
