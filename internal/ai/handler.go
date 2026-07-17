package ai

import (
	"errors"
	"net/http"

	"nexora/internal/api/middleware"
	"nexora/internal/api/rest"
	"nexora/internal/pkg/logger"
)

type Handler struct {
	manager *Manager
	log     *logger.Logger
}

func NewHandler(manager *Manager, log *logger.Logger) *Handler {
	return &Handler{manager: manager, log: log}
}

type listProvidersResponse struct {
	Providers []ProviderInfo `json:"providers"`
	Total     int            `json:"total"`
}

func (h *Handler) ListProviders(ctx *rest.Context) {
	providers := h.manager.ListProviders()
	ctx.JSON(http.StatusOK, listProvidersResponse{
		Providers: providers,
		Total:     len(providers),
	})
}

func (h *Handler) HealthCheck(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	report, err := h.manager.Health(ctx.Request.Context())
	if err != nil && !errors.Is(err, ErrHealthCheckFailed) {
		h.log.Error("AI health check failed", "error", err)
		ctx.Error(http.StatusInternalServerError, "HEALTH_FAILED", "health check failed")
		return
	}

	_ = siteID
	ctx.JSON(http.StatusOK, report)
}

func (h *Handler) TestProvider(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var req struct {
		Provider string `json:"provider"`
	}
	if err := ctx.Decode(&req); err != nil {
		req.Provider = ""
	}

	provider, err := h.manager.Provider(req.Provider)
	if err != nil {
		provider, err = h.manager.DefaultProvider()
		if err != nil {
			ctx.Error(http.StatusNotFound, "NO_PROVIDER", "no AI provider available")
			return
		}
	}

	result := AITestResult{
		Provider: provider.Name(),
		Model:    "",
	}

	caps := provider.Capabilities()
	for _, cap := range caps {
		switch cap {
		case CapGenerate:
			_, testErr := provider.Generate(ctx.Request.Context(), CompletionRequest{
				Prompt: "Test prompt",
			})
			result.Generate = testErr == nil
			if testErr != nil {
				result.Error = testErr.Error()
			}
		case CapStream:
			result.Stream = true
		case CapEmbeddings:
			result.Embeddings = true
		case CapSummarize:
			result.Summarize = true
		case CapRewrite:
			result.Rewrite = true
		case CapClassify:
			result.Classify = true
		}
	}

	_ = siteID
	ctx.JSON(http.StatusOK, result)
}

func (h *Handler) PreviewPrompt(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var req struct {
		TemplateID string            `json:"template_id"`
		Variables  map[string]string `json:"variables"`
	}
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.TemplateID == "" {
		ctx.Error(http.StatusBadRequest, "MISSING_TEMPLATE", "template_id is required")
		return
	}

	completionReq, err := h.manager.Prompts().Build(ctx.Request.Context(), req.TemplateID, req.Variables)
	if err != nil {
		if errors.Is(err, ErrInvalidPromptTemplate) {
			ctx.Error(http.StatusNotFound, "TEMPLATE_NOT_FOUND", "prompt template not found")
		} else {
			h.log.Error("failed to build prompt", "error", err)
			ctx.Error(http.StatusInternalServerError, "BUILD_FAILED", "failed to build prompt")
		}
		return
	}

	resp := map[string]interface{}{
		"template_id": req.TemplateID,
		"prompt":      completionReq.Prompt,
		"system":      completionReq.System,
	}
	_ = siteID
	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) GetCapabilities(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	caps := h.manager.Capabilities()
	providers := h.manager.ListProviders()

	resp := map[string]interface{}{
		"capabilities": caps,
		"providers":    providers,
		"total_providers": len(providers),
	}
	_ = siteID
	ctx.JSON(http.StatusOK, resp)
}
