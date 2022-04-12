package disable

import (
	"context"
	"time"

	"github.com/zostay/aws-github-rotate/pkg/secret"
)

type Client interface {
	LastUpdated(context.Context, secret.Info) (time.Time, error)
	DisableSecret(context.Context, secret.Info) error
}
