package setup

import (
	"errors"
	"net/http"

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

func (h *Handler) Status(ctx *rest.Context) {
	resp, err := h.svc.Status(ctx.Request.Context())
	if err != nil {
		h.log.Error("failed to check setup status", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to check setup status")
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) Install(ctx *rest.Context) {
	var req InstallRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	result, err := h.svc.Install(ctx.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrAlreadyInstalled) {
			ctx.Error(http.StatusForbidden, "ALREADY_INSTALLED", "system is already installed")
			return
		}
		var valErr *ValidationError
		if errors.As(err, &valErr) {
			ctx.Error(http.StatusBadRequest, "VALIDATION_ERROR", "validation failed", valErr.Errors)
			return
		}
		h.log.Error("installation failed", "error", err)
		ctx.Error(http.StatusInternalServerError, "INSTALL_FAILED", "installation failed: "+err.Error())
		return
	}

	ctx.JSON(http.StatusCreated, result)
}

func (h *Handler) Config(ctx *rest.Context) {
	resp, err := h.svc.GetConfig(ctx.Request.Context())
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get config")
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) Finish(ctx *rest.Context) {
	resp, err := h.svc.Finish(ctx.Request.Context())
	if err != nil {
		if errors.Is(err, ErrNotInstalled) {
			ctx.Error(http.StatusForbidden, "NOT_INSTALLED", "system is not installed yet")
			return
		}
		h.log.Error("finish setup failed", "error", err)
		ctx.Error(http.StatusInternalServerError, "FINISH_FAILED", "failed to complete setup")
		return
	}
	ctx.JSON(http.StatusOK, resp)
}
