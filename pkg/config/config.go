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

// ClientList is a map of names to client configurations.
type ClientList map[string]Client

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
	StorageClient string `yaml:"storage"`
	StorageName   string `yaml:"name"`
	Keys          KeyMap `yaml:"keys"`

	cache
}

func (sm *StorageMap) Name() string {
	return sm.StorageName
}

// Secret defines a single rotatable secret.
type Secret struct {
	SecretName string       `yaml:"name"`
	Storages   []StorageMap `yaml:"storages"`

	cache
}

// SecretSet is used to difine a set of secret to use with rotation and/or
// disablement processes.
type SecretSet struct {
	Name    string   `yaml:"name"`
	Secrets []Secret `yaml:"secrets"`
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
	Clients      ClientList    `yaml:"clients"`
	Rotations    []Rotation    `yaml:"rotations"`
	Disablements []Disablement `yaml:disablements"`
	SecretSets   []SecretSet   `yaml:"secret_sets"`
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
