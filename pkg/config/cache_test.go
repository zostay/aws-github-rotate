package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	c := new(cache)

	assert.Nil(t, c.cache, "cache starts empty")

	c.initCache()
	assert.NotNil(t, c.cache, "initCache() initializes")
	assert.Empty(t, c.cache, "... to empty")

	v, ok := c.CacheGet("key")
	assert.Nil(t, v, "non-existent key is nil")
	assert.Equal(t, ok, false, "non-existent key is non-existent")

	c.CacheSet("key", "value")

	v, ok = c.CacheGet("key")
	assert.Equal(t, v, "value", "existent key is correct")
	assert.Equal(t, ok, true, "existent key is existent")

	c.CacheSet("key", "value2")

	v, ok = c.CacheGet("key")
	assert.Equal(t, v, "value2", "replacing key works")
	assert.Equal(t, ok, true, "replacing key exists")

	c.CacheClear("key")

	v, ok = c.CacheGet("key")
	assert.Nil(t, v, "deleting key is gone")
	assert.Equal(t, ok, false, "deleted key does not exist")
}

func TestCacheStruct(t *testing.T) {
	type foo struct{}

	c := new(cache)

	assert.Nil(t, c.cache, "cache starts empty")

	c.initCache()
	assert.NotNil(t, c.cache, "initCache() initializes")
	assert.Empty(t, c.cache, "... to empty")

	v, ok := c.CacheGet(foo{})
	assert.Nil(t, v, "non-existent key is nil")
	assert.Equal(t, ok, false, "non-existent key is non-existent")

	c.CacheSet(foo{}, "value")

	v, ok = c.CacheGet(foo{})
	assert.Equal(t, v, "value", "existent key is correct")
	assert.Equal(t, ok, true, "existent key is existent")

	c.CacheSet(foo{}, "value2")

	v, ok = c.CacheGet(foo{})
	assert.Equal(t, v, "value2", "replacing key works")
	assert.Equal(t, ok, true, "replacing key exists")

	c.CacheClear(foo{})

	v, ok = c.CacheGet(foo{})
	assert.Nil(t, v, "deleting key is gone")
	assert.Equal(t, ok, false, "deleted key does not exist")
}
