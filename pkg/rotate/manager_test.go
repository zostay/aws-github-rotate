package rotate

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	recentButPastDate = time.Date(
		time.Now().Year(), time.Now().Month(), time.Now().Day(),
		time.Now().Hour()-12, 0, 0, 0,
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
		return secret.Map{
			"alpha": "one",
			"beta":  "two",
		}, nil
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

type testStorage struct {
	storage       map[string]map[string]string
	lastSaved     time.Time
	failLastSaved int
}

func (t *testStorage) Name() string {
	return "test storage"
}

func (t *testStorage) testStorage(store secret.Storage) map[string]string {
	if t.storage == nil {
		t.storage = make(map[string]map[string]string)
	}

	if _, ok := t.storage[store.Name()]; !ok {
		t.storage[store.Name()] = make(map[string]string)
	}

	return t.storage[store.Name()]
}

func (t *testStorage) LastSaved(
	ctx context.Context,
	store secret.Storage,
	key string,
) (time.Time, error) {
	if t.failLastSaved == 0 {
		return time.Time{}, fmt.Errorf("last saved bad stuff")
	} else {
		t.failLastSaved--
	}

	ts := t.testStorage(store)
	if _, found := ts[key]; !found {
		return time.Time{}, secret.ErrKeyNotFound
	}
	return t.lastSaved, nil
}

func (t *testStorage) SaveKeys(
	ctx context.Context,
	store secret.Storage,
	ss secret.Map,
) error {
	ts := t.testStorage(store)

	for k, v := range ss {
		ts[k] = v
	}

	return nil
}

type testRotationBuilder struct{}

func (b *testRotationBuilder) Build(
	ctx context.Context,
	c *config.Plugin,
) (plugin.Instance, error) {
	return NewTestClient(), nil
}

type testStorageBuilder struct{}

func (b *testStorageBuilder) Build(
	ctx context.Context,
	c *config.Plugin,
) (plugin.Instance, error) {
	failLastSaved := -1
	if fls, ok := c.Options["failLastSaved"]; ok {
		flsi, ok := fls.(int)
		if !ok {
			panic("test configuration is wrong in the failLastSaved key")
		}
		failLastSaved = flsi
	}
	return &testStorage{
		lastSaved:     futureDate,
		failLastSaved: failLastSaved,
	}, nil
}

func init() {
	plugin.Register("testStorage", new(testStorageBuilder))
	plugin.Register("testRotation", new(testRotationBuilder))
}

func TestHappyRotationStorage(t *testing.T) {
	pluginMgr := plugin.NewManager(
		config.PluginList{
			"test": config.Plugin{
				Name:    "test",
				Package: "testStorage",
			},
		},
	)

	tstoreSettings := []testStorage{
		{
			storage:   nil,
			lastSaved: futureDate,
		},
		{
			storage: map[string]map[string]string{
				"Matthew": map[string]string{
					"omega": "hunter2",
					"beta":  "hunter",
				},
			},
			lastSaved: pastDate,
		},
	}

	for _, tss := range tstoreSettings {
		c := NewTestClient()
		c.lastRotated = recentButPastDate
		m := New(c, 24*time.Hour, false,
			pluginMgr,
			[]config.Secret{
				{
					SecretName: "Matthew",
					Storages: []config.StorageMap{
						{
							StorageClient: "test",
							StorageName:   "Matthew",
							Keys: config.KeyMap{
								"alpha": "omega",
							},
						},
					},
				},
			},
		)

		// cheating: we trigger the lazy construction here so we can manipulate
		// the state of the test object. This is highly dependent on how plugin
		// instance caching works.
		ctx := context.Background()
		store, err := pluginMgr.Instance(ctx, "test")

		assert.NoError(t, err, "got no errors retrieving storage instance")

		tstore, ok := store.(*testStorage)
		require.True(t, ok, "type coercion to testStorage works")

		// ensure we have a clean store before rotation
		tstore.storage = tss.storage
		tstore.lastSaved = tss.lastSaved

		err = m.RotateSecrets(ctx)

		assert.NoError(t, err, "got no errors during rotation")

		assert.Equal(t, tstore.storage,
			map[string]map[string]string{
				"Matthew": map[string]string{
					"omega": "one",
					"beta":  "two",
				},
			},
			"expected keys found in store",
		)
	}
}

func TestHappyRotationStorageSkipping(t *testing.T) {
	pluginMgr := plugin.NewManager(
		config.PluginList{
			"test": config.Plugin{
				Name:    "test",
				Package: "testStorage",
			},
		},
	)

	fixtures := []struct {
		moniker string
		store   testStorage
	}{
		{
			moniker: "with existing storage",
			store: testStorage{
				storage: map[string]map[string]string{
					"Matthew": map[string]string{
						"omega": "hunter2",
						"beta":  "hunter",
					},
				},
				lastSaved: futureDate,
			},
		},
	}

	for _, fixture := range fixtures {
		tss := fixture.store
		c := NewTestClient()
		c.lastRotated = recentButPastDate
		m := New(c, 24*time.Hour, false,
			pluginMgr,
			[]config.Secret{
				{
					SecretName: "Matthew",
					Storages: []config.StorageMap{
						{
							StorageClient: "test",
							StorageName:   "Matthew",
							Keys: config.KeyMap{
								"alpha": "omega",
							},
						},
					},
				},
			},
		)

		// cheating: we trigger the lazy construction here so we can manipulate
		// the state of the test object. This is highly dependent on how plugin
		// instance caching works.
		ctx := context.Background()
		store, err := pluginMgr.Instance(ctx, "test")

		assert.NoErrorf(t, err,
			"got no errors retrieving storage instance [%s]", fixture.moniker)

		tstore, ok := store.(*testStorage)
		require.Truef(t, ok, "type coercion to testStorage works [%s]",
			fixture.moniker)

		// ensure we have a clean store before rotation
		tstore.storage = tss.storage
		tstore.lastSaved = tss.lastSaved

		err = m.RotateSecrets(ctx)

		assert.NoErrorf(t, err, "got no errors during rotation [%s]",
			fixture.moniker)

		assert.Equalf(t, tss.storage, tstore.storage,
			"keys in storage are unchanged [%s]", fixture.moniker)
	}
}

func TestSadRotationMissingStorage(t *testing.T) {
	pluginMgr := plugin.NewManager(
		config.PluginList{},
	)

	c := NewTestClient()
	c.lastRotated = recentButPastDate
	m := New(c, 24*time.Hour, false,
		pluginMgr,
		[]config.Secret{
			{
				SecretName: "Thomas",
				Storages: []config.StorageMap{
					{
						StorageClient: "test",
						StorageName:   "Thomas",
						Keys: config.KeyMap{
							"alpha": "omega",
						},
					},
				},
			},
		},
	)

	// cheating: we trigger the lazy construction here so we can manipulate
	// the state of the test object. This is highly dependent on how plugin
	// instance caching works.
	ctx := context.Background()

	err := m.RotateSecrets(ctx)

	// TODO It would be nice to test for log messages.

	assert.NoError(t, err, "error occurrd, but only logged")
}

func TestSadRotationBrokenStorage(t *testing.T) {
	pluginMgr := plugin.NewManager(
		config.PluginList{
			"test": config.Plugin{
				Name:    "test",
				Package: "testStorage",
				Options: map[string]any{
					"failLastSaved": 0,
				},
			},
		},
	)

	c := NewTestClient()
	c.lastRotated = recentButPastDate
	m := New(c, 24*time.Hour, false,
		pluginMgr,
		[]config.Secret{
			{
				SecretName: "Thomas",
				Storages: []config.StorageMap{
					{
						StorageClient: "test",
						StorageName:   "Thomas",
						Keys: config.KeyMap{
							"alpha": "omega",
						},
					},
				},
			},
		},
	)

	// cheating: we trigger the lazy construction here so we can manipulate
	// the state of the test object. This is highly dependent on how plugin
	// instance caching works.
	ctx := context.Background()

	err := m.RotateSecrets(ctx)

	// TODO It would be nice to test for log messages.

	assert.NoError(t, err, "error occurrd, but only logged")
}

func TestSadRotationStorageWrongType(t *testing.T) {
	pluginMgr := plugin.NewManager(
		config.PluginList{
			"test": config.Plugin{
				Name:    "test",
				Package: "testRotation",
			},
		},
	)

	c := NewTestClient()
	c.lastRotated = recentButPastDate
	m := New(c, 24*time.Hour, false,
		pluginMgr,
		[]config.Secret{
			{
				SecretName: "Thomas",
				Storages: []config.StorageMap{
					{
						StorageClient: "test",
						StorageName:   "Thomas",
						Keys: config.KeyMap{
							"alpha": "omega",
						},
					},
				},
			},
		},
	)

	// cheating: we trigger the lazy construction here so we can manipulate
	// the state of the test object. This is highly dependent on how plugin
	// instance caching works.
	ctx := context.Background()

	err := m.RotateSecrets(ctx)

	// TODO It would be nice to test for log messages.

	assert.NoError(t, err, "error occurrd, but only logged")
}
