package media

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"nexora/internal/api/middleware"
	"nexora/internal/api/rest"
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

	userID, ok := middleware.GetUserID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	maxSize := int64(100 << 20)

	if err := ctx.Request.ParseMultipartForm(maxSize); err != nil {
		ctx.Error(http.StatusBadRequest, "FILE_TOO_LARGE", "file exceeds maximum upload size")
		return
	}

	files := ctx.Request.MultipartForm.File["files"]
	if len(files) == 0 {
		file, header, err := ctx.Request.FormFile("file")
		if err != nil {
			ctx.Error(http.StatusBadRequest, "MISSING_FILE", "file is required")
			return
		}
		defer file.Close()

		req := UploadRequest{
			AltText: ctx.Request.FormValue("alt_text"),
			Caption: ctx.Request.FormValue("caption"),
		}

		if fid := ctx.Request.FormValue("folder_id"); fid != "" {
			uid, err := uuid.Parse(fid)
			if err == nil {
				req.FolderID = &uid
			}
		}

		media, err := h.svc.Upload(ctx.Request.Context(), siteID, userID, file, header, req)
		if err != nil {
			h.handleUploadError(ctx, err)
			return
		}

		ctx.JSON(http.StatusCreated, media)
		return
	}

	var results []Media
	var errs []string

	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: failed to open", header.Filename))
			continue
		}

		req := UploadRequest{
			AltText: ctx.Request.FormValue("alt_text"),
			Caption: ctx.Request.FormValue("caption"),
		}

		if fid := ctx.Request.FormValue("folder_id"); fid != "" {
			uid, err := uuid.Parse(fid)
			if err == nil {
				req.FolderID = &uid
			}
		}

		media, err := h.svc.Upload(ctx.Request.Context(), siteID, userID, file, header, req)
		file.Close()
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %s", header.Filename, err.Error()))
			continue
		}

		results = append(results, *media)
	}

	if results == nil {
		results = []Media{}
	}

	ctx.JSON(http.StatusCreated, map[string]interface{}{
		"media":  results,
		"errors": errs,
		"total":  len(results),
	})
}

func (h *Handler) Get(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	mediaID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid media ID")
		return
	}

	media, err := h.svc.GetByID(ctx.Request.Context(), siteID, mediaID)
	if err != nil {
		if errors.Is(err, ErrMediaNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "media not found")
		} else {
			h.log.Error("failed to get media", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get media")
		}
		return
	}

	ctx.JSON(http.StatusOK, media)
}

func (h *Handler) List(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	q := ctx.Request.URL.Query()

	req := MediaListRequest{
		SiteID:    siteID,
		Extension: q.Get("extension"),
		Search:    q.Get("search"),
		Sort:      q.Get("sort"),
		Order:     q.Get("order"),
		MimeType:  q.Get("mime_type"),
	}

	if t := q.Get("type"); t != "" {
		req.Type = MediaType(t)
	}

	if page, err := strconv.Atoi(q.Get("page")); err == nil {
		req.Page = page
	} else {
		req.Page = 1
	}

	if perPage, err := strconv.Atoi(q.Get("per_page")); err == nil {
		req.PerPage = perPage
	} else {
		req.PerPage = 20
	}

	if fid := q.Get("folder_id"); fid != "" {
		uid, err := uuid.Parse(fid)
		if err == nil {
			req.FolderID = &uid
		}
	}

	resp, err := h.svc.List(ctx.Request.Context(), req)
	if err != nil {
		h.log.Error("failed to list media", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list media")
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

	mediaID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid media ID")
		return
	}

	var req UpdateMediaRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	media, err := h.svc.Update(ctx.Request.Context(), siteID, mediaID, req)
	if err != nil {
		if errors.Is(err, ErrMediaNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "media not found")
		} else {
			h.log.Error("failed to update media", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update media")
		}
		return
	}

	ctx.JSON(http.StatusOK, media)
}

func (h *Handler) Delete(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	mediaID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid media ID")
		return
	}

	err = h.svc.Delete(ctx.Request.Context(), siteID, mediaID)
	if err != nil {
		if errors.Is(err, ErrMediaNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "media not found")
		} else {
			h.log.Error("failed to delete media", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to delete media")
		}
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) Move(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var req MoveMediaRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if len(req.MediaIDs) == 0 {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "media_ids is required")
		return
	}

	err := h.svc.Move(ctx.Request.Context(), siteID, req.MediaIDs, req.FolderID)
	if err != nil {
		h.log.Error("failed to move media", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to move media")
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "moved"})
}

func (h *Handler) Copy(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	userID, ok := middleware.GetUserID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	var req CopyMediaRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if len(req.MediaIDs) == 0 {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "media_ids is required")
		return
	}

	copied, err := h.svc.Copy(ctx.Request.Context(), siteID, userID, req.MediaIDs, req.FolderID)
	if err != nil {
		h.log.Error("failed to copy media", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to copy media")
		return
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"media": copied,
		"total": len(copied),
	})
}

func (h *Handler) Restore(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	mediaID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid media ID")
		return
	}

	err = h.svc.Restore(ctx.Request.Context(), siteID, mediaID)
	if err != nil {
		h.log.Error("failed to restore media", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to restore media")
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "restored"})
}

func (h *Handler) Search(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	query := ctx.Request.URL.Query().Get("q")
	if query == "" {
		query = ctx.Request.URL.Query().Get("search")
	}
	if query == "" {
		query = ctx.Request.URL.Query().Get("query")
	}

	page, _ := strconv.Atoi(ctx.Request.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(ctx.Request.URL.Query().Get("per_page"))

	resp, err := h.svc.Search(ctx.Request.Context(), siteID, query, page, perPage)
	if err != nil {
		h.log.Error("failed to search media", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to search media")
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) CreateFolder(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	userID, ok := middleware.GetUserID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	var req CreateFolderRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Name == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "name is required")
		return
	}

	folder, err := h.svc.CreateFolder(ctx.Request.Context(), siteID, userID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidFolderName) {
			ctx.Error(http.StatusBadRequest, "INVALID_NAME", err.Error())
		} else {
			h.log.Error("failed to create folder", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create folder")
		}
		return
	}

	ctx.JSON(http.StatusCreated, folder)
}

func (h *Handler) ListFolders(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	folders, err := h.svc.ListFolders(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to list folders", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list folders")
		return
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"folders": folders,
		"total":   len(folders),
	})
}

func (h *Handler) UpdateFolder(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	folderID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid folder ID")
		return
	}

	var req UpdateFolderRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	folder, err := h.svc.UpdateFolder(ctx.Request.Context(), siteID, folderID, req)
	if err != nil {
		if errors.Is(err, ErrFolderNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "folder not found")
		} else if errors.Is(err, ErrInvalidFolderName) {
			ctx.Error(http.StatusBadRequest, "INVALID_NAME", err.Error())
		} else {
			h.log.Error("failed to update folder", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update folder")
		}
		return
	}

	ctx.JSON(http.StatusOK, folder)
}

func (h *Handler) DeleteFolder(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	folderID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid folder ID")
		return
	}

	err = h.svc.DeleteFolder(ctx.Request.Context(), siteID, folderID)
	if err != nil {
		if errors.Is(err, ErrFolderNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "folder not found")
		} else if errors.Is(err, ErrFolderNotEmpty) {
			ctx.Error(http.StatusConflict, "FOLDER_NOT_EMPTY", "folder is not empty")
		} else {
			h.log.Error("failed to delete folder", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to delete folder")
		}
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) handleUploadError(ctx *rest.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidFileType):
		ctx.Error(http.StatusBadRequest, "INVALID_FILE_TYPE", err.Error())
	case errors.Is(err, ErrFileTooLarge):
		ctx.Error(http.StatusBadRequest, "FILE_TOO_LARGE", err.Error())
	case errors.Is(err, ErrInvalidFile):
		ctx.Error(http.StatusBadRequest, "INVALID_FILE", err.Error())
	case errors.Is(err, ErrDuplicateFile):
		ctx.Error(http.StatusConflict, "DUPLICATE_FILE", err.Error())
	case errors.Is(err, ErrStorageLimitReached):
		ctx.Error(http.StatusInsufficientStorage, "STORAGE_LIMIT", err.Error())
	default:
		h.log.Error("failed to upload media", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to upload media")
	}
}
