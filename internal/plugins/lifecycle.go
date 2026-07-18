package plugins

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type Lifecycle struct {
	registry *Registry
	hooks    *Hooks
	events   EventEmitter
}

type EventEmitter interface {
	Emit(ctx context.Context, eventType string, payload interface{}, siteID string) error
}

func NewLifecycle(registry *Registry, hooks *Hooks, events EventEmitter) *Lifecycle {
	return &Lifecycle{
		registry: registry,
		hooks:    hooks,
		events:   events,
	}
}

func (lc *Lifecycle) Install(ctx context.Context, instance *PluginInstance) error {
	manifest := instance.Manifest

	for _, dep := range manifest.Dependencies {
		if dep.ID == "" {
			continue
		}
		depPlugin := lc.registry.Get(dep.ID)
		if depPlugin == nil {
			return fmt.Errorf("missing dependency %q for plugin %q", dep.ID, manifest.ID)
		}
		if depPlugin.Status != PluginStatusActive {
			return fmt.Errorf("dependency %q is not active for plugin %q", dep.ID, manifest.ID)
		}
	}

	if err := lc.hooks.DoAction(ctx, HookBeforePluginInstall, map[string]interface{}{
		"plugin_id": manifest.ID,
		"version":   manifest.Version,
	}); err != nil {
		slog.Warn("HookBeforePluginInstall failed", "error", err, "plugin_id", manifest.ID)
	}

	instance.Status = PluginStatusInstalled

	if err := lc.events.Emit(ctx, "plugin.installed", map[string]interface{}{
		"plugin_id": manifest.ID,
		"version":   manifest.Version,
		"name":      manifest.Name,
	}, ""); err != nil {
		slog.Warn("plugin.installed event emit failed", "error", err, "plugin_id", manifest.ID)
	}

	if err := lc.hooks.DoAction(ctx, HookAfterPluginInstall, map[string]interface{}{
		"plugin_id": manifest.ID,
		"version":   manifest.Version,
	}); err != nil {
		slog.Warn("HookAfterPluginInstall failed", "error", err, "plugin_id", manifest.ID)
	}

	return nil
}

func (lc *Lifecycle) Activate(ctx context.Context, instance *PluginInstance) error {
	manifest := instance.Manifest

	if err := lc.hooks.DoAction(ctx, HookBeforePluginActivate, map[string]interface{}{
		"plugin_id": manifest.ID,
	}); err != nil {
		slog.Warn("HookBeforePluginActivate failed", "error", err, "plugin_id", manifest.ID)
	}

	for _, h := range manifest.Hooks {
		if !IsValidHook(h.Hook) {
			return fmt.Errorf("plugin %q declares invalid hook %q", manifest.ID, h.Hook)
		}
	}

	instance.Status = PluginStatusActive

	if err := lc.events.Emit(ctx, "plugin.activated", map[string]interface{}{
		"plugin_id": manifest.ID,
		"version":   manifest.Version,
		"name":      manifest.Name,
	}, ""); err != nil {
		slog.Warn("plugin.activated event emit failed", "error", err, "plugin_id", manifest.ID)
	}

	if err := lc.hooks.DoAction(ctx, HookAfterPluginActivate, map[string]interface{}{
		"plugin_id": manifest.ID,
	}); err != nil {
		slog.Warn("HookAfterPluginActivate failed", "error", err, "plugin_id", manifest.ID)
	}

	return nil
}

func (lc *Lifecycle) Deactivate(ctx context.Context, instance *PluginInstance) error {
	manifest := instance.Manifest

	if err := lc.hooks.DoAction(ctx, HookBeforePluginDeactivate, map[string]interface{}{
		"plugin_id": manifest.ID,
	}); err != nil {
		slog.Warn("HookBeforePluginDeactivate failed", "error", err, "plugin_id", manifest.ID)
	}

	instance.Status = PluginStatusInactive

	if err := lc.events.Emit(ctx, "plugin.deactivated", map[string]interface{}{
		"plugin_id": manifest.ID,
		"version":   manifest.Version,
		"name":      manifest.Name,
	}, ""); err != nil {
		slog.Warn("plugin.deactivated event emit failed", "error", err, "plugin_id", manifest.ID)
	}

	if err := lc.hooks.DoAction(ctx, HookAfterPluginDeactivate, map[string]interface{}{
		"plugin_id": manifest.ID,
	}); err != nil {
		slog.Warn("HookAfterPluginDeactivate failed", "error", err, "plugin_id", manifest.ID)
	}

	return nil
}

func (lc *Lifecycle) Update(ctx context.Context, instance *PluginInstance, newManifest *PluginManifest) error {
	oldVersion := instance.Manifest.Version
	instance.Manifest = newManifest

	if err := lc.events.Emit(ctx, "plugin.updated", map[string]interface{}{
		"plugin_id":   instance.Manifest.ID,
		"old_version": oldVersion,
		"new_version": newManifest.Version,
		"name":        instance.Manifest.Name,
	}, ""); err != nil {
		slog.Warn("plugin.updated event emit failed", "error", err, "plugin_id", instance.Manifest.ID)
	}

	return nil
}

func (lc *Lifecycle) Uninstall(ctx context.Context, instance *PluginInstance) error {
	manifest := instance.Manifest

	if err := lc.hooks.DoAction(ctx, HookBeforeDelete, map[string]interface{}{
		"plugin_id": manifest.ID,
		"version":   manifest.Version,
		"name":      manifest.Name,
	}); err != nil {
		slog.Warn("HookBeforeDelete failed", "error", err, "plugin_id", manifest.ID)
	}

	lc.hooks.RemovePlugin(manifest.ID)
	lc.registry.Remove(manifest.ID)

	if err := lc.events.Emit(ctx, "plugin.removed", map[string]interface{}{
		"plugin_id": manifest.ID,
		"version":   manifest.Version,
		"name":      manifest.Name,
	}, ""); err != nil {
		slog.Warn("plugin.removed event emit failed", "error", err, "plugin_id", manifest.ID)
	}

	if err := lc.hooks.DoAction(ctx, HookAfterDelete, map[string]interface{}{
		"plugin_id": manifest.ID,
	}); err != nil {
		slog.Warn("HookAfterDelete failed", "error", err, "plugin_id", manifest.ID)
	}

	return nil
}

type PluginRecord struct {
	ID             string     `json:"id"`
	PluginID       string     `json:"plugin_id"`
	Name           string     `json:"name"`
	Version        string     `json:"version"`
	Author         string     `json:"author"`
	Description    string     `json:"description"`
	License        string     `json:"license"`
	Homepage       string     `json:"homepage"`
	MinCoreVersion string     `json:"min_core_version"`
	Status         string     `json:"status"`
	InstalledAt    time.Time  `json:"installed_at"`
	ActivatedAt    *time.Time `json:"activated_at,omitempty"`
}
