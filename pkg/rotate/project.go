package rotate

import (
	"errors"
	"time"

	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/zostay/aws-github-rotate/pkg/config"
)

// Project is the compiled metadata about each project for which we manage the
// secrets and the associated secret metadata.
type Project struct {
	*config.Project

	SecretUpdatedAt time.Time // last update time of the github action secret

	// We cache access key metadata to avoid making multiple calls to IAM that
	// return the same information.

	OldestKey  *iam.AccessKeyMetadata // the oldest IAM key metadata
	NewestKey  *iam.AccessKeyMetadata // the newest IAM key metadata
	keysCached bool                   // true after oldestKey/newestKey are set (possibly to nil)
}

// TouchGithub sets the SecretUpdatedAt time to right now.
func (p *Project) TouchGithub() {
	p.SecretUpdatedAt = time.Now()
}

// ClearAWS clears the oldest and newest key cache.
func (p *Project) ClearAWSKeyCache() {
	p.OldestKey = nil
	p.NewestKey = nil
	p.keysCached = false
}

var (
	ErrNotCached = errors.New("AWS keys not cached")
)

// GetAWSCache returns the cached keys and nil if they are cached or two nils
// and an error if they are not cached.
func (p *Project) GetAWSKeyCache() (*iam.AccessKeyMetadata, *iam.AccessKeyMetadata, error) {
	if p.keysCached {
		return p.OldestKey, p.NewestKey, nil
	} else {
		return nil, nil, ErrNotCached
	}
}

// SetAWSCache sets the AWS key cache to the given keys.
func (p *Project) SetAWSKeyCache(o, n *iam.AccessKeyMetadata) {
	p.OldestKey = o
	p.NewestKey = n
	p.keysCached = true
}
