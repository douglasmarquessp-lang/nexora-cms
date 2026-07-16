package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/mod/semver"
)

type PluginManifest struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Version        string          `json:"version"`
	Author         string          `json:"author"`
	Description    string          `json:"description"`
	License        string          `json:"license"`
	Homepage       string          `json:"homepage"`
	MinCoreVersion string          `json:"min_core_version"`
	Dependencies   []PluginDep     `json:"dependencies"`
	Permissions    []PluginPerm    `json:"permissions"`
	Hooks          []PluginHookDef `json:"hooks"`
	Routes         []PluginRoute   `json:"routes"`
	AdminPages     []AdminPage     `json:"admin_pages"`
}

type PluginDep struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type PluginPerm struct {
	Permission  string   `json:"permission"`
	Description string   `json:"description"`
	Roles       []string `json:"default_roles"`
}

type PluginHookDef struct {
	Hook     string `json:"hook"`
	Priority int    `json:"priority"`
}

type PluginRoute struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Handler string `json:"handler"`
	Type    string `json:"type"`
}

type AdminPage struct {
	Title    string `json:"title"`
	Path     string `json:"path"`
	Icon     string `json:"icon"`
	Position int    `json:"position"`
}

const CoreVersion = "0.1.0"

func LoadManifest(path string) (*PluginManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest %s: %w", path, err)
	}

	var m PluginManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse manifest %s: %w", path, err)
	}

	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("invalid manifest %s: %w", path, err)
	}

	return &m, nil
}

func (m *PluginManifest) Validate() error {
	var errs []string

	if m.ID == "" {
		errs = append(errs, "id is required")
	}
	if m.Name == "" {
		errs = append(errs, "name is required")
	}
	if m.Version == "" {
		errs = append(errs, "version is required")
	} else if !semver.IsValid("v"+strings.TrimPrefix(m.Version, "v")) {
		errs = append(errs, fmt.Sprintf("version %q is not valid semver", m.Version))
	}
	if m.MinCoreVersion != "" {
		cv := "v" + strings.TrimPrefix(m.MinCoreVersion, "v")
		if !semver.IsValid(cv) {
			errs = append(errs, fmt.Sprintf("min_core_version %q is not valid semver", m.MinCoreVersion))
		} else if semver.Compare(cv, "v"+CoreVersion) > 0 {
			errs = append(errs, fmt.Sprintf("requires core version %s but current is %s", m.MinCoreVersion, CoreVersion))
		}
	}
	for i, dep := range m.Dependencies {
		if dep.ID == "" {
			errs = append(errs, fmt.Sprintf("dependency %d has empty id", i))
		}
		if dep.Version != "" && !semver.IsValid("v"+strings.TrimPrefix(dep.Version, "v")) {
			errs = append(errs, fmt.Sprintf("dependency %q version %q is not valid semver", dep.ID, dep.Version))
		}
	}
	for i, r := range m.Routes {
		if r.Path == "" {
			errs = append(errs, fmt.Sprintf("route %d has empty path", i))
		}
		if r.Method == "" {
			errs = append(errs, fmt.Sprintf("route %d has empty method", i))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

var PluginDir = "plugins"
