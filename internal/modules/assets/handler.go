package assets

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"nexora/internal/api/middleware"
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

func (h *Handler) Upload(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	userID, ok := auth.GetUserIDFromCtx(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	maxSize := int64(100 << 20)
	ctx.Request.Body = http.MaxBytesReader(ctx.ResponseWriter, ctx.Request.Body, maxSize)

	if err := ctx.Request.ParseMultipartForm(maxSize); err != nil {
		ctx.Error(http.StatusBadRequest, "FILE_TOO_LARGE", "file exceeds maximum upload size")
		return
	}

	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		ctx.Error(http.StatusBadRequest, "MISSING_FILE", "file is required")
		return
	}
	defer func() { _ = file.Close() }()

	req := UploadRequest{
		AltText:     ctx.Request.FormValue("alt_text"),
		Title:       ctx.Request.FormValue("title"),
		Caption:     ctx.Request.FormValue("caption"),
		Description: ctx.Request.FormValue("description"),
	}

	asset, err := h.svc.Upload(ctx.Request.Context(), siteID, userID, file, header, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidFileType):
			ctx.Error(http.StatusBadRequest, "INVALID_FILE_TYPE", err.Error())
		case errors.Is(err, ErrFileTooLarge):
			ctx.Error(http.StatusBadRequest, "FILE_TOO_LARGE", err.Error())
		case errors.Is(err, ErrInvalidFile):
			ctx.Error(http.StatusBadRequest, "INVALID_FILE", err.Error())
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to upload asset", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to upload asset")
		}
		return
	}

	ctx.JSON(http.StatusCreated, asset)
}

func (h *Handler) Get(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	assetID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid asset ID")
		return
	}

	asset, err := h.svc.GetByID(ctx.Request.Context(), siteID, assetID)
	if err != nil {
		switch {
		case errors.Is(err, ErrAssetNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "asset not found")
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to get asset", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get asset")
		}
		return
	}

	ctx.JSON(http.StatusOK, asset)
}

func (h *Handler) List(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	q := ctx.Request.URL.Query()

	var assetType AssetType
	if t := q.Get("type"); t != "" {
		assetType = AssetType(t)
	}

	extension := q.Get("extension")
	search := q.Get("search")
	sort := q.Get("sort")
	order := q.Get("order")

	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))

	resp, err := h.svc.List(ctx.Request.Context(), AssetListRequest{
		SiteID:    siteID,
		Type:      assetType,
		Extension: extension,
		Search:    search,
		Page:      page,
		PerPage:   perPage,
		Sort:      sort,
		Order:     order,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidPagination):
			ctx.Error(http.StatusBadRequest, "INVALID_PAGINATION", err.Error())
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to list assets", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list assets")
		}
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) Update(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	assetID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid asset ID")
		return
	}

	var req UpdateAssetRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	asset, err := h.svc.Update(ctx.Request.Context(), siteID, assetID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrAssetNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "asset not found")
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to update asset", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update asset")
		}
		return
	}

	ctx.JSON(http.StatusOK, asset)
}

func (h *Handler) Delete(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	assetID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid asset ID")
		return
	}

	err = h.svc.Delete(ctx.Request.Context(), siteID, assetID)
	if err != nil {
		switch {
		case errors.Is(err, ErrAssetNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "asset not found")
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to delete asset", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to delete asset")
		}
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) LinkToPost(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var req LinkAssetRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.PostID == uuid.Nil || req.AssetID == uuid.Nil {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "post_id and asset_id are required")
		return
	}

	postAsset, err := h.svc.LinkToPost(ctx.Request.Context(), siteID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrAssetNotFound):
			ctx.Error(http.StatusNotFound, "ASSET_NOT_FOUND", "asset not found")
		case errors.Is(err, ErrPostNotFound):
			ctx.Error(http.StatusNotFound, "POST_NOT_FOUND", "post not found")
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to link asset to post", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to link asset")
		}
		return
	}

	ctx.JSON(http.StatusCreated, postAsset)
}

func (h *Handler) UnlinkFromPost(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	postID, err := uuid.Parse(chi.URLParam(ctx.Request, "postID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_POST_ID", "invalid post ID")
		return
	}

	assetID, err := uuid.Parse(chi.URLParam(ctx.Request, "assetID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ASSET_ID", "invalid asset ID")
		return
	}

	err = h.svc.UnlinkFromPost(ctx.Request.Context(), siteID, postID, assetID)
	if err != nil {
		switch {
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to unlink asset from post", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to unlink asset")
		}
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "unlinked"})
}

func (h *Handler) GetPostAssets(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	postID, err := uuid.Parse(chi.URLParam(ctx.Request, "postID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_POST_ID", "invalid post ID")
		return
	}

	postAssets, err := h.svc.GetPostAssets(ctx.Request.Context(), siteID, postID)
	if err != nil {
		switch {
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to get post assets", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get post assets")
		}
		return
	}

	ctx.JSON(http.StatusOK, postAssets)
}
