package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"nexora/internal/modules/auth"
	sitePkg "nexora/internal/modules/site"
)

type rlsCtxKey string

const rlsAppliedKey rlsCtxKey = "rls_applied"

func RLSContext(svc *sitePkg.Service, dbPool interface{ Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) }) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			userID, ok := auth.GetUserIDFromCtx(ctx)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			role, _ := GetUserRole(ctx)
			if role == "" {
				role = "user"
			}

			siteID, _ := GetSiteID(ctx)
			if siteID == uuid.Nil {
				siteID = uuid.Nil
			}

			ctx = svc.SetRLSContext(ctx, userID, role, siteID)

			if dbPool != nil {
				_, _ = dbPool.Exec(ctx,
					`SELECT set_config('app.current_user_id', $1, true)`,
					userID.String(),
				)
				_, _ = dbPool.Exec(ctx,
					`SELECT set_config('app.current_user_role', $1, true)`,
					role,
				)
				_, _ = dbPool.Exec(ctx,
					`SELECT set_config('app.current_site_id', $1, true)`,
					siteID.String(),
				)
			}

			ctx = context.WithValue(ctx, rlsAppliedKey, true)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func IsRLSApplied(ctx context.Context) bool {
	v, ok := ctx.Value(rlsAppliedKey).(bool)
	return ok && v
}
