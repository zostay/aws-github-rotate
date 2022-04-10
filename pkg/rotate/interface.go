package rotate

import (
	"context"
	"time"
)

type CacheInfo interface {
	CacheSet(any, any)
	CacheGet(any) (any, bool)
	CacheClear(any)
}

type StorageInfo interface {
	CacheInfo
	Name() string
}

type SecretInfo interface {
	CacheInfo
	Secret() string
}

type Secrets map[string]string

type Storage interface {
	Name() string
	LastSaved(context.Context, StorageInfo, string) (time.Time, error)
	SaveKeys(context.Context, StorageInfo, Secrets) error
}

type Storages []Storage

type Client interface {
	Name() string
	LastRotated(context.Context, SecretInfo) (time.Time, error)
	RotateSecret(context.Context, SecretInfo) (Secrets, error)
}

type DisableClient interface {
	LastUpdated(context.Context, SecretInfo) (time.Time, error)
	DisableSecret(context.Context, SecretInfo) error
}
