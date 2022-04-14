package cmd

import (
	"fmt"

	"github.com/zostay/aws-github-rotate/pkg/config"
)

func findSecretSet(name string) (*config.SecretSet, error) {
	for i := range c.SecretSets {
		ss := &c.SecretSets[i]
		if ss.Name == name {
			return ss, nil
		}
	}
	return nil, fmt.Errorf("no secret set named %q found in configuration", name)
}
