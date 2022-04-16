package config

import (
	"fmt"
	"time"
)

// KeyMap maps the keys produced by the source rotator to the keys to use in storage.
type KeyMap map[string]string

// Plugin is used to load plugins the implement various client interfaces.
type Plugin struct {
	Name    string         `mapstructure:"-"`
	Package string         `mapstructure:"package"`
	Options map[string]any `mapstructure:"option"`
}

// PluginList is a map of names to client configurations.
type PluginList map[string]Plugin

// Rotation is used to define a rotation process.
type Rotation struct {
	RotateClient string        `mapstructure:"client"`
	RotateAfter  time.Duration `mapstructure:"rotate_after"`
	SecretSet    string        `mapstructure:"secret_set"`
}

// Disablement is used to define a disablement process.
type Disablement struct {
	DisableClient string        `mapstructure:"client"`
	DisableAfter  time.Duration `mapstructure:"disable_after"`
	SecretSet     string        `mapstructure:"secret_set"`
}

// StorageMap describes how a secret should be stored when rotated.
type StorageMap struct {
	StorageClient string `mapstructure:"storage"`
	StorageName   string `mapstructure:"name"`
	Keys          KeyMap `mapstructure:"keys"`

	cache
}

// Name returns the configured name where the the storage plugin is expected to
// store the data.
func (sm *StorageMap) Name() string {
	return sm.StorageName
}

// Secret defines a single rotatable secret.
type Secret struct {
	SecretName string       `mapstructure:"secret"`
	Storages   []StorageMap `mapstructure:"storages"`

	cache
}

// SecretSet is used to difine a set of secret to use with rotation and/or
// disablement processes.
type SecretSet struct {
	Name    string   `mapstructure:"name"`
	Secrets []Secret `mapstructure:"secrets"`
}

// Names returns all the storage client names used in the secret set
// configuration.
func (ss *SecretSet) Names() []string {
	nm := make(map[string]struct{})
	for _, sec := range ss.Secrets {
		for _, store := range sec.Storages {
			nm[store.StorageClient] = struct{}{}
		}
	}
	names := make([]string, len(nm))
	for name := range nm {
		names = append(names, name)
	}
	return names
}

// Config is the programmatic representation of the loaded configuration.
type Config struct {
	Plugins      PluginList    `mapstructure:"plugins"`
	Rotations    []Rotation    `mapstructure:"rotations"`
	Disablements []Disablement `mapstructure:"disablements"`
	SecretSets   []SecretSet   `mapstructure:"secret_sets"`
}

// Prepare should be called after the configuration object has been unmarshaled
// from the configuration file. This will normalize the file and fill in any
// details that can be inferred. It also checks for errors in the configuration
// that are unrelated to syntax.
//
// Returns an error if there's a problem is detected with the configuration or
// nil if no problem is found.
func (c *Config) Prepare() error {
	for k, c := range c.Plugins {
		c.Name = k
	}

	secSetSet := make(map[string]struct{}, len(c.SecretSets))
	for i := range c.SecretSets {
		secSet := &c.SecretSets[i]
		if _, alreadyExists := secSetSet[secSet.Name]; alreadyExists {
			return fmt.Errorf("secret set %q is duplicated", secSet.Name)
		}
		secSetSet[secSet.Name] = struct{}{}

		secMap := make(map[string]struct{}, len(secSet.Secrets))
		for j := range secSet.Secrets {
			sec := &secSet.Secrets[j]
			sec.initCache()
			if _, alreadyExists := secMap[sec.SecretName]; alreadyExists {
				return fmt.Errorf("in set %q, secret named %q is repeated twice in the configuration", secSet.Name, sec.SecretName)
			}

			for k := range sec.Storages {
				sm := &sec.Storages[k]
				sm.initCache()
			}

			secMap[sec.SecretName] = struct{}{}
		}
	}

	return nil
}

// Name returns the name of the secret.
func (s *Secret) Name() string {
	return s.SecretName
}
