package rotate

import (
	"context"
	"time"

	"github.com/zostay/aws-github-rotate/pkg/secret"
)

type Storage interface {
	Name() string
	LastSaved(context.Context, secret.Storage, string) (time.Time, error)
	SaveKeys(context.Context, secret.Storage, secret.Map) error
}

type Storages []Storage

type Client interface {
	Name() string
	LastRotated(context.Context, secret.Info) (time.Time, error)
	RotateSecret(context.Context, secret.Info) (secret.Map, error)
}

