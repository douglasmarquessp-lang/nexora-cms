package plugins

import (
	"fmt"
	"os"
	"path/filepath"
)

type Loader struct {
	pluginsDir string
	registry   *Registry
}

func NewLoader(pluginsDir string, registry *Registry) *Loader {
	return &Loader{
		pluginsDir: pluginsDir,
		registry:   registry,
	}
}

func (l *Loader) LoadAll() ([]*PluginInstance, error) {
	entries, err := os.ReadDir(l.pluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read plugins dir %s: %w", l.pluginsDir, err)
	}

	var loaded []*PluginInstance
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name()[0] == '.' {
			continue
		}

		plugin, err := l.Load(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to load plugin %q: %w", entry.Name(), err)
		}
		if plugin != nil {
			loaded = append(loaded, plugin)
		}
	}
	return loaded, nil
}

func (l *Loader) Load(dirName string) (*PluginInstance, error) {
	pluginDir := filepath.Join(l.pluginsDir, dirName)
	manifestPath := filepath.Join(pluginDir, "plugin.json")

	info, err := os.Stat(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to stat %s: %w", manifestPath, err)
	}
	if info.IsDir() {
		return nil, nil
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, err
	}

	instance := &PluginInstance{
		Manifest: manifest,
		Dir:      pluginDir,
		Status:   PluginStatusInstalled,
	}

	return instance, nil
}

func (l *Loader) Discover() ([]string, error) {
	entries, err := os.ReadDir(l.pluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() && e.Name()[0] != '.' {
			manifestPath := filepath.Join(l.pluginsDir, e.Name(), "plugin.json")
			if _, err := os.Stat(manifestPath); err == nil {
				dirs = append(dirs, e.Name())
			}
		}
	}
	return dirs, nil
}
