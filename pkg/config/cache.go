package config

// cache is a very simple caching system to allow clients to cache data
// associated with a configuration object.
type cache struct {
	cache map[any]any
}

// initCache initializes a cache.
func (c *cache) initCache() {
	if c.cache == nil {
		c.cache = make(map[any]any)
	}
}

// CacheSet sets a cache key associated with the secret.
func (c *cache) CacheSet(k, v any) {
	c.cache[k] = v
}

// CacheGet returns a set cache key. The return value from this function is the
// value set (or the zero value if nothing is set for that key), and a boolean
// indicating whether a value has been set.
func (c *cache) CacheGet(k any) (any, bool) {
	v, ok := c.cache[k]
	return v, ok
}

// CacheClear deletes the cache key.
func (c *cache) CacheClear(k any) {
	delete(c.cache, k)
}
