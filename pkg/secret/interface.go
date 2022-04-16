// Package secret defines some generic interfaces used to describe secrets in
// different contexts. There are two contexts:
//
// * Info
// * Storage
//
// The Info context is used to describe the secret for use with the rotation and
// disablement plugin clients. This will describe the user or role or
// host or whatever that owns the secret. This is the server side
// description of the secret. There is only one Info context per secret.
//
// The Storage context is used to describe teh secret for use with the storage
// plugin clients. This will describe the project, the service, the application,
// or whatever that uses the secret to do something. This is the cilent side of
// description of the secret. There can be zero or more Storage contexts per
// secret.
package secret

// Cache each secret context has an associated cache, which allows the plugin
// client to store any information that is expensive to caculate with the secret
// context.
type Cache interface {
	// CacheSet stores a value in the secret cache.
	CacheSet(any, any)

	// CacheGet retrieves a value from the secret cache. It returns the value
	// stored and a boolean value indicating whether any value was stored (that
	// way, a nil value can be stored).
	CacheGet(any) (any, bool)

	// CacheClear delets a value from the secret cache.
	CacheClear(any)
}

// Storage describes the secret from the client side for use with the associated
// storage plugin client. It provides a Cache for storing data with the secret.
type Storage interface {
	Cache

	// Name describing where to store this secret.
	Name() string
}

// Info describes the secret for server side use with rotation or disablement
// plugin clients. It provides a Cache for storing data with the secret.
type Info interface {
	Cache

	// Name describing which secret account to rotation.
	Name() string
}

// Map is the object used to contain a map of keys to secret values. Upon
// rotation, the rotation plugin will return one of these objects containing new
// the new secret values. It is recommended that the keys in such a case be the
// same for every secret for that plugin client and be the most natural names
// associated with each field for that secre value.
//
// These names may be remapped by the rotation business logic to provide
// per-storage names.
type Map map[string]string
