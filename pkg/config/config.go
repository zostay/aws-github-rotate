package config

import (
	"fmt"
	"time"
)

// Project is the compiled metadata about each project for which we manage the
// secrets and the associated secret metadata.
type Project struct {
	Name string // The user/name of the repo from the configuration
	User string `yaml:"user"` // the IAM user name associated with the repo

	AccessKey string `yaml:"access_key"` // The action secret key in which the access key is stored
	SecretKey string `yaml:"secret_key"` // The action secret key in which the secret key is stored
}

// Config is the programmatic representation of the loaded configuration.
type Config struct {
	GithubToken string `yaml:"github_token"` // the github token configured (you should set this via the GITHUB_TOKEN environment variable)

	DefaultAccessKey string `yaml:"default_access_key"` // the action secret key in which the access key is stored (unless overidden by the project)
	DefaultSecretKey string `yaml:"default_secret_key"` // the action secret key in which the secret key is stored (unless overidden by the project)

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
	pm := make(map[string]Project, len(c.Projects))

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
	_, name, _ := p.Cut(p.Name, "/")
	return name
}

// Owner is the organization/user part of the Name.
func (p Project) Owner() string {
	repo, _, _ := p.Cut(p.Name, "/")
	return repo
}
