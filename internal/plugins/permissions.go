package plugins

import (
	"fmt"
	"sync"
)

type PermissionDef struct {
	PluginID    string   `json:"plugin_id"`
	Permission  string   `json:"permission"`
	Description string   `json:"description"`
	DefaultRoles []string `json:"default_roles"`
}

type Permissions struct {
	mu          sync.RWMutex
	permissions map[string][]PermissionDef
}

func NewPermissions() *Permissions {
	return &Permissions{
		permissions: make(map[string][]PermissionDef),
	}
}

func (p *Permissions) Register(pluginID string, perms []PermissionDef) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.permissions[pluginID] = perms
}

func (p *Permissions) GetByPlugin(pluginID string) []PermissionDef {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.permissions[pluginID]
}

func (p *Permissions) GetAll() []PermissionDef {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var result []PermissionDef
	for _, perms := range p.permissions {
		result = append(result, perms...)
	}
	return result
}

func (p *Permissions) RemovePlugin(pluginID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.permissions, pluginID)
}

func (p *Permissions) Check(pluginID, permission string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	perms, exists := p.permissions[pluginID]
	if !exists {
		return fmt.Errorf("plugin %q has no registered permissions", pluginID)
	}

	for _, perm := range perms {
		if perm.Permission == permission {
			return nil
		}
	}

	return fmt.Errorf("plugin %q does not have permission %q", pluginID, permission)
}
