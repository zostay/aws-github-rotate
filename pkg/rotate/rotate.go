package rotate

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/google/go-github/v42/github"
	"github.com/jamesruan/sodium"

	"github.com/zostay/aws-github-rotate/pkg/config"
)

type Projects []ProjectInfo
type Users []UserInfo

type CacheInfo interface {
	CacheSet(any, any)
	CacheGet(any) (any, bool)
	CacheClear(any)
}

type ProjectInfo interface {
	CacheInfo
	Project() string
}

type UserInfo interface {
	CacheInfo
	User() string
}

type Secrets map[string]string

type SaveClient interface {
	LastSaved(context.Context, ProjectInfo, string) (time.Time, error)
	SaveKey(context.Context, ProjectInfo, Secrets) error
}

type RotateClient interface {
	LastRotated(context.Context, UserInfo) (time.Time, error)
	RotateSecret(context.Context, UserInfo) (Secrets, error)
}

type DisableClient interface {
	LastUpdated(context.Context, UserInfo) (time.Time, error)
	DisableSecret(context.Context, UserInfo) error
}

// Rotate is an object capable of rotating a bunch of configured AWS password
// related to github objects and then update the related action secrets.
type Rotate struct {
	gc     *github.Client
	svcIam *iam.IAM

	rotateAfter  time.Duration
	disableAfter time.Duration
	dryRun       bool
	verbose      bool

	Projects map[string]*Project
}

// New constructs a new object to perform password rotation.
func New(
	gc *github.Client,
	svcIam *iam.IAM,
	rotateAfter time.Duration,
	disableAfter time.Duration,
	dryRun bool,
	verbose bool,
	projectMap map[string]*config.Project,
) *Rotate {
	ps := make(map[string]*Project, len(projectMap))
	for k, p := range projectMap {
		ps[k] = &Project{
			Project: p,
		}
	}

	return &Rotate{
		gc:           gc,
		svcIam:       svcIam,
		rotateAfter:  rotateAfter,
		disableAfter: disableAfter,
		dryRun:       dryRun,
		verbose:      verbose,
		Projects:     ps,
	}
}

// RefreshGithubState compiles all the Project metadata for projects we manage.
// It prepares the object for perforing rotations.
func (r *Rotate) RefreshGithubState(ctx context.Context) error {
	nextPage := 1
	for {
		opt := &github.RepositoryListOptions{
			ListOptions: github.ListOptions{
				Page: nextPage,
			},
		}
		repos, res, err := r.gc.Repositories.List(ctx, "", opt)
		if err != nil {
			return fmt.Errorf("unable to list repositories: %w", err)
		}

		for _, repo := range repos {
			owner := repo.Owner.GetLogin()
			repo := repo.GetName()
			name := strings.Join([]string{owner, repo}, "/")
			p, configured := r.Projects[name]
			if !configured {
				continue
			}

			if r.verbose {
				fmt.Printf("Checking state of project %s\n", name)
			}

			//fmt.Printf("Try %s/%s\n", owner, name)
			secrets, _, err := r.gc.Actions.ListRepoSecrets(ctx, owner, repo, nil)
			if err != nil {
				// assume this is a 403 not admin error and try the next
				continue
				//return nil, fmt.Errorf("unable to list repository secrets for %s/%s: %w", owner, name, err)
			}

			var ak, sk bool
			updated := time.Now()
			recordEarliestUpdateDate := func(t time.Time) {
				if t.Before(updated) {
					updated = t
				}
			}
			for _, secret := range secrets.Secrets {
				//fmt.Printf("name = %q\n", secret.Name)
				if secret.Name == p.AccessKey {
					ak = true
					recordEarliestUpdateDate(secret.UpdatedAt.Time)
				}
				if secret.Name == p.SecretKey {
					sk = true
					recordEarliestUpdateDate(secret.UpdatedAt.Time)
				}
				if ak && sk {
					break
				}
			}

			if ak && sk {
				p.SecretUpdatedAt = updated
			} else if ak || sk {
				fmt.Fprintf(
					os.Stderr,
					"WARNING: project %s/%s is missing %q or %q in action secrets\n",
					owner, name,
					p.AccessKey, p.SecretKey,
				)
			}
		}

		//fmt.Printf("next: %d last: %d\n", nextPage, res.LastPage)
		if res.LastPage == 0 {
			break
		}

		nextPage = res.NextPage
	}

	return nil
}

// updateGithub will replace the github action secrets with newly minted values.
func (r *Rotate) updateGithub(ctx context.Context, ak, sk string, p *Project) error {
	pubKey, _, err := r.gc.Actions.GetRepoPublicKey(ctx, p.Owner(), p.Repo())
	if err != nil {
		return fmt.Errorf("gc.Actions.GetRepoPublicKey(%q, %q): %w", p.Owner, p.Repo, err)
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

	akBox := sodium.Bytes([]byte(ak))
	akSealed := akBox.SealedBox(pkBox)
	akEncSealed := base64.StdEncoding.EncodeToString(akSealed)

	skBox := sodium.Bytes([]byte(sk))
	skSealed := skBox.SealedBox(pkBox)
	skEncSealed := base64.StdEncoding.EncodeToString(skSealed)

	if r.verbose {
		fmt.Printf(" - updating github action secret %q\n", p.AccessKey)
		fmt.Printf(" - updating github action secret %q\n", p.SecretKey)
	}

	if !r.dryRun {
		akEncSec := &github.EncryptedSecret{
			Name:           p.AccessKey,
			KeyID:          keyIDStr,
			EncryptedValue: akEncSealed,
		}
		_, err = r.gc.Actions.CreateOrUpdateRepoSecret(ctx, p.Owner(), p.Repo(), akEncSec)
		if err != nil {
			return fmt.Errorf("gc.Actions.CreateOrUpdateRepoSecret(%q, %q, %q): %w", p.Owner(), p.Repo(), p.AccessKey, err)
		}

		skEncSec := &github.EncryptedSecret{
			Name:           p.SecretKey,
			KeyID:          keyIDStr,
			EncryptedValue: skEncSealed,
		}
		_, err = r.gc.Actions.CreateOrUpdateRepoSecret(ctx, p.Owner(), p.Repo(), skEncSec)
		if err != nil {
			return fmt.Errorf("gc.Actions.CreateOrUpdateRepoSecret(%q, %q, %q): %w", p.Owner(), p.Repo(), p.SecretKey, err)
		}

		p.TouchGithub()
	}

	return nil
}

// getOlderApiKeyAge returns the oldest key age for a given IAM user.
func (r *Rotate) getOlderApiKeyAge(ctx context.Context, p *Project) (time.Time, error) {
	oldKey, newKey, err := r.getAccessKeys(ctx, p)
	if err != nil {
		return time.Time{}, fmt.Errorf("getAccessKeys(%s): %w", p.User, err)
	}

	if oldKey == nil {
		return aws.TimeValue(newKey.CreateDate), nil
	}

	return aws.TimeValue(oldKey.CreateDate), nil
}

// getAccessKeys returns the access key metadata for an IAM user. This will
// either return two nils (no key set) and an error or return two metadata
// objects and no error (if one or two keys are set). If only a single key is
// set then the first and second key returned will be equal.
func (r *Rotate) getAccessKeys(ctx context.Context, p *Project) (*iam.AccessKeyMetadata, *iam.AccessKeyMetadata, error) {
	if o, n, err := p.GetAWSKeyCache(); err == nil {
		return o, n, nil
	}

	ak, err := r.svcIam.ListAccessKeys(&iam.ListAccessKeysInput{
		UserName: aws.String(p.User),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("svcIam.ListAccessKeys(%q): %w", p.User, err)
	}

	oldKey, newKey := examineKeys(ak.AccessKeyMetadata)
	p.SetAWSKeyCache(oldKey, newKey)
	return oldKey, newKey, nil
}

// examineKeys will take a list of keys and will return exactly two: the first
// returned is the oldest and the second returned is the newest.
func examineKeys(akmds []*iam.AccessKeyMetadata) (*iam.AccessKeyMetadata, *iam.AccessKeyMetadata) {
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

// RotateAWSSecret performs all the actions on IAM required to rotate a new
// access key. If there are multiple keys, the oldest one will be deleted. Then
// a new one will be generated. This will not disable a key. This either returns
// two nils and an error or the newly minted access key and secret key and no
// error.
func (r *Rotate) rotateAWSSecret(ctx context.Context, p *Project) (string, string, error) {
	if r.verbose {
		fmt.Print(" - ")
	}
	fmt.Printf("rotating IAM account for %s\n", p.Name)

	oldKey, newKey, err := r.getAccessKeys(ctx, p)
	if err != nil {
		return "", "", fmt.Errorf("getAccessKeys(%q): %w", p.User, err)
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
			p.ClearAWSKeyCache()
			_, err := r.svcIam.DeleteAccessKey(&iam.DeleteAccessKeyInput{
				UserName:    aws.String(p.User),
				AccessKeyId: oldKey.AccessKeyId,
			})
			if err != nil {
				return "", "", fmt.Errorf("svcIam.DeleteAccessKey(): %w", err)
			}
		}
	}

	var accessKey, secretKey string
	if !r.dryRun {
		p.keysCached = false
		ck, err := r.svcIam.CreateAccessKey(&iam.CreateAccessKeyInput{
			UserName: aws.String(p.User),
		})
		if err != nil {
			return "", "", fmt.Errorf("svcIam.CreateAccessKey(): %w", err)
		}

		accessKey = aws.StringValue(ck.AccessKey.AccessKeyId)
		secretKey = aws.StringValue(ck.AccessKey.SecretAccessKey)
	} else {
		accessKey = "dryrunfakeaccesskey"
		secretKey = "dryrunfakesecretkey"
	}

	return accessKey, secretKey, nil
}

// needsRotation returns true if the named secret requires rotation.
func (r *Rotate) NeedsRotation(ctx context.Context, p *Project) (bool, error) {
	// is github action secret too old?
	needsRotation := time.Since(p.SecretUpdatedAt) > r.rotateAfter

	if r.verbose && needsRotation {
		fmt.Printf(" - Secret updated %v has not been updated in more than %v\n", p.SecretUpdatedAt, r.rotateAfter)
	}

	// Even if the github secret is new, it may be the the github action secret
	// is older than the current IAM secret. Let's check.
	if !needsRotation {
		// the github secret is too old to be the current secret: rotate
		keyAge, err := r.getOlderApiKeyAge(ctx, p)
		if err != nil {
			return false, fmt.Errorf("failed to retrieve key age: %w", err)
		}

		// is the github copy of the secret too old?
		needsRotation = p.SecretUpdatedAt.Before(keyAge)
	}

	return needsRotation, nil
}

// rotateSecret rotates a single project's secret in IAM and then updates the
// github action secret keys with the newly minted access key and secret key.
func (r *Rotate) rotateSecret(ctx context.Context, p *Project) error {
	needed, err := r.NeedsRotation(ctx, p)
	if err != nil {
		return err
	} else if !needed {
		return nil
	}

	ak, sk, err := r.rotateAWSSecret(ctx, p)
	if err != nil {
		return fmt.Errorf("rotateSecretIam(): %w", err)
	}

	err = r.updateGithub(ctx, ak, sk, p)
	if err != nil {
		return fmt.Errorf("updateSecrets(): %w", err)
	}

	return nil
}

// RotateSecrets goes through all the projects, determines which have
// outdated keys (i.e., they are older than maxAge) or a mismatch between IAM
// information and github information and performs rotation on those services.
// All github services should have working keys after this operation is
// performed.
func (r *Rotate) RotateSecrets(ctx context.Context) error {
	for k := range r.Projects {
		if r.verbose {
			fmt.Printf("Consider for rotation, project %s\n", r.Projects[k].Name)
		}
		err := r.rotateSecret(ctx, r.Projects[k])
		if err != nil {
			return fmt.Errorf("failed to rotate secret: %w", err)
		}
	}

	return nil
}

// disableAWSSecret examines the given project to see if the oldestKey is older
// than the DisableAfter time. If it is older, then the key is disabled in IAM.
// If not, then it is left alone.
func (r *Rotate) disableAWSSecret(ctx context.Context, p *Project) error {
	okey, _, err := r.getAccessKeys(ctx, p)
	if err != nil {
		return fmt.Errorf("failed to retrieve key information: %w", err)
	}

	createDate := aws.TimeValue(okey.CreateDate)
	needsDisabled := time.Since(createDate) > r.rotateAfter+r.disableAfter
	if !needsDisabled {
		return nil
	}

	if r.verbose && needsDisabled {
		fmt.Printf(" - Secret updated %v has not been updated in more than %v\n", p.SecretUpdatedAt, r.rotateAfter+r.disableAfter)
	}

	if r.verbose {
		fmt.Print(" - ")
	}
	fmt.Printf("disabling old IAM account key for project %s\n", p.Name)

	p.ClearAWSKeyCache()
	_, err = r.svcIam.UpdateAccessKey(&iam.UpdateAccessKeyInput{
		AccessKeyId: okey.AccessKeyId,
		Status:      aws.String(iam.StatusTypeInactive),
		UserName:    aws.String(p.User),
	})
	if err != nil {
		return fmt.Errorf("failed to update access key status to inactive: %w", err)
	}

	return nil
}

// DisableOldSecrets examines all the IAM keys and disables any of the
// non-active keys that have surpassed the maxActiveAge.
func (r *Rotate) DisableOldSecrets(ctx context.Context) error {
	for k := range r.Projects {
		if r.verbose {
			fmt.Printf("Consider for disablement, project %s\n", r.Projects[k].Name)
		}
		err := r.disableAWSSecret(ctx, r.Projects[k])
		if err != nil {
			return fmt.Errorf("failed to disable secret: %w", err)
		}
	}

	return nil
}
