package plugin

import (
	"context"
	"fmt"

	"github.com/zostay/garotate/pkg/config"
)

// Instance is the interface that all constructed plugins must implement.
// Presumably, they will also implement rotate.Client and/or rotate.Storage
// and/or disable.Client as well.
type Instance interface {
	// Name is the descriptive name of the plugin used in logging messages.
	Name() string
}

// InstanceList maps strings to constructed instances.
type InstanceList map[string]Instance

// Builder is the interface that the registered plugins will implement. It
// simply provides a means for constructing the plugin.
type Builder interface {
	Build(ctx context.Context, c *config.Plugin) (Instance, error)
}

// registy is where all the Builders are held after registered.
var registry = make(map[string]Builder)

// Register should be called during package initialization to add a plugin
// package to the registered list of plugins. The Go package name is preferred
// as the registered alias by convention, but it could be anything.
func Register(pkg string, b Builder) {
	registry[pkg] = b
}

// Get retrieves the builder associated with the given package or nil.
func Get(pkg string) Builder {
	return registry[pkg]
}

// Build will construct a plugin instance and return it. If the instance fails
// during construction, an error will be returned. If no plugin is registered
// for the given package, an error will be returned.
func Build(ctx context.Context, c *config.Plugin) (Instance, error) {
	p := Get(c.Package)
	if p != nil {
		return p.Build(ctx, c)
	}

	return nil, fmt.Errorf("no plugin found for package %q", c.Package)
}
