package config

import "time"

type Project struct {
	Name string
	User string `yaml:"user"`

	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
}

type Config struct {
	GithubToken string `yaml:"github_token"`

	DefaultAccessKey string `yaml:"default_access_key"`
	DefaultSecretKey string `yaml:"default_secret_key"`

	RotateAfter  time.Duration `yaml:"rotate_after"`
	DisableAfter time.Duration `yaml:"disable_after"`

	Projects []Project
}

func (c Config) ProjectMap() map[string]*Project {
	pm := make(map[string]Project, len(c.Projects))

	for i := range c.Projects {
		pm[c.Projects[i].Name] = &c.Projects[i]
	}

	return pm
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
