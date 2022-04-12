package secret

type Cache interface {
	CacheSet(any, any)
	CacheGet(any) (any, bool)
	CacheClear(any)
}

type Storage interface {
	Cache
	Name() string
}

type Info interface {
	Cache
	Secret() string
}

type Map map[string]string
