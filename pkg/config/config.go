package config

import (
	"fmt"
	"time"
)

// KeyMap maps the keys produced by the source rotator to the keys to use in storage.
type KeyMap map[string]string

// Client is used to load plugins the implement various client interfaces.
type Client struct {
	Name    string         `yaml:"-"`
	Package string         `yaml:"package"`
	Options map[string]any `yaml:"option"`
}

// Rotation is used to define a rotation process.
type Rotation struct {
	Client      string        `yaml:"client"`
	RotateAfter time.Duration `yaml:"rotate_after"`
	SecretSet   string        `yaml:"secret_set"`
}

// Disablement is used to define a disablement process.
type Disablement struct {
	Client       string        `yaml:"client"`
	DisableAfter time.Duration `yaml:"disable_after"`
	SecretSet    string        `yaml:"secret_set"`
}

// StorageMap describes how a secret should be stored when rotated.
type StorageMap struct {
	Storage string `yaml:"storage"`
	Name    string `yaml:"name"`
	Keys    KeyMap `yaml:"keys"`
}

// Secret defines a single rotatable secret.
type Secret struct {
	Name     string       `yaml:"name"`
	Storages []StorageMap `yaml:"storages"`
}

// SecretSet is used to difine a set of secret to use with rotation and/or
// disablement processes.
type SecretSet struct {
	Name    string   `yaml:"name"`
	Secrets []Secret `yaml:"secrets"`
}

// Config is the programmatic representation of the loaded configuration.
type Config struct {
	Clients      map[string]Client `yaml:"clients"`
	Rotations    []Rotation        `yaml:"rotations"`
	Disablements []Disablement     `yaml:disablements"`
	SecretSets   []SecretSet       `yaml:"secret_sets"`
}

// Prepare should be called after the configuration object has been unmarshaled
// from the configuration file. This will normalize the file and fill in any
// details that can be inferred. It also checks for errors in the configuration
// that are unrelated to syntax.
//
// Returns an error if there's a problem is detected with the configuration or
// nil if no problem is found.
func (c *Config) Prepare() error {
	for k, c := range c.Clients {
		c.Name = k
	}

	secSetSet := make(map[string]struct{}, len(c.SecretSets))
	for i := range c.SecretSets {
		secSet := &c.SecretSet
		if _, alreadyExists := secSetSet[secSet.Name]; alreadyExists {
			return fmt.Errorf("secret set %q is duplicated", secSet.Name)
		}
		secSetSet[secSet.Name] = struct{}{}

		secMap := make(map[string]struct{}, len(secSet.Secrets))
		for j := range secSet.Secrets {
			sec := &secSet.Secrets[j]
			if _, alreadyExists := secMap[sec.Name]; alreadyExists {
				return fmt.Errorf("in set %q, secret named %q is repeated twice in the configuration", secSet.Name, sec.Name)
			}

			secMap[sec.Name] = struct{}{}
		}
	}

	return nil
}
