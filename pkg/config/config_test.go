package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareSadDuplicateSecretSet(t *testing.T) {
	c := &Config{
		SecretSets: []SecretSet{
			{Name: "Zebulun"},
			{Name: "Zebulun"},
		},
	}

	err := c.Prepare()
	assert.ErrorContains(t,
		err,
		"is duplicated",
		"should have an error with duplicate secret sets",
	)
}

func TestPrepareSadDuplicateSecret(t *testing.T) {
	c := &Config{
		SecretSets: []SecretSet{
			{
				Name: "Joseph",
				Secrets: []Secret{
					{SecretName: "Gad"},
					{SecretName: "Gad"},
				},
			},
		},
	}

	err := c.Prepare()
	assert.ErrorContains(t,
		err,
		"is repeated twice",
		"should have an error with duplicate secrets",
	)

}

func TestPrepareHappyNormalized(t *testing.T) {
	c := &Config{
		Plugins: PluginList{
			"Naphtali": Plugin{
				Package: "github.com/zostay/garotate/pkg/plugin/iam",
			},
			"Simeon": Plugin{
				Package: "github.com/zostay/garotate/pkg/plugin/github",
			},
		},
		SecretSets: []SecretSet{
			{
				Name: "Asher",
				Secrets: []Secret{
					{
						SecretName: "Reuben",
						Storages: []StorageMap{
							{
								StorageClient: "Simeon",
								StorageName:   "example/project1",
							},
						},
					},
					{
						SecretName: "Levi",
						Storages: []StorageMap{
							{
								StorageClient: "Simeon",
								StorageName:   "example/project2",
							},
						},
					},
				},
			},
		},
	}

	err := c.Prepare()
	assert.NoError(t, err, "no error on happy prepare")

	require.Equal(t, len(c.SecretSets), 1, "secret sets is still len 1")

	ss := &c.SecretSets[0]
	assert.Equal(t, len(ss.Secrets), 2, "secrets is still len 2")

	s0 := &ss.Secrets[0]
	assert.NotNil(t, s0.cache, "first secret cache has been initialized")
	require.Equal(t, len(s0.Storages), 1, "the first key has one storage")
	assert.NotNil(t, s0.Storages[0].cache, "storage on first key cache initialized")

	s1 := &ss.Secrets[1]
	assert.NotNil(t, s1.cache, "second secret cache has been intiializes")
	require.Equal(t, len(s1.Storages), 1, "the second key has one storage")
	assert.NotNil(t, s1.Storages[0].cache, "storage on second key cache initialized")

	// make sure each cache is a separate cache
	assert.Same(t, &s0.cache, &s0.cache, "sameness test control case")
	assert.NotSame(t, &s0.cache, &s1.cache, "each cache is separate 1")
	assert.NotSame(t, &s0.cache, &s0.Storages[0].cache, "each cache is separate 2")
	assert.NotSame(t, &s1.cache, &s1.Storages[0].cache, "each cache is separate 3")
	assert.NotSame(t, &s0.Storages[0].cache, &s1.Storages[0].cache, "each cache is separate 4")

	assert.Equal(t, s0.Name(), s0.SecretName, "Secret Name() is expected value")
	assert.Equal(t, s1.Name(), s1.SecretName, "Secret Name() is expected value")

	assert.Equal(t, s0.Storages[0].Name(), s0.Storages[0].StorageName, "StorageMap Name() is expected value")
	assert.Equal(t, s1.Storages[0].Name(), s1.Storages[0].StorageName, "StorageMap Name() is expected value")
}
