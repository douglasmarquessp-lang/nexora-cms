package site

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"nexora/internal/api/rest"
	"nexora/internal/modules/auth"
	"nexora/internal/pkg/logger"
)

type Handler struct {
	svc *Service
	log *logger.Logger
}

func NewHandler(svc *Service, log *logger.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) Create(ctx *rest.Context) {
	var req CreateSiteRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Name == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "name is required")
		return
	}
	if req.Slug == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "slug is required")
		return
	}

	userID, ok := auth.GetUserIDFromCtx(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	site, err := h.svc.CreateSite(ctx.Request.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrSiteSlugAlreadyExists):
			ctx.Error(http.StatusConflict, "SLUG_EXISTS", err.Error())
		default:
			h.log.Error("failed to create site", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create site")
		}
		return
	}

	ctx.JSON(http.StatusCreated, SiteResponse{Site: site})
}

func (h *Handler) Get(ctx *rest.Context) {
	siteID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid site ID")
		return
	}

	site, err := h.svc.GetSite(ctx.Request.Context(), siteID)
	if err != nil {
		switch {
		case errors.Is(err, ErrSiteNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "site not found")
		default:
			h.log.Error("failed to get site", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get site")
		}
		return
	}

	ctx.JSON(http.StatusOK, SiteResponse{Site: site})
}

func (h *Handler) List(ctx *rest.Context) {
	userID, ok := auth.GetUserIDFromCtx(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	page, _ := strconv.Atoi(ctx.Request.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(ctx.Request.URL.Query().Get("per_page"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	resp, err := h.svc.ListSites(ctx.Request.Context(), userID, page, perPage)
	if err != nil {
		h.log.Error("failed to list sites", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list sites")
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) Update(ctx *rest.Context) {
	siteID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid site ID")
		return
	}

	var req UpdateSiteRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	site, err := h.svc.UpdateSite(ctx.Request.Context(), siteID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrSiteNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "site not found")
		default:
			h.log.Error("failed to update site", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update site")
		}
		return
	}

	ctx.JSON(http.StatusOK, SiteResponse{Site: site})
}

func (h *Handler) Delete(ctx *rest.Context) {
	siteID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid site ID")
		return
	}

	userID, ok := auth.GetUserIDFromCtx(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	err = h.svc.DeleteSite(ctx.Request.Context(), siteID, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrSiteNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "site not found")
		case errors.Is(err, ErrSiteNotAvailable):
			ctx.Error(http.StatusForbidden, "FORBIDDEN", "site not available")
		default:
			h.log.Error("failed to delete site", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to delete site")
		}
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) AddDomain(ctx *rest.Context) {
	siteID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid site ID")
		return
	}

	var req AddDomainRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Domain == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "domain is required")
		return
	}

	sd, err := h.svc.AddDomain(ctx.Request.Context(), siteID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidDomain):
			ctx.Error(http.StatusBadRequest, "INVALID_DOMAIN", err.Error())
		case errors.Is(err, ErrDomainAlreadyExists):
			ctx.Error(http.StatusConflict, "DOMAIN_EXISTS", err.Error())
		case errors.Is(err, ErrSiteNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "site not found")
		default:
			h.log.Error("failed to add domain", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to add domain")
		}
		return
	}

	ctx.JSON(http.StatusCreated, sd)
}

func (h *Handler) RemoveDomain(ctx *rest.Context) {
	domainID, err := uuid.Parse(chi.URLParam(ctx.Request, "domainID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid domain ID")
		return
	}

	err = h.svc.RemoveDomain(ctx.Request.Context(), domainID)
	if err != nil {
		switch {
		case errors.Is(err, ErrDomainNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "domain not found")
		default:
			h.log.Error("failed to remove domain", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to remove domain")
		}
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) ListDomains(ctx *rest.Context) {
	siteID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid site ID")
		return
	}

	domains, err := h.svc.ListDomains(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to list domains", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list domains")
		return
	}

	if domains == nil {
		domains = []SiteDomain{}
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{"domains": domains})
}

func (h *Handler) SetPrimaryDomain(ctx *rest.Context) {
	siteID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid site ID")
		return
	}

	domainID, err := uuid.Parse(chi.URLParam(ctx.Request, "domainID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid domain ID")
		return
	}

	err = h.svc.SetPrimaryDomain(ctx.Request.Context(), siteID, domainID)
	if err != nil {
		switch {
		case errors.Is(err, ErrDomainNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "domain not found")
		default:
			h.log.Error("failed to set primary domain", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to set primary domain")
		}
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) GetGlobalSetting(ctx *rest.Context) {
	key := chi.URLParam(ctx.Request, "key")
	if key == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "key is required")
		return
	}

	gs, err := h.svc.GetGlobalSetting(ctx.Request.Context(), key)
	if err != nil {
		switch {
		case errors.Is(err, ErrGlobalSettingNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "setting not found")
		default:
			h.log.Error("failed to get global setting", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get setting")
		}
		return
	}

	ctx.JSON(http.StatusOK, gs)
}

func (h *Handler) SetGlobalSetting(ctx *rest.Context) {
	var req UpdateGlobalSettingRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	gs, err := h.svc.SetGlobalSetting(ctx.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidSettingType):
			ctx.Error(http.StatusBadRequest, "INVALID_TYPE", err.Error())
		default:
			h.log.Error("failed to set global setting", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to set setting")
		}
		return
	}

	ctx.JSON(http.StatusOK, gs)
}

func (h *Handler) ListGlobalSettings(ctx *rest.Context) {
	settings, err := h.svc.ListGlobalSettings(ctx.Request.Context())
	if err != nil {
		h.log.Error("failed to list global settings", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list settings")
		return
	}

	if settings == nil {
		settings = []GlobalSetting{}
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{"settings": settings})
}

func (h *Handler) GetSiteSetting(ctx *rest.Context) {
	siteID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid site ID")
		return
	}

	key := chi.URLParam(ctx.Request, "key")
	if key == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "key is required")
		return
	}

	ss, err := h.svc.GetSiteSetting(ctx.Request.Context(), siteID, key)
	if err != nil {
		switch {
		case errors.Is(err, ErrSiteSettingNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "setting not found")
		default:
			h.log.Error("failed to get site setting", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get setting")
		}
		return
	}

	ctx.JSON(http.StatusOK, ss)
}

func (h *Handler) SetSiteSetting(ctx *rest.Context) {
	siteID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid site ID")
		return
	}

	var req SetSiteSettingRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Key == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "key is required")
		return
	}

	ss, err := h.svc.SetSiteSetting(ctx.Request.Context(), siteID, req)
	if err != nil {
		h.log.Error("failed to set site setting", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to set setting")
		return
	}

	ctx.JSON(http.StatusOK, ss)
}

func (h *Handler) ListSiteSettings(ctx *rest.Context) {
	siteID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid site ID")
		return
	}

	settings, err := h.svc.ListSiteSettings(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to list site settings", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list settings")
		return
	}

	if settings == nil {
		settings = []SiteSetting{}
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{"settings": settings})
}


