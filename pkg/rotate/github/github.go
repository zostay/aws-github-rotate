package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/jamesruan/sodium"
	"github.com/zostay/aws-github-rotate/pkg/config"
	"github.com/zostay/aws-github-rotate/pkg/rotate"
)

type secretUpdatedAt struct {
	name string
}

type projectMap map[string]rotate.ProjectInfo

type Client struct {
	gc       *github.Client
	projects projectMap
}

func projectsMap(ps rotate.Projects) ProjectMap {
	pm := make(ProjectMap, len(ps))
	for i, p := range ps {
		pm[p.Name()] = p
	}
	return pm
}

func parts(p rotate.ProjectInfo) (string, string) {
	o, r, _ := strings.Cut(p.Name(), "/")
	return o, r
}

func setCachedKeyTime(p rotate.ProjectInfo, secret string, upd time.Time) {
	p.CacheSet(secretUpdatedAt{secret}, upd)
}

func getCachedKeyTime(p rotate.ProjectInfo, secret string) (time.Time, bool) {
	t, ok := p.CacheGet(secretUpdatedAt{secret})
	if time, typeOk := t.(time.Time); ok && typeOk {
		return time, true
	}
	return time.Time{}, false
}

func touchCachedKeyTime(p rotate.ProjectInfo, secret string) {
	setCachedKeyTime(secret, time.Now())
}

func (c *Client) LastSaved(
	ctx context.Context,
	p rotate.ProjectInfo,
	key string,
) (time.Time, error) {
	if upd, ok := getCachedKeyTime(p, key); ok {
		return upd, nil
	}

	owner, repo := parts(p.Name())
	logger := config.LoggerFrom(ctx).Sugar()
	secrets, _, err := c.gc.Actions.ListRepoSecrets(ctx, owner, repo, nil)
	if err != nil {
		logger.Errorw(
			"project is missing secret",
			"project", p.Name(),
			"secret", key,
		)
		return time.Time{}, nil
	}

	var upd time.Time
	for _, secret := range secrets.Secrets {
		setCacheKeyTime(p, secret.Name, secret.UpdatedAt.Time)
		if secret.Name == key {
			upd = secret.UpdatedAt.Time
		}
	}

	return upd, nil
}

func (c *Client) SaveKey(
	ctx context.Context,
	p rotate.ProjectInfo,
	ss rotate.Secrets,
) error {
	owner, repo := parts(p.Name())
	pubKey, _, err := c.gc.Actions.GetRepoPublicKey(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("gc.Actions.GetRepoPublicKey(%q, %q): %w", owner, repo, err)
	}

	keyStr := pubKey.GetKey()
	decKeyBytes, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return fmt.Errorf("base64.StdEncoding.DecodeString(): %w", err)
	}
	keyStr = string(decKeyBytes)

	keyIDStr := pubKey.GetKeyID()

	pkBox := sodium.BoxPublicKey{
		Bytes: sodium.Bytes([]byte(keyStr)),
	}

	logger := config.LoggerFrom(ctx).Sugar()
	for key, secret := range ss {
		keyBox := sodium.Bytes([]byte(secret))
		keySealed := keyBox.SealedBox(pkBox)
		keyEncSealed := base64.StdEncoding.EncodeToString(keySealed)

		logger.Debugw(
			"updating github action secret",
			"project", p.Name(),
			"secret", key,
		)

		encSec := &github.EncryptedSecret{
			Name:           key,
			KeyID:          keyIDStr,
			EncryptedValue: keyEncSealed,
		}
		_, err = r.gc.Actions.CreateOrUpdateRepoSecret(ctx, p.Owner(), p.Repo(), akEncSec)
		if err != nil {
			return fmt.Errorf("gc.Actions.CreateOrUpdateRepoSecret(%q, %q, %q): %w", owner, repo, key, err)
		}

		touchCachedKeyTime(p, key)
	}

	return nil
}
