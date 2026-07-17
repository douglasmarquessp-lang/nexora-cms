package ai

import (
	"context"
	"sort"
	"sync"
)

type providerEntry struct {
	provider AIProvider
	config   ProviderCfg
}

type Registry struct {
	mu       sync.RWMutex
	entries  map[string]providerEntry
	ordered  []string
	defaultP string
}

func NewRegistry() *Registry {
	return &Registry{
		entries: make(map[string]providerEntry),
	}
}

func (r *Registry) Register(provider AIProvider, cfg ProviderCfg) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := provider.Name()
	if _, exists := r.entries[name]; exists {
		r.entries[name] = providerEntry{provider: provider, config: cfg}
	} else {
		r.entries[name] = providerEntry{provider: provider, config: cfg}
	}

	r.rebuildOrder()
	if r.defaultP == "" {
		r.defaultP = name
	}
	return nil
}

func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.entries, name)
	r.rebuildOrder()
}

func (r *Registry) Get(name string) (AIProvider, ProviderCfg, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.entries[name]
	if !ok {
		return nil, ProviderCfg{}, false
	}
	return entry.provider, entry.config, true
}

func (r *Registry) Default() (AIProvider, ProviderCfg, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.getByPriorityLocked()
}

func (r *Registry) getByPriorityLocked() (AIProvider, ProviderCfg, bool) {
	if r.defaultP != "" {
		if entry, ok := r.entries[r.defaultP]; ok {
			return entry.provider, entry.config, true
		}
	}
	for _, name := range r.ordered {
		if entry, ok := r.entries[name]; ok {
			return entry.provider, entry.config, true
		}
	}
	return nil, ProviderCfg{}, false
}

func (r *Registry) SetDefault(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.entries[name]; !ok {
		return ErrProviderNotFound
	}
	r.defaultP = name
	return nil
}

func (r *Registry) List() []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var infos []ProviderInfo
	for _, name := range r.ordered {
		entry, ok := r.entries[name]
		if !ok {
			continue
		}
		caps := entry.provider.Capabilities()
		infos = append(infos, ProviderInfo{
			Name:         name,
			Model:        entry.config.Model,
			Capabilities: caps,
			Priority:     entry.config.Priority,
			Weight:       entry.config.Weight,
			Enabled:      entry.config.Enabled,
		})
	}
	return infos
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.entries)
}

func (r *Registry) HasCapability(cap Capability) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, entry := range r.entries {
		for _, c := range entry.provider.Capabilities() {
			if c == cap {
				return true
			}
		}
	}
	return false
}

func (r *Registry) FindByCapability(cap Capability) []AIProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var providers []AIProvider
	for _, name := range r.ordered {
		entry, ok := r.entries[name]
		if !ok {
			continue
		}
		for _, c := range entry.provider.Capabilities() {
			if c == cap {
				providers = append(providers, entry.provider)
				break
			}
		}
	}
	return providers
}

func (r *Registry) HealthCheck(ctx context.Context) *ProviderHealthReport {
	r.mu.RLock()
	names := make([]string, len(r.ordered))
	copy(names, r.ordered)
	r.mu.RUnlock()

	report := &ProviderHealthReport{
		Providers: make([]HealthStatus, 0),
		Overall:   ProviderHealthy,
	}

	for _, name := range names {
		r.mu.RLock()
		entry, ok := r.entries[name]
		r.mu.RUnlock()
		if !ok {
			continue
		}

		status, err := entry.provider.Health(ctx)
		if err != nil {
			status.State = ProviderUnhealthy
		}
		status.Provider = name
		report.Providers = append(report.Providers, *status)

		if status.State == ProviderUnhealthy {
			report.Overall = ProviderDegraded
		}
	}

	if len(report.Providers) == 0 {
		report.Overall = ProviderUnhealthy
	}
	return report
}

func (r *Registry) rebuildOrder() {
	type namedEntry struct {
		name     string
		priority int
		weight   int
		enabled  bool
	}
	var list []namedEntry
	for name, entry := range r.entries {
		list = append(list, namedEntry{
			name:     name,
			priority: entry.config.Priority,
			weight:   entry.config.Weight,
			enabled:  entry.config.Enabled,
		})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].enabled != list[j].enabled {
			return list[i].enabled
		}
		if list[i].priority != list[j].priority {
			return list[i].priority < list[j].priority
		}
		return list[i].weight > list[j].weight
	})

	r.ordered = make([]string, len(list))
	for i, e := range list {
		r.ordered[i] = e.name
	}
}
