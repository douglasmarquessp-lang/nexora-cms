package plugins

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"nexora/internal/api/rest"
)

type Handler struct {
	manager *Manager
}

func NewHandler(manager *Manager) *Handler {
	return &Handler{manager: manager}
}

func (h *Handler) List(ctx *rest.Context) {
	plugins := h.manager.ListPlugins()

	type pluginResponse struct {
		ID             string          `json:"id"`
		Name           string          `json:"name"`
		Version        string          `json:"version"`
		Author         string          `json:"author"`
		Description    string          `json:"description"`
		License        string          `json:"license"`
		Homepage       string          `json:"homepage"`
		MinCoreVersion string          `json:"min_core_version"`
		Status         PluginStatus    `json:"status"`
		Dependencies   []PluginDep     `json:"dependencies"`
		Permissions    []PluginPerm    `json:"permissions"`
		Hooks          []PluginHookDef `json:"hooks"`
		AdminPages     []AdminPage     `json:"admin_pages"`
		HasSettings    bool            `json:"has_settings"`
	}

	items := make([]pluginResponse, 0, len(plugins))
	for _, p := range plugins {
		items = append(items, pluginResponse{
			ID:             p.Manifest.ID,
			Name:           p.Manifest.Name,
			Version:        p.Manifest.Version,
			Author:         p.Manifest.Author,
			Description:    p.Manifest.Description,
			License:        p.Manifest.License,
			Homepage:       p.Manifest.Homepage,
			MinCoreVersion: p.Manifest.MinCoreVersion,
			Status:         p.Status,
			Dependencies:   p.Manifest.Dependencies,
			Permissions:    p.Manifest.Permissions,
			Hooks:          p.Manifest.Hooks,
			AdminPages:     p.Manifest.AdminPages,
			HasSettings:    len(p.Manifest.Permissions) > 0,
		})
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{"plugins": items})
}

func (h *Handler) Get(ctx *rest.Context) {
	id := chi.URLParam(ctx.Request, "id")
	plugin := h.manager.GetPlugin(id)
	if plugin == nil {
		ctx.Error(http.StatusNotFound, "NOT_FOUND", "plugin not found")
		return
	}

	registrations := h.manager.Hooks().GetRegistrations(id)

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"id":                 plugin.Manifest.ID,
		"name":               plugin.Manifest.Name,
		"version":            plugin.Manifest.Version,
		"author":             plugin.Manifest.Author,
		"description":        plugin.Manifest.Description,
		"license":            plugin.Manifest.License,
		"homepage":           plugin.Manifest.Homepage,
		"min_core_version":   plugin.Manifest.MinCoreVersion,
		"status":             plugin.Status,
		"dependencies":       plugin.Manifest.Dependencies,
		"permissions":        plugin.Manifest.Permissions,
		"hooks":              plugin.Manifest.Hooks,
		"routes":             plugin.Manifest.Routes,
		"admin_pages":        plugin.Manifest.AdminPages,
		"hook_registrations": registrations,
	})
}

func (h *Handler) Install(ctx *rest.Context) {
	var req struct {
		Source string `json:"source"`
	}
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.Source == "" {
		ctx.Error(http.StatusBadRequest, "MISSING_SOURCE", "source is required")
		return
	}

	plugin, err := h.manager.Install(ctx.Request.Context(), req.Source)
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INSTALL_FAILED", err.Error())
		return
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"message":   "plugin installed successfully",
		"plugin_id": plugin.Manifest.ID,
		"version":   plugin.Manifest.Version,
	})
}

func (h *Handler) Activate(ctx *rest.Context) {
	var req struct {
		PluginID string `json:"plugin_id"`
	}
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.PluginID == "" {
		ctx.Error(http.StatusBadRequest, "MISSING_PLUGIN_ID", "plugin_id is required")
		return
	}

	if err := h.manager.Activate(ctx.Request.Context(), req.PluginID); err != nil {
		ctx.Error(http.StatusBadRequest, "ACTIVATE_FAILED", err.Error())
		return
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"message":   "plugin activated",
		"plugin_id": req.PluginID,
	})
}

func (h *Handler) Deactivate(ctx *rest.Context) {
	var req struct {
		PluginID string `json:"plugin_id"`
	}
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.PluginID == "" {
		ctx.Error(http.StatusBadRequest, "MISSING_PLUGIN_ID", "plugin_id is required")
		return
	}

	if err := h.manager.Deactivate(ctx.Request.Context(), req.PluginID); err != nil {
		ctx.Error(http.StatusBadRequest, "DEACTIVATE_FAILED", err.Error())
		return
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"message":   "plugin deactivated",
		"plugin_id": req.PluginID,
	})
}

func (h *Handler) Update(ctx *rest.Context) {
	id := chi.URLParam(ctx.Request, "id")
	if id == "" {
		ctx.Error(http.StatusBadRequest, "MISSING_ID", "plugin id is required")
		return
	}

	if err := h.manager.Update(ctx.Request.Context(), id); err != nil {
		ctx.Error(http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"message":   "plugin updated",
		"plugin_id": id,
	})
}

func (h *Handler) Delete(ctx *rest.Context) {
	id := chi.URLParam(ctx.Request, "id")
	if id == "" {
		ctx.Error(http.StatusBadRequest, "MISSING_ID", "plugin id is required")
		return
	}

	if err := h.manager.Uninstall(ctx.Request.Context(), id); err != nil {
		ctx.Error(http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"message":   "plugin removed",
		"plugin_id": id,
	})
}

func (h *Handler) GetSettings(ctx *rest.Context) {
	id := chi.URLParam(ctx.Request, "id")
	plugin := h.manager.GetPlugin(id)
	if plugin == nil {
		ctx.Error(http.StatusNotFound, "NOT_FOUND", "plugin not found")
		return
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"settings": map[string]interface{}{},
	})
}

func (h *Handler) UpdateSettings(ctx *rest.Context) {
	id := chi.URLParam(ctx.Request, "id")
	plugin := h.manager.GetPlugin(id)
	if plugin == nil {
		ctx.Error(http.StatusNotFound, "NOT_FOUND", "plugin not found")
		return
	}

	var settings map[string]interface{}
	if err := ctx.Decode(&settings); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid settings body")
		return
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"message":   "settings updated",
		"plugin_id": id,
	})
}
