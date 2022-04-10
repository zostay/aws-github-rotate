package rotate

import (
	"time"

	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/zostay/aws-github-rotate/pkg/config"
)

// Project is the compiled metadata about each project for which we manage the
// secrets and the associated secret metadata.
type Project struct {
	*config.Project

	secretStoredAt  time.Time // timestamp of most recent secret stored in the client
	secretRotatedAt time.Time // timestamp of most recent secret updated in the server

	// Provide tools for caching metadata related to project secrets.

	cache *map[any]any

	OldestKey  *iam.AccessKeyMetadata // the oldest IAM key metadata
	NewestKey  *iam.AccessKeyMetadata // the newest IAM key metadata
	keysCached bool                   // true after oldestKey/newestKey are set (possibly to nil)
}

// NewProject sets up a new project configuration from the project.
func NewProject(p *config.Project) *Project {
	return &Project{
		Project: p,
		cache:   make(map[any]any, 0),
	}
}

// CacheSet sets a cache key associated with the project.
func (p *Project) CacheSet(k, v any) {
	p.cache[k] = v
}

// CacheGet returns a set cache key. The return value from this function is the
// value set (or the zero value if nothing is set for that key), and a boolean
// indicating whether a value has been set.
func (p *Project) CacheGet(k any) (any, bool) {
	v, ok := p.cache[k]
	return v, ok
}

// CacheClear deletes the cache key.
func (p *Project) CacheClear(k any) {
	delete(p.cache, k)
}
