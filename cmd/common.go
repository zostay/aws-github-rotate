package cmd

import (
	"context"

	"github.com/zostay/aws-github-rotate/pkg/config"
	"github.com/zostay/aws-github-rotate/pkg/plugin"
)

var (
	availablePlugins map[string]plugin.Instance
)

func loadPlugins(
	ctx context.Context,
	c map[string]config.Client,
) error {
	for k, cc := range c {
		i, err := plugin.Build(ctx, c.Package)
		if err != nil {
			return fmt.Errorf("failed to load plugin %q in package %q: %w", k, c.Package, err)
		}

		availablePlugins[k] = i
	}
	return nil
}
