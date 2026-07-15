package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"nexora/internal/modules/site"
)

type siteCtxKey string

const (
	CtxSiteID   siteCtxKey = "site_id"
	CtxSiteSlug siteCtxKey = "site_slug"
)

func GetSiteID(ctx context.Context) (uuid.UUID, bool) {
	v := ctx.Value(CtxSiteID)
	if v == nil {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

func GetSiteSlug(ctx context.Context) (string, bool) {
	v := ctx.Value(CtxSiteSlug)
	if v == nil {
		return "", false
	}
	slug, ok := v.(string)
	return slug, ok
}

func IdentifySite(svc *site.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			siteIDStr := r.Header.Get("X-Site-ID")
			if siteIDStr != "" {
				siteID, err := uuid.Parse(siteIDStr)
				if err == nil {
					s, err := svc.GetSite(ctx, siteID)
					if err == nil && s != nil {
						ctx = context.WithValue(ctx, CtxSiteID, s.ID)
						ctx = context.WithValue(ctx, CtxSiteSlug, s.Slug)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}

			host := r.Host
			if host == "" {
				host = r.Header.Get("X-Forwarded-Host")
			}
			if host == "" {
				host = r.Header.Get("Host")
			}

			if host != "" {
				host = strings.Split(host, ":")[0]
				s, err := svc.GetSiteByDomain(ctx, host)
				if err == nil && s != nil {
					ctx = context.WithValue(ctx, CtxSiteID, s.ID)
					ctx = context.WithValue(ctx, CtxSiteSlug, s.Slug)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			ctx = context.WithValue(ctx, CtxSiteID, uuid.Nil)
			ctx = context.WithValue(ctx, CtxSiteSlug, "")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireSite(svc *site.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			siteIDStr := r.Header.Get("X-Site-ID")
			if siteIDStr != "" {
				siteID, err := uuid.Parse(siteIDStr)
				if err == nil {
					s, err := svc.GetSite(ctx, siteID)
					if err == nil && s != nil {
						ctx = context.WithValue(ctx, CtxSiteID, s.ID)
						ctx = context.WithValue(ctx, CtxSiteSlug, s.Slug)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}

			host := r.Host
			if host == "" {
				host = r.Header.Get("X-Forwarded-Host")
			}
			if host == "" {
				host = r.Header.Get("Host")
			}

			if host != "" {
				host = strings.Split(host, ":")[0]
				s, err := svc.GetSiteByDomain(ctx, host)
				if err == nil && s != nil {
					ctx = context.WithValue(ctx, CtxSiteID, s.ID)
					ctx = context.WithValue(ctx, CtxSiteSlug, s.Slug)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			http.Error(w, `{"error":{"code":"SITE_REQUIRED","message":"site identifier is required"}}`, http.StatusBadRequest)
		})
	}
}
