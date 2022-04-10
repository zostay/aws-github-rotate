package config

import (
	"fmt"
	"strings"
	"time"
)

// KeyMap maps the keys produced by the source rotator to the keys to use in storage.
type KeyMap map[string]string

// GithubConfig is the Github-specific configuration.
type GithubConfig struct {
	Token string `yaml:"token"` // the github token configured (you should set this via the GITHUB_TOKEN environment variable)
}

// Config is the programmatic representation of the loaded configuration.
type Config struct {
	Github GithubConfig `yaml:"github"` // github-specific configuration

	KeyMap KeyMap `yaml:"key_map"` // the map of source keys to destination storage names

	RotateAfter  time.Duration `yaml:"rotate_after"`  // the amount of time to wait before rotating a secret
	DisableAfter time.Duration `yaml:"disable_after"` // the amount of time to wait before an old secret become disabled

	Projects []Project // the project configurations

	ProjectMap map[string]*Project // the project configurations in map form
}

// Prepare performs some cleanup on the configuration to make it ready for
// general consumption. This will:
//
// 1. Check for duplicated configuration (i.e., each project name must only
//    appear once),
// 2. Make sure every Project configuration has an AccessKey and SecretKey
//    setting.
// 3. Constructs ProjectMap from Projects
//
// This will return an error if the configuration is found to be invalid for
// soem reason. Returns nil on success.
func (c *Config) Prepare() error {
	pm := make(map[string]*Project, len(c.Projects))

	for i := range c.Projects {
		if _, alreadyExists := pm[c.Projects[i].Name]; alreadyExists {
			return fmt.Errorf("The project named %q is repeated twice in the configuration.", c.Projects[i].Name)
		}

		if c.Projects[i].AccessKey == "" {
			c.Projects[i].AccessKey = c.DefaultAccessKey
		}

		if c.Projects[i].SecretKey == "" {
			c.Projects[i].SecretKey = c.DefaultSecretKey
		}

		pm[c.Projects[i].Name] = &c.Projects[i]
	}

	c.ProjectMap = pm

	return nil
}

// Repo is the repository part of the Name.
func (p Project) Repo() string {
	_, name, _ := strings.Cut(p.Name, "/")
	return name
}

// Owner is the organization/user part of the Name.
func (p Project) Owner() string {
	repo, _, _ := strings.Cut(p.Name, "/")
	return repo
}
