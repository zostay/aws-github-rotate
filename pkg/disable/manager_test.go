package disable

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zostay/garotate/pkg/config"
	"github.com/zostay/garotate/pkg/secret"
)

type testClientSecret struct {
	call string
	sec  secret.Info
}

var (
	lastUpdated = time.Date(
		2022, time.April, 1,
		0, 0, 0, 0,
		time.UTC,
	)
)

type testClient struct {
	lastCallSecrets   []testClientSecret
	failLastUpdated   int
	failDisableSecret int
}

func NewTestClient() *testClient {
	return &testClient{
		lastCallSecrets:   []testClientSecret{},
		failLastUpdated:   -1,
		failDisableSecret: -1,
	}
}

func (c *testClient) Name() string {
	return "test"
}

func (c *testClient) LastUpdated(ctx context.Context, s secret.Info) (time.Time, error) {
	c.lastCallSecrets = append(c.lastCallSecrets, testClientSecret{
		call: "LastUpdated",
		sec:  s,
	})
	if c.failLastUpdated == 0 {
		return time.Time{}, fmt.Errorf("last updated bad stuff")
	} else {
		c.failLastUpdated--
		return lastUpdated, nil
	}
}

func (c *testClient) DisableSecret(ctx context.Context, s secret.Info) error {
	c.lastCallSecrets = append(c.lastCallSecrets, testClientSecret{
		call: "DisableSecret",
		sec:  s,
	})
	if c.failDisableSecret == 0 {
		return fmt.Errorf("disable bad stuff")
	} else {
		c.failDisableSecret--
		return nil
	}
}

func TestHappyManagerDryRun(t *testing.T) {
	c := NewTestClient()
	c.failDisableSecret = 0
	m := New(c, 0, true,
		[]config.Secret{
			{SecretName: "James"},
			{SecretName: "John"},
		},
	)

	ctx := context.Background()
	err := m.DisableSecrets(ctx)

	assert.NoError(t, err, "no error on disable secrets dry run")

	callSecrets := []testClientSecret{
		{call: "LastUpdated", sec: &config.Secret{SecretName: "James"}},
		{call: "LastUpdated", sec: &config.Secret{SecretName: "John"}},
	}

	assert.Equal(t, callSecrets, c.lastCallSecrets, "only two calls made")
}

func TestSadManagerDryRun(t *testing.T) {
	c := NewTestClient()
	c.failLastUpdated = 0
	m := New(c, 0, true,
		[]config.Secret{
			{SecretName: "Andrew"},
			{SecretName: "Peter"},
		},
	)

	ctx := context.Background()
	err := m.DisableSecrets(ctx)

	assert.NoError(t, err, "no error on disable secretsd dry run even with errors")

	callSecrets := []testClientSecret{
		{call: "LastUpdated", sec: &config.Secret{SecretName: "Andrew"}},
		{call: "LastUpdated", sec: &config.Secret{SecretName: "Peter"}},
	}

	assert.Equal(t, callSecrets, c.lastCallSecrets, "only two calls made")
}

func TestHappyManager(t *testing.T) {
	c := NewTestClient()
	c.failDisableSecret = 0
	m := New(c, 0, false,
		[]config.Secret{
			{SecretName: "Philip"},
			{SecretName: "Bartholomew"},
		},
	)

	ctx := context.Background()
	err := m.DisableSecrets(ctx)

	assert.NoError(t, err, "no error on disable secrets dry run")

	callSecrets := []testClientSecret{
		{call: "LastUpdated", sec: &config.Secret{SecretName: "Philip"}},
		{call: "DisableSecret", sec: &config.Secret{SecretName: "Philip"}},
		{call: "LastUpdated", sec: &config.Secret{SecretName: "Bartholomew"}},
		{call: "DisableSecret", sec: &config.Secret{SecretName: "Bartholomew"}},
	}

	assert.Equal(t, callSecrets, c.lastCallSecrets, "only two calls made")
}
