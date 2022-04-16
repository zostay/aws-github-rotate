// Package cmd provides tools for running the various sub-commands.
package cmd

import (
	"fmt"

	"github.com/zostay/aws-github-rotate/pkg/config"
)

// TODO Maybe findSecretSet belongs in config?

// findSecretSet looks up a secret set by name from the configuration and
// returns it or returns an error if no such set can be found.
func findSecretSet(name string) (*config.SecretSet, error) {
	for i := range c.SecretSets {
		ss := &c.SecretSets[i]
		if ss.Name == name {
			return ss, nil
		}
	}
	return nil, fmt.Errorf("no secret set named %q found in configuration", name)
}
