package iam

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/zostay/aws-github-rotate/pkg/config"
	"github.com/zostay/aws-github-rotate/pkg/rotate"
)

const (
	// This is the key that will be used to map to the AWS IAM access key when
	// returned from RotateSecret()
	AccessKeyName = "AWS_ACCESS_KEY_ID"

	// This is the key that will be used to map to the AWS IAM secret key when
	// returned from RotateSecret()
	SecretKeyName = "AWS_SECRET_ACCESS_KEY"
)

// gotkeys is the cache used to store the keys cached from a previous AWS fetch.
type gotkeys struct{}

// Client implements both the rotate.Client and disable.Client interfaces.
type Client struct {
	svcIam *iam.IAM
}

// clearCache is a helper for clearing the cache of keys fetched from AWS.
func clearCache(u UserInfo) {
	u.CacheClear(gotkeys{})
}

// getCache is a helper for retrieving the keys cached from a previous AWS
// fetch.
func getCache(u UserInfo) (*iam.AccessKeyMetadata, *iam.AccessKeyMetadata, bool) {
	k, ok := u.CacheGet(gotkeys{})
	if keys, typeOk := k.([]*iam.AccessKeyMetadata); ok && typeOk && len(keys) == 2 {
		return keys[0], keys[1], true
	}
	return nil, nil, false
}

// setCache is a helper for setting the keys just gotten from an AWS fetch.
func setCache(u UserInfo, oldKey, newKey *iam.AccessKeyMetadata) {
	u.CacheSet(gotkeys{}, []*iam.AccessKeyMetadata{oldKey, newKey})
}

// New returns a the client which implements rotate.Client and disable.Client.
func New(svcIam *iam.IAM) *Client {
	return &Client{svcIam}
}

// Prepare does any preparatory work to get the client ready for performing
// either rotation or disablement. As of this writing, this function is a no-op.
func Prepare(
	ctx context.Context,
	users rotate.Users,
) error {
	return nil
}

// LastRotated will return the data of the newest key on the IAM account.
func (c *Client) LastRotated(
	ctx context.Context,
	user rotate.UserInfo,
) (time.Time, error) {
	_, newKey, err := c.getAccessKeys(ctx, user)
	if err != nil {
		return time.Time{}, err
	}

	return aws.TimeValue(newKey.CreateDate), nil
}

// RotateSecret will perform rotation of the secret for the given user. On
// success, this will return the secrets map with two keys, AWS_ACCESS_KEY_ID
// and AWS_SECRET_ACCESS_KEY, set to the newly minted values. The previous
// newest key will now by the old key and any previous key will have been
// removed (at least, that is how IAM works as of this writing).
//
// On error, an empty map is returned with an error.
func (c *Client) RotateSecret(
	ctx context.Context,
	u rotate.UserInfo,
) (rotate.Secrets, error) {
	logger := config.LoggerFrom(ctx).Sugar()
	logger.Infow(
		"rotating IAM account for %s",
		"user", p.Name,
	)

	oldKey, newKey, err := c.getAccessKeys(ctx, u)
	if err != nil {
		return rotate.Secrets{}, fmt.Errorf("getAccessKeys(%q): %w", u.User(), err)
	}

	var oak, nak string
	if oldKey != nil {
		oak = aws.StringValue(oldKey.AccessKeyId)
	}
	if newKey != nil {
		nak = aws.StringValue(newKey.AccessKeyId)
	}

	if oldKey != nil && oak != nak {
		if !r.dryRun {
			clearCache(u)
			_, err := r.svcIam.DeleteAccessKey(&iam.DeleteAccessKeyInput{
				UserName:    aws.String(p.User),
				AccessKeyId: oldKey.AccessKeyId,
			})
			if err != nil {
				return rotate.Secrets{}, fmt.Errorf("svcIam.DeleteAccessKey(): %w", err)
			}
		}
	}

	var accessKey, secretKey string
	clearCache(u)
	ck, err := c.svcIam.CreateAccessKey(&iam.CreateAccessKeyInput{
		UserName: aws.String(p.User),
	})
	if err != nil {
		return rotate.Secrets{}, fmt.Errorf("svcIam.CreateAccessKey(): %w", err)
	}

	accessKey = aws.StringValue(ck.AccessKey.AccessKeyId)
	secretKey = aws.StringValue(ck.AccessKey.SecretAccessKey)

	return rotate.Secrets{
		AccessKeyName: accessKey,
		SecretKeyName: secretKey,
	}, nil
}

// LastUpdated returns the date of the old key associated with the IAM user.
func (c *Client) LastUpdated(
	ctx context.Context,
	user rotate.UserInfo,
) (time.Time, error) {
	oldKey, _, err := c.getAccessKeys(ctx, user)
	if err != nil {
		return time.Time{}, nil
	}

	return aws.TimeValue(oldKey.CreateDate), nil
}

// DisableSecret performs disabling of the old key on AWS IAM.
func (c *Client) DisableSecret(
	ctx context.Context,
	u rotate.UserInfo,
) error {
	okey, _, err := c.getAccessKeys(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to retrieve key information: %w", err)
	}

	logger := config.LoggerFrom(ctx).Sugar()
	logger.Infow(
		"disabling old IAM account key",
		"user", u.User(),
	)

	clearCache(u)
	_, err = c.svcIam.UpdateAccessKey(&iam.UpdateAccessKeyInput{
		AccessKeyId: okey.AccessKeyId,
		Status:      aws.String(iam.StatusTypeInactive),
		UserName:    aws.String(p.User),
	})
	if err != nil {
		return fmt.Errorf("failed to update access key status to inactive: %w", err)
	}

	return nil
}

// getAccessKeys returns the access key metadata for an IAM user. This will
// either return two nils (no key set) and an error or return two metadata
// objects and no error (if one or two keys are set). If only a single key is
// set then the first and second key returned will be equal.
func (c *Client) getAccessKeys(
	ctx context.Context,
	u UserInfo,
) (*iam.AccessKeyMetadata, *iam.AccessKeyMetadata, error) {
	if o, n, ok := getCache(u); ok {
		return o, n, nil
	}

	ak, err := c.svcIam.ListAccessKeys(&iam.ListAccessKeysInput{
		UserName: aws.String(u.User()),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("svcIam.ListAccessKeys(%q): %w", p.User, err)
	}

	oldKey, newKey := examineKeys(ak.AccessKeyMetadata)
	setCache(u, oldKey, newKey)
	return oldKey, newKey, nil
}

// examineKeys will take a list of keys and will return exactly two: the first
// returned is the oldest and the second returned is the newest.
func examineKeys(
	akmds []*iam.AccessKeyMetadata,
) (*iam.AccessKeyMetadata, *iam.AccessKeyMetadata) {
	var (
		oldestTime = time.Now()
		oldestKey  *iam.AccessKeyMetadata
		newestTime time.Time
		newestKey  *iam.AccessKeyMetadata
	)
	for _, akmd := range akmds {
		if akmd.CreateDate != nil && akmd.CreateDate.Before(oldestTime) {
			oldestTime = *akmd.CreateDate
			oldestKey = akmd
		}
		if akmd.CreateDate != nil && akmd.CreateDate.After(newestTime) {
			newestTime = *akmd.CreateDate
			newestKey = akmd
		}
	}

	return oldestKey, newestKey
}
