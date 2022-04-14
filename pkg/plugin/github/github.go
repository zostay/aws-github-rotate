package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v42/github"
	"github.com/jamesruan/sodium"
	"github.com/zostay/aws-github-rotate/pkg/config"
	"github.com/zostay/aws-github-rotate/pkg/secret"
)

// secretUpdateAt is the container for last updated date's cache keys.
type secretUpdatedAt struct {
	name string
}

// Client implements the rotate.SaveClient interface for storing keys following
// rotation.
//
// To use this client, a GITHUB_TOKEN environment variable must be set to a
// github access token with adequate permissions to update action secrets.
type Client struct {
	gc *github.Client
}

// parts splits a project name into the owner/repo form used for github
// projects.
func parts(s secret.Storage) (string, string) {
	o, r, _ := strings.Cut(s.Name(), "/")
	return o, r
}

// setCachedKeyTime is a helper that stores the cached secret UpdatedAt value.
func setCachedKeyTime(c secret.Cache, secret string, upd time.Time) {
	c.CacheSet(secretUpdatedAt{secret}, upd)
}

// getCachedKeyTime is a helper that retrieves the cached secret UpdatedAt
// value.
func getCachedKeyTime(c secret.Cache, secret string) (time.Time, bool) {
	t, ok := c.CacheGet(secretUpdatedAt{secret})
	if time, typeOk := t.(time.Time); ok && typeOk {
		return time, true
	}
	return time.Time{}, false
}

// touchCachedKeyTime is a helper that sets the cached secret UpdatedAt value to
// now.
func touchCachedKeyTime(c secret.Cache, sec string) {
	setCachedKeyTime(c, sec, time.Now())
}

// Name returns "github action secrets"
func (c *Client) Name() string {
	return "github action secrets"
}

// LastSaved checks for the given key on the given project to see when it was
// last saved. It will return that value, if it has been stored previously. If
// it has not been stored previously, it returns the zero value.
func (c *Client) LastSaved(
	ctx context.Context,
	store secret.Storage,
	key string,
) (time.Time, error) {
	if upd, ok := getCachedKeyTime(store, key); ok {
		return upd, nil
	}

	owner, repo := parts(store)
	logger := config.LoggerFrom(ctx).Sugar()
	gsecs, _, err := c.gc.Actions.ListRepoSecrets(ctx, owner, repo, nil)
	if err != nil {
		logger.Errorw(
			"project is missing secret",
			"client", c.Name(),
			"store", store.Name(),
			"secret", key,
		)
		return time.Time{}, nil
	}

	var upd time.Time
	for _, gsec := range gsecs.Secrets {
		setCacheKeyTime(store, gsec.Name, gsec.UpdatedAt.Time)
		if gsec.Name == key {
			upd = gsec.UpdatedAt.Time
		}
	}

	return upd, nil
}

// SaveKeys saves each of the secrets given in the project.
func (c *Client) SaveKeys(
	ctx context.Context,
	store secret.Storage,
	ss secret.Map,
) error {
	owner, repo := parts(store.Name())
	pubKey, _, err := c.gc.Actions.GetRepoPublicKey(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to retrieve github project public key for project %q: %w", store.Name(), err)
	}

	keyStr := pubKey.GetKey()
	decKeyBytes, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return fmt.Errorf("failed to decode github project public key string for project %q: %w", store.Name, err)
	}
	keyStr = string(decKeyBytes)

	keyIDStr := pubKey.GetKeyID()

	pkBox := sodium.BoxPublicKey{
		Bytes: sodium.Bytes([]byte(keyStr)),
	}

	logger := config.LoggerFrom(ctx).Sugar()
	for key, sec := range ss {
		keyBox := sodium.Bytes([]byte(sec))
		keySealed := keyBox.SealedBox(pkBox)
		keyEncSealed := base64.StdEncoding.EncodeToString(keySealed)

		logger.Debugw(
			"updating github action secret",
			"storage", store.Name(),
			"secret", key,
		)

		encSec := &github.EncryptedSecret{
			Name:           key,
			KeyID:          keyIDStr,
			EncryptedValue: keyEncSealed,
		}
		_, err = c.gc.Actions.CreateOrUpdateRepoSecret(ctx, owner, repo, akEncSec)
		if err != nil {
			return fmt.Errorf("failed to create or update github action secret named %q for project %q: %w", key, store.Name(), err)
		}

		touchCachedKeyTime(store, key)
	}

	return nil
}
