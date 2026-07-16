package plugins

import (
	"fmt"
	"sync"
)

type PluginStatus string

const (
	PluginStatusInstalled PluginStatus = "installed"
	PluginStatusActive    PluginStatus = "active"
	PluginStatusInactive  PluginStatus = "inactive"
)

type PluginInstance struct {
	Manifest   *PluginManifest
	Status     PluginStatus
	Dir        string
	DBID       string
	routes     []RegisteredRoute
	adminPages []AdminPage
}

type RegisteredRoute struct {
	Method  string
	Path    string
	Handler string
	Type    string
	Plugin  string
}

type Registry struct {
	mu      sync.RWMutex
	plugins map[string]*PluginInstance
}

func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]*PluginInstance),
	}
}

func (r *Registry) Register(plugin *PluginInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[plugin.Manifest.ID]; exists {
		return fmt.Errorf("plugin %q already registered", plugin.Manifest.ID)
	}

	r.plugins[plugin.Manifest.ID] = plugin
	return nil
}

func (r *Registry) Get(id string) *PluginInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.plugins[id]
}

func (r *Registry) GetAll() []*PluginInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*PluginInstance, 0, len(r.plugins))
	for _, p := range r.plugins {
		result = append(result, p)
	}
	return result
}

func (r *Registry) ListByStatus(status PluginStatus) []*PluginInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*PluginInstance
	for _, p := range r.plugins {
		if p.Status == status {
			result = append(result, p)
		}
	}
	return result
}

func (r *Registry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.plugins, id)
}

func (r *Registry) SetStatus(id string, status PluginStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, exists := r.plugins[id]
	if !exists {
		return fmt.Errorf("plugin %q not found", id)
	}
	p.Status = status
	return nil
}

func (r *Registry) SetDBID(id, dbID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, exists := r.plugins[id]
	if !exists {
		return fmt.Errorf("plugin %q not found", id)
	}
	p.DBID = dbID
	return nil
}

func (r *Registry) Exists(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.plugins[id]
	return exists
}
