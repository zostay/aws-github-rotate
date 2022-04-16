package plugin

import (
	"context"
	"fmt"
	"strings"

	"github.com/zostay/aws-github-rotate/pkg/config"
)

// Manager provides a mechanism for building plugis lazily and caching them.
type Manager struct {
	clients config.ClientList
	cache   map[string]Instance
}

func NewManager(clients config.ClientList) *Manager {
	return &Manager{
		clients: clients,
		cache:   make(map[string]Instance, len(clients)),
	}
}

func (m *Manager) Instance(ctx context.Context, name string) (Instance, error) {
	if inst, ok := m.cache[name]; ok {
		return inst, nil
	}

	lcname := strings.ToLower(name)
	c, ok := m.clients[lcname]
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
