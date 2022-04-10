package config

// Project is the compiled metadata about each project for which we manage the
// secrets and the associated secret metadata.
type Project struct {
	Name string // The user/name of the repo from the configuration
	User string `yaml:"user"` // the IAM user name associated with the repo
}
