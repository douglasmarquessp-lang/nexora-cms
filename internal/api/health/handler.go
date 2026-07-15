package health

import (
	"context"
	"net/http"
	"time"

	"nexora/internal/api/dto"
	"nexora/internal/api/rest"
)

const Version = "0.1.0"

type Handler struct {
	db CheckFunc
}

type CheckFunc func(ctx context.Context) error

func NewHandler(db CheckFunc) *Handler {
	return &Handler{db: db}
}

func (h *Handler) Check(ctx *rest.Context) {
	status := "ok"
	dbStatus := "connected"

	if err := h.db(ctx.Request.Context()); err != nil {
		status = "degraded"
		dbStatus = "disconnected"
	}

	ctx.JSON(http.StatusOK, dto.HealthResponse{
		Status:    status,
		Version:   Version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Database:  dbStatus,
	})
}
