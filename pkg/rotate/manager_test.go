package rotate

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zostay/garotate/pkg/config"
	"github.com/zostay/garotate/pkg/plugin"
	"github.com/zostay/garotate/pkg/secret"
)

var (
	pastDate = time.Date(
		2022, time.April, 1,
		0, 0, 0, 0,
		time.UTC,
	)
	futureDate = time.Date(
		time.Now().Year()+1, time.April, 1,
		0, 0, 0, 0,
		time.UTC,
	)
)

type testClientSecret struct {
	call string
	sec  secret.Info
}

type testClient struct {
	lastCallSecrets  []testClientSecret
	lastRotated      time.Time
	failLastRotated  int
	failRotateSecret int
}

func NewTestClient() *testClient {
	return &testClient{
		lastCallSecrets:  []testClientSecret{},
		lastRotated:      pastDate,
		failLastRotated:  -1,
		failRotateSecret: -1,
	}
}

func (c *testClient) Name() string {
	return "test"
}

func (c *testClient) Keys() secret.Map {
	return secret.Map{
		"alpha": "",
		"beta":  "",
	}
}

func (c *testClient) LastRotated(ctx context.Context, s secret.Info) (time.Time, error) {
	c.lastCallSecrets = append(c.lastCallSecrets, testClientSecret{
		call: "LastRotated",
		sec:  s,
	})
	if c.failLastRotated == 0 {
		return time.Time{}, fmt.Errorf("last updated bad stuff")
	} else {
		c.failLastRotated--
		return c.lastRotated, nil
	}
}

func (c *testClient) RotateSecret(
	ctx context.Context,
	s secret.Info,
) (secret.Map, error) {
	c.lastCallSecrets = append(c.lastCallSecrets, testClientSecret{
		call: "RotateSecret",
		sec:  s,
	})
	if c.failRotateSecret == 0 {
		return nil, fmt.Errorf("rotate bad stuff")
	} else {
		c.failRotateSecret--
		return nil, nil
	}
}

func TestHappyManagerDryRun(t *testing.T) {
	pluginMgr := plugin.NewManager(
		config.PluginList{},
	)
	c := NewTestClient()
	c.failRotateSecret = 0
	m := New(c, 0, true,
		pluginMgr,
		[]config.Secret{
			{SecretName: "James"},
			{SecretName: "John"},
		},
	)

	ctx := context.Background()
	err := m.RotateSecrets(ctx)

	assert.NoError(t, err, "no error on rotate secrets dry run")

	callSecrets := []testClientSecret{
		{call: "LastRotated", sec: &config.Secret{SecretName: "James"}},
		{call: "LastRotated", sec: &config.Secret{SecretName: "John"}},
	}

	assert.Equal(t, callSecrets, c.lastCallSecrets, "only two calls made")
}

func TestSadManagerDryRun(t *testing.T) {
	pluginMgr := plugin.NewManager(
		config.PluginList{},
	)
	c := NewTestClient()
	c.failLastRotated = 0
	m := New(c, 0, true,
		pluginMgr,
		[]config.Secret{
			{SecretName: "Andrew"},
			{SecretName: "Peter"},
		},
	)

	ctx := context.Background()
	err := m.RotateSecrets(ctx)

	assert.NoError(t, err, "no error on rotate secretsd dry run even with errors")

	callSecrets := []testClientSecret{
		{call: "LastRotated", sec: &config.Secret{SecretName: "Andrew"}},
		{call: "LastRotated", sec: &config.Secret{SecretName: "Peter"}},
	}

	assert.Equal(t, callSecrets, c.lastCallSecrets, "only two calls made")
}

func TestHappyManager(t *testing.T) {
	pluginMgr := plugin.NewManager(
		config.PluginList{},
	)
	c := NewTestClient()
	m := New(c, 0, false,
		pluginMgr,
		[]config.Secret{
			{SecretName: "Philip"},
			{SecretName: "Bartholomew"},
		},
	)

	ctx := context.Background()
	err := m.RotateSecrets(ctx)

	assert.NoError(t, err, "no error on rotate secrets happy run")

	callSecrets := []testClientSecret{
		{call: "LastRotated", sec: &config.Secret{SecretName: "Philip"}},
		{call: "RotateSecret", sec: &config.Secret{SecretName: "Philip"}},
		{call: "LastRotated", sec: &config.Secret{SecretName: "Bartholomew"}},
		{call: "RotateSecret", sec: &config.Secret{SecretName: "Bartholomew"}},
	}

	assert.Equal(t, callSecrets, c.lastCallSecrets, "all four calls made on happy run")
}

func TestSadManagerFailToRotate(t *testing.T) {
	pluginMgr := plugin.NewManager(
		config.PluginList{},
	)
	c := NewTestClient()
	c.failRotateSecret = 0
	m := New(c, 0, false,
		pluginMgr,
		[]config.Secret{
			{SecretName: "Philip"},
			{SecretName: "Bartholomew"},
		},
	)

	ctx := context.Background()
	err := m.RotateSecrets(ctx)

	assert.Error(t, err, "got errors during sad rotation")

	callSecrets := []testClientSecret{
		{call: "LastRotated", sec: &config.Secret{SecretName: "Philip"}},
		{call: "RotateSecret", sec: &config.Secret{SecretName: "Philip"}},
		{call: "LastRotated", sec: &config.Secret{SecretName: "Bartholomew"}},
		{call: "RotateSecret", sec: &config.Secret{SecretName: "Bartholomew"}},
	}

	assert.Equal(t, callSecrets, c.lastCallSecrets, "all four calls made even when sad")
}
