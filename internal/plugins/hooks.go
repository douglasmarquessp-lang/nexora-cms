package plugins

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

type HookType string

const (
	HookBeforePostSave   HookType = "before_post_save"
	HookAfterPostSave    HookType = "after_post_save"
	HookBeforeDelete     HookType = "before_delete"
	HookAfterDelete      HookType = "after_delete"
	HookBeforeLogin      HookType = "before_login"
	HookAfterLogin       HookType = "after_login"
	HookBeforeRender     HookType = "before_render"
	HookAfterRender      HookType = "after_render"
	HookBeforeUpload     HookType = "before_upload"
	HookAfterUpload      HookType = "after_upload"
	HookBeforeMediaSave  HookType = "before_media_save"
	HookAfterMediaSave   HookType = "after_media_save"
	HookBeforePluginInstall  HookType = "before_plugin_install"
	HookAfterPluginInstall   HookType = "after_plugin_install"
	HookBeforePluginActivate  HookType = "before_plugin_activate"
	HookAfterPluginActivate   HookType = "after_plugin_activate"
	HookBeforePluginDeactivate HookType = "before_plugin_deactivate"
	HookAfterPluginDeactivate  HookType = "after_plugin_deactivate"
)

type ActionHandler func(ctx context.Context, args map[string]interface{}) error

type FilterHandler func(ctx context.Context, value interface{}, args map[string]interface{}) (interface{}, error)

type HookRegistration struct {
	PluginID string
	Hook     HookType
	Priority int
}

type actionEntry struct {
	pluginID string
	priority int
	handler  ActionHandler
}

type filterEntry struct {
	pluginID string
	priority int
	handler  FilterHandler
}

type Hooks struct {
	mu      sync.RWMutex
	actions map[HookType][]actionEntry
	filters map[HookType][]filterEntry
}

func NewHooks() *Hooks {
	return &Hooks{
		actions: make(map[HookType][]actionEntry),
		filters: make(map[HookType][]filterEntry),
	}
}

func (h *Hooks) AddAction(pluginID string, hook HookType, handler ActionHandler, priority int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.actions[hook] = append(h.actions[hook], actionEntry{
		pluginID: pluginID,
		priority: priority,
		handler:  handler,
	})
	sort.Slice(h.actions[hook], func(i, j int) bool {
		return h.actions[hook][i].priority < h.actions[hook][j].priority
	})
}

func (h *Hooks) AddFilter(pluginID string, hook HookType, handler FilterHandler, priority int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.filters[hook] = append(h.filters[hook], filterEntry{
		pluginID: pluginID,
		priority: priority,
		handler:  handler,
	})
	sort.Slice(h.filters[hook], func(i, j int) bool {
		return h.filters[hook][i].priority < h.filters[hook][j].priority
	})
}

func (h *Hooks) DoAction(ctx context.Context, hook HookType, args map[string]interface{}) error {
	h.mu.RLock()
	entries, exists := h.actions[hook]
	h.mu.RUnlock()
	if !exists {
		return nil
	}
	for _, e := range entries {
		if err := e.handler(ctx, args); err != nil {
			return fmt.Errorf("plugin %q action %q error: %w", e.pluginID, hook, err)
		}
	}
	return nil
}

func (h *Hooks) ApplyFilter(ctx context.Context, hook HookType, value interface{}, args map[string]interface{}) (interface{}, error) {
	h.mu.RLock()
	entries, exists := h.filters[hook]
	h.mu.RUnlock()
	if !exists {
		return value, nil
	}
	var err error
	result := value
	for _, e := range entries {
		result, err = e.handler(ctx, result, args)
		if err != nil {
			return nil, fmt.Errorf("plugin %q filter %q error: %w", e.pluginID, hook, err)
		}
	}
	return result, nil
}

func (h *Hooks) RemovePlugin(pluginID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for hook := range h.actions {
		var filtered []actionEntry
		for _, e := range h.actions[hook] {
			if e.pluginID != pluginID {
				filtered = append(filtered, e)
			}
		}
		h.actions[hook] = filtered
	}
	for hook := range h.filters {
		var filtered []filterEntry
		for _, e := range h.filters[hook] {
			if e.pluginID != pluginID {
				filtered = append(filtered, e)
			}
		}
		h.filters[hook] = filtered
	}
}

func (h *Hooks) GetRegistrations(pluginID string) []HookRegistration {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var regs []HookRegistration
	for hook, entries := range h.actions {
		for _, e := range entries {
			if e.pluginID == pluginID {
				regs = append(regs, HookRegistration{PluginID: pluginID, Hook: hook, Priority: e.priority})
			}
		}
	}
	for hook, entries := range h.filters {
		for _, e := range entries {
			if e.pluginID == pluginID {
				regs = append(regs, HookRegistration{PluginID: pluginID, Hook: hook, Priority: e.priority})
			}
		}
	}
	return regs
}

var AvailableHooks = []HookType{
	HookBeforePostSave,
	HookAfterPostSave,
	HookBeforeDelete,
	HookAfterDelete,
	HookBeforeLogin,
	HookAfterLogin,
	HookBeforeRender,
	HookAfterRender,
	HookBeforeUpload,
	HookAfterUpload,
	HookBeforeMediaSave,
	HookAfterMediaSave,
	HookBeforePluginInstall,
	HookAfterPluginInstall,
	HookBeforePluginActivate,
	HookAfterPluginActivate,
	HookBeforePluginDeactivate,
	HookAfterPluginDeactivate,
}

func IsValidHook(hook string) bool {
	for _, h := range AvailableHooks {
		if string(h) == hook {
			return true
		}
	}
	return false
}
