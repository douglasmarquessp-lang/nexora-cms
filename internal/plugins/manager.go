package plugins

import (
	"context"
	"fmt"

	"nexora/internal/pkg/logger"
)

type Manager struct {
	cfg         *ManagerConfig
	log         *logger.Logger
	registry    *Registry
	loader      *Loader
	lifecycle   *Lifecycle
	hooks       *Hooks
	permissions *Permissions
	sandbox     *Sandbox
}

type ManagerConfig struct {
	PluginsDir string
	Sandbox    SandboxConfig
}

func NewManager(cfg *ManagerConfig, log *logger.Logger, events EventEmitter) *Manager {
	registry := NewRegistry()
	hooks := NewHooks()

	return &Manager{
		cfg:         cfg,
		log:         log,
		registry:    registry,
		loader:      NewLoader(cfg.PluginsDir, registry),
		lifecycle:   NewLifecycle(registry, hooks, events),
		hooks:       hooks,
		permissions: NewPermissions(),
		sandbox:     NewSandbox(cfg.Sandbox),
	}
}

func (m *Manager) Init(ctx context.Context) error {
	plugins, err := m.loader.LoadAll()
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	for _, p := range plugins {
		if err := m.registry.Register(p); err != nil {
			m.log.Warn("failed to register plugin", "plugin", p.Manifest.ID, "error", err)
			continue
		}

		m.permissions.Register(p.Manifest.ID, toPermissionDefs(p.Manifest))
		m.log.Info("plugin loaded", "plugin", p.Manifest.ID, "version", p.Manifest.Version)
	}

	m.log.Info("plugin manager initialized", "count", len(plugins))
	return nil
}

func (m *Manager) GetPlugin(id string) *PluginInstance {
	return m.registry.Get(id)
}

func (m *Manager) ListPlugins() []*PluginInstance {
	return m.registry.GetAll()
}

func (m *Manager) ListActive() []*PluginInstance {
	return m.registry.ListByStatus(PluginStatusActive)
}

func (m *Manager) Install(ctx context.Context, dirName string) (*PluginInstance, error) {
	instance, err := m.loader.Load(dirName)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin from %s: %w", dirName, err)
	}
	if instance == nil {
		return nil, fmt.Errorf("no plugin found in %s", dirName)
	}

	if err := m.sandbox.ValidateManifest(instance.Manifest); err != nil {
		return nil, fmt.Errorf("manifest validation failed: %w", err)
	}

	if err := m.registry.Register(instance); err != nil {
		return nil, err
	}

	m.permissions.Register(instance.Manifest.ID, toPermissionDefs(instance.Manifest))

	if err := m.lifecycle.Install(ctx, instance); err != nil {
		m.registry.Remove(instance.Manifest.ID)
		return nil, err
	}

	m.log.Info("plugin installed", "plugin", instance.Manifest.ID, "version", instance.Manifest.Version)
	return instance, nil
}

func (m *Manager) Activate(ctx context.Context, id string) error {
	instance := m.registry.Get(id)
	if instance == nil {
		return fmt.Errorf("plugin %q not found", id)
	}

	if err := m.lifecycle.Activate(ctx, instance); err != nil {
		return err
	}

	m.log.Info("plugin activated", "plugin", id)
	return nil
}

func (m *Manager) Deactivate(ctx context.Context, id string) error {
	instance := m.registry.Get(id)
	if instance == nil {
		return fmt.Errorf("plugin %q not found", id)
	}

	if err := m.lifecycle.Deactivate(ctx, instance); err != nil {
		return err
	}

	m.log.Info("plugin deactivated", "plugin", id)
	return nil
}

func (m *Manager) Uninstall(ctx context.Context, id string) error {
	instance := m.registry.Get(id)
	if instance == nil {
		return fmt.Errorf("plugin %q not found", id)
	}

	if err := m.lifecycle.Uninstall(ctx, instance); err != nil {
		return err
	}

	m.permissions.RemovePlugin(id)
	m.log.Info("plugin uninstalled", "plugin", id)
	return nil
}

func (m *Manager) Update(ctx context.Context, id string) error {
	instance := m.registry.Get(id)
	if instance == nil {
		return fmt.Errorf("plugin %q not found", id)
	}

	newInstance, err := m.loader.Load(instance.Manifest.ID)
	if err != nil {
		return fmt.Errorf("failed to reload plugin: %w", err)
	}
	if newInstance == nil {
		return fmt.Errorf("plugin %q no longer exists on disk", id)
	}

	if err := m.lifecycle.Update(ctx, instance, newInstance.Manifest); err != nil {
		return err
	}

	m.log.Info("plugin updated", "plugin", id, "version", instance.Manifest.Version)
	return nil
}

func (m *Manager) Hooks() *Hooks {
	return m.hooks
}

func (m *Manager) Registry() *Registry {
	return m.registry
}

func (m *Manager) Permissions() *Permissions {
	return m.permissions
}

func (m *Manager) Sandbox() *Sandbox {
	return m.sandbox
}

func (m *Manager) Lifecycle() *Lifecycle {
	return m.lifecycle
}

func toPermissionDefs(m *PluginManifest) []PermissionDef {
	defs := make([]PermissionDef, len(m.Permissions))
	for i, p := range m.Permissions {
		defs[i] = PermissionDef{
			PluginID:     m.ID,
			Permission:   p.Permission,
			Description:  p.Description,
			DefaultRoles: p.Roles,
		}
	}
	return defs
}
