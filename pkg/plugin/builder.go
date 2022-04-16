package plugin

import (
	"context"
	"fmt"
	"strings"

	"github.com/zostay/garotate/pkg/config"
)

// Manager provides a mechanism for building plugis lazily and caching them.
// Plugins returned by the methods of this object will be constructed the first
// the first time they are requested. Every subsequent call will return the
// cached value.
type Manager struct {
	plugins config.PluginList
	cache   map[string]Instance
}

// NewManager returns a Manager object for the given configuration.
func NewManager(plugins config.PluginList) *Manager {
	return &Manager{
		plugins: plugins,
		cache:   make(map[string]Instance, len(plugins)),
	}
}

// Instance first checks to see if the named plugin has already been built and
// cached. If so, it short-circuits the build process and returns the cached
// copy. If not, it looks up the configuration for the named plugin and then
// calls the Build() function to build it. It caches the instance and returns
// it.
//
// If no plugin with the given name can be found it will return a nil instance
// and an error.
//
// If an error occurs building the plugin, it will return an nil instance and an
// error.
func (m *Manager) Instance(ctx context.Context, name string) (Instance, error) {
	if inst, ok := m.cache[name]; ok {
		return inst, nil
	}

	lcname := strings.ToLower(name)
	c, ok := m.plugins[lcname]
	if !ok {
		return nil, fmt.Errorf("no plugin configuration found for name %q", name)
	}

	inst, err := Build(ctx, &c)
	if err != nil {
		return nil, fmt.Errorf("error while building plugin %q in package %q: %w", name, c.Package, err)
	}

	m.cache[name] = inst

	return inst, nil
}
