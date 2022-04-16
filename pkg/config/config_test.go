package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
		Plugins: []Plugin{
			"Naphtali": Plugin{
				Package: "github.com/zostay/garotate/pkg/plugin/iam",
			},
		},
		SecretSets: []SecretSet{
			{
				Name: "Asher",
				Secrets: []Secret{
					{SecretName: "Reuben"},
				},
			},
		},
	}
}
