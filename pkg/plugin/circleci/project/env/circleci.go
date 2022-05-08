// Package circleci provides a plugin that implemetns the
// rotate.Storage interface for storing keys in CircleCI environment variables.
package env

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/zostay/garotate/pkg/secret"
)

// envVarsSeen is the key used for caching.
type envVarsSeen struct{}

// Client implements the rotate.
// SaveClient interface for storing keys following
// rotation.
//
// To use this client, a CIRCLECI_TOKEN environment variable must be set to
// CircleCI access token.
type Client struct {
	hc    *http.Client
	token string
}

// setCachedEnvVars sets the environment variables that are known to have
// exist from a recent call to LastSaved or SaveKeys.
func setCachedEnvVars(c secret.Cache, vars map[string]struct{}) {
	c.CacheSet(envVarsSeen{}, vars)
}

// getCachedEnvVars is a helper that retrieves the environment variables that
// are known to exist from a previous call to LastSaved or SaveKeys.
func getCachedEnvVars(c secret.Cache) (map[string]struct{}, bool) {
	t, ok := c.CacheGet(envVarsSeen{})
	if vars, typeOk := t.(map[string]struct{}); ok && typeOk {
		return vars, true
	}
	return nil, false
}

// Name returns "CircleCI environment variables"
func (c *Client) Name() string {
	return "CircleCI environment variables"
}

// LastSaved always returns an error if the key does not exist, but returns
// time.Now() if it does because CircleCI provides no facilities for determining
// age.
func (c *Client) LastSaved(
	ctx context.Context,
	store secret.Storage,
	key string,
) (time.Time, error) {
	if vars, ok := getCachedEnvVars(store); ok {
		if _, found := vars[key]; found {
			return time.Now(), nil
		} else {
			return time.Time{}, secret.ErrKeyNotFound
		}
	}

	req, err := http.NewRequest(
		"GET",
		defaultHost+defaultRestEndpoint+"/project/"+store.Name()+"/envvar",
		nil,
	)
	req.Header.Add("Circle-Token", c.token)
	res, err := c.hc.Do(req)
	if err != nil {
		return time.Time{}, err
	}

	if res.StatusCode != 200 {
		return time.Time{}, fmt.Errorf("unexpected status code: %d",
			res.StatusCode)
	}

	type envVarResponse struct {
		Items []struct {
			Name  string
			Value string
		}
		NextPageToken string
	}

	var envVarRes envVarResponse
	d := json.NewDecoder(res.Body)
	err = d.Decode(&envVarRes)
	if err != nil {
		return time.Time{}, err
	}

	// TODO Add handling of NextPageToken

	evs := envVarRes.Items

	found := make(map[string]struct{}, len(evs))
	for _, k := range evs {
		found[k.Name] = struct{}{}
	}

	setCachedEnvVars(store, found)

	if _, ok := found[key]; ok {
		return time.Now(), nil
	}

	return time.Time{}, secret.ErrKeyNotFound
}

// SaveKeys saves each of the secrets given into the environment variables
// for the project.
func (c *Client) SaveKeys(
	ctx context.Context,
	store secret.Storage,
	ss secret.Map,
) error {
	found, isCached := getCachedEnvVars(store)
	if !isCached {
		found = make(map[string]struct{}, len(ss))
	}

	for key, sec := range ss {
		found[key] = struct{}{}

		secretJson, err := json.Marshal(map[string]string{
			"name":  key,
			"value": sec,
		})
		if err != nil {
			return err
		}

		secret := bytes.NewReader(secretJson)

		req, err := http.NewRequest(
			"POST",
			defaultHost+defaultRestEndpoint+"/project/"+store.Name()+"/envvar",
			secret,
		)
		if err != nil {
			return err
		}

		req.Header.Add("Circle-Token", c.token)
		req.Header.Add("Content-Type", "application/json")

		res, err := c.hc.Do(req)
		if err != nil {
			return err
		}

		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return fmt.Errorf("unexpected status code %d", res.StatusCode)
		}
	}

	setCachedEnvVars(store, found)

	return nil
}
