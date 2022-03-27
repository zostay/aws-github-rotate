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
}

// ProjectMap converts the list of project configurations into a map. It returns
// an error if, in the process, it is discovered that a project name is
// repeated.
func (c Config) ProjectMap() (map[string]*Project, error) {
	pm := make(map[string]Project, len(c.Projects))

	for i := range c.Projects {
		if _, alreadyExists := pm[c.Projects[i].Name]; alreadyExists {
			return nil, fmt.Errorf("The project named %q is repeated twice in the configuration.", c.Projects[i].Name)
		}
		pm[c.Projects[i].Name] = &c.Projects[i]
	}

	return pm, nil
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
