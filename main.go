package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/google/go-github/v42/github"
	"github.com/jamesruan/sodium"
	"golang.org/x/oauth2"
)

const (
	githubAccessKeyKey = "AWS_ACCESS_KEY_ID"     // github secret key for AWS access keys
	githubSecretKeyKey = "AWS_SECRET_ACCESS_KEY" // github secret key for AWS secret keys

	dryRun = false
)

// TODO Move all this into a viper configuration.
var (
	githubAccessToken string // access token read from GITHUB_ACCESS_TOKEN

	maxAge       = 168 * time.Hour // new keys must be newer than this or be rotated
	maxActiveAge = 216 * time.Hour // old keys must be nwere than this or be disabled

	// allowListOrg is the list of orgs I manage keys for
	allowListOrg = map[string]struct{}{
		"zostay": struct{}{},
	}

	// iamUsers is the list of projecs and the IAM users associated with them.
	iamUsers = map[string]string{
		"***REMOVED***":             "***REMOVED***",
		"***REMOVED***":    "***REMOVED***",
		"***REMOVED***":       "***REMOVED***",
		"zostay/periodic-s3-sync": "***REMOVED***",
		"zostay/postfix":          "***REMOVED***",
		"***REMOVED***":    "***REMOVED***",
		"***REMOVED***":      "***REMOVED***",
		"***REMOVED***":             "***REMOVED***",
		"***REMOVED***":       "***REMOVED***",
	}
)

// githubClient connects to the github API client and returns it or returns an
// error.
func githubClient(ctx context.Context, gat string) (*github.Client, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubAccessToken},
	)
	oc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(oc)
	return client, nil
}

// Project is the compiled metadata about each project for which we manage the
// secrets and the associated secret metadata.
type Project struct {
	Owner string // the org/user owning the repo
	Repo  string // the name of the repo

	User string // the username associated with the repo (from local config)

	SecretUpdatedAt time.Time // last update time of the github action secret

	// We cache access key metadata to avoid making multiple calls to IAM that
	// return the same information.

	OldestKey  *iam.AccessKeyMetadata // the oldest IAM key metadata
	NewestKey  *iam.AccessKeyMetadata // the newest IAM key metadata
	keysCached bool                   // true after oldestKey/newestKey are set (possibly to nil)
}

// ptrString is a clone of aws.String() for use with the github API. I suppose I
// should just use aws.String(), but that feels wrong somehow. Does the github
// API provide an analog? I'm too lazy to check.
func ptrString(p *string) string {
	if p != nil {
		return *p
	} else {
		return ""
	}
}

// listReposWithSecrets compile all the Project metadata for projects we manage.
func listReposWithSecrets(ctx context.Context, gc *github.Client) ([]Project, error) {
	nextPage := 1
	pws := make([]Project, 0)
	for {
		opt := &github.RepositoryListOptions{
			ListOptions: github.ListOptions{
				Page: nextPage,
			},
		}
		repos, res, err := gc.Repositories.List(ctx, "", opt)
		if err != nil {
			return nil, fmt.Errorf("unable to list repositories: %w", err)
		}

		for _, repo := range repos {
			owner := ptrString(repo.Owner.Login)
			if _, allowed := allowListOrg[owner]; !allowed {
				continue
			}

			name := ptrString(repo.Name)
			//fmt.Printf("Try %s/%s\n", owner, name)
			secrets, _, err := gc.Actions.ListRepoSecrets(ctx, owner, name, nil)
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
				if secret.Name == githubAccessKeyKey {
					ak = true
					recordEarliestUpdateDate(secret.UpdatedAt.Time)
				}
				if secret.Name == githubSecretKeyKey {
					sk = true
					recordEarliestUpdateDate(secret.UpdatedAt.Time)
				}
				if ak && sk {
					break
				}
			}

			if ak && sk {
				if user, ok := iamUsers[strings.Join([]string{owner, name}, "/")]; ok {
					p := Project{
						Owner: owner,
						Repo:  name,

						User: user,

						SecretUpdatedAt: updated,
					}
					pws = append(pws, p)
				} else {
					fmt.Fprintf(
						os.Stderr,
						"WARNING: project %s/%s has an AWS identity, but is not configured for rotation\n",
						owner, name,
					)
				}
			} else if ak || sk {
				fmt.Fprintf(
					os.Stderr,
					"WARNING: project %s/%s is missing %q or %q in action secrets\n",
					owner, name,
					githubAccessKeyKey, githubSecretKeyKey,
				)
			}
		}

		//fmt.Printf("next: %d last: %d\n", nextPage, res.LastPage)
		if res.LastPage == 0 {
			break
		}

		nextPage = res.NextPage
	}

	return pws, nil
}

// updateSecrets will replace the github action secrets with newly minted
// values.
func updateSecrets(ctx context.Context, gc *github.Client, ak, sk string, p Project) error {
	pubKey, _, err := gc.Actions.GetRepoPublicKey(ctx, p.Owner, p.Repo)
	if err != nil {
		return fmt.Errorf("gc.Actions.GetRepoPublicKey(%q, %q): %w", p.Owner, p.Repo, err)
	}

	keyStr := ptrString(pubKey.Key)
	decKeyBytes, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return fmt.Errorf("base64.StdEncoding.DecodeString(): %w", err)
	}
	keyStr = string(decKeyBytes)

	keyIDStr := ptrString(pubKey.KeyID)

	pkBox := sodium.BoxPublicKey{
		Bytes: sodium.Bytes([]byte(keyStr)),
	}

	akBox := sodium.Bytes([]byte(ak))
	akSealed := akBox.SealedBox(pkBox)
	akEncSealed := base64.StdEncoding.EncodeToString(akSealed)

	skBox := sodium.Bytes([]byte(sk))
	skSealed := skBox.SealedBox(pkBox)
	skEncSealed := base64.StdEncoding.EncodeToString(skSealed)

	if !dryRun {
		akEncSec := &github.EncryptedSecret{
			Name:           githubAccessKeyKey,
			KeyID:          keyIDStr,
			EncryptedValue: akEncSealed,
		}
		_, err = gc.Actions.CreateOrUpdateRepoSecret(ctx, p.Owner, p.Repo, akEncSec)
		if err != nil {
			return fmt.Errorf("gc.Actions.CreateOrUpdateRepoSecret(%q, %q, %q): %w", p.Owner, p.Repo, githubAccessKeyKey, err)
		}

		skEncSec := &github.EncryptedSecret{
			Name:           githubSecretKeyKey,
			KeyID:          keyIDStr,
			EncryptedValue: skEncSealed,
		}
		_, err = gc.Actions.CreateOrUpdateRepoSecret(ctx, p.Owner, p.Repo, skEncSec)
		if err != nil {
			return fmt.Errorf("gc.Actions.CreateOrUpdateRepoSecret(%q, %q, %q): %w", p.Owner, p.Repo, githubAccessKeyKey, err)
		}
	}

	return nil
}

// getOlderApiKeyAge returns the oldest key age for a given IAM user.
func getOlderApiKeyAge(ctx context.Context, svcIam *iam.IAM, p *Project) (time.Time, error) {
	oldKey, newKey, err := getAccessKeys(ctx, svcIam, p)
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
func getAccessKeys(ctx context.Context, svcIam *iam.IAM, p *Project) (*iam.AccessKeyMetadata, *iam.AccessKeyMetadata, error) {
	if p.keysCached {
		return p.OldestKey, p.NewestKey, nil
	}

	ak, err := svcIam.ListAccessKeys(&iam.ListAccessKeysInput{
		UserName: aws.String(p.User),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("svcIam.ListAccessKeys(%q): %w", p.User, err)
	}

	oldKey, newKey := examineKeys(ak.AccessKeyMetadata)
	p.OldestKey = oldKey
	p.NewestKey = newKey
	p.keysCached = true
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

// rotateSecretIam performs all the actions on IAM required to rotate a new
// access key. If there are multiple keys, the oldest one will be deleted. Then
// a new one will be generated. This will not disable a key. This either returns
// two nils and an error or the newly minted access key and secret key and no
// error.
func rotateSecretIam(ctx context.Context, svcIam *iam.IAM, p *Project) (string, string, error) {
	fmt.Printf("Rotating IAM account for %s/%s\n", p.Owner, p.Repo)

	oldKey, newKey, err := getAccessKeys(ctx, svcIam, p)
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
		if !dryRun {
			p.keysCached = false
			_, err := svcIam.DeleteAccessKey(&iam.DeleteAccessKeyInput{
				UserName:    aws.String(p.User),
				AccessKeyId: oldKey.AccessKeyId,
			})
			if err != nil {
				return "", "", fmt.Errorf("svcIam.DeleteAccessKey(): %w", err)
			}
		}
	}

	var accessKey, secretKey string
	if !dryRun {
		p.keysCached = false
		ck, err := svcIam.CreateAccessKey(&iam.CreateAccessKeyInput{
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
func needsRotation(ctx context.Context, svcIam *iam.IAM, p *Project) (bool, error) {
	// is github action secret too old?
	needsRotation := time.Since(p.SecretUpdatedAt) > maxAge

	// Even if the github secret is new, it may be the the github action secret
	// is older than the current IAM secret. Let's check.
	if !needsRotation {
		// the github secret is too old to be the current secret: rotate
		keyAge, err := getOlderApiKeyAge(ctx, svcIam, p)
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
func rotateSecret(ctx context.Context, gc *github.Client, svcIam *iam.IAM, p *Project) error {
	needed, err := needsRotation(ctx, svcIam, p)
	if err != nil {
		return err
	} else if !needed {
		return nil
	}

	ak, sk, err := rotateSecretIam(ctx, svcIam, p)
	if err != nil {
		return fmt.Errorf("rotateSecretIam(): %w", err)
	}

	err = updateSecrets(ctx, gc, ak, sk, *p)
	if err != nil {
		return fmt.Errorf("updateSecrets(): %w", err)
	}

	return nil
}

// rotateSecrets goes through all the projects, determines which have
// outdated keys (i.e., they are older than maxAge) or a mismatch between IAM
// information and github information and performs rotation on those services.
// All github services should have working keys after this operation is
// performed.
func rotateSecrets(ctx context.Context, gc *github.Client, svcIam *iam.IAM, ps []Project) error {
	for i := range ps {
		err := rotateSecret(ctx, gc, svcIam, &ps[i])
		if err != nil {
			return fmt.Errorf("failed to rotate secret: %w", err)
		}
	}

	return nil
}

// disableSecret examines the given project to see if the oldestKey is older
// than the maxActiveAge. If it is older, then the key is disabled in IAM. If
// not, then it is left alone.
func disableSecret(ctx context.Context, svcIam *iam.IAM, p *Project) error {
	okey, _, err := getAccessKeys(ctx, svcIam, p)
	if err != nil {
		return fmt.Errorf("failed to retrieve key information: %w", err)
	}

	createDate := aws.TimeValue(okey.CreateDate)
	needsDisabled := time.Since(createDate) > maxActiveAge
	if !needsDisabled {
		return nil
	}

	fmt.Printf("Disabling old IAM account key for %s/%s\n", p.Owner, p.Repo)

	p.keysCached = false
	_, err = svcIam.UpdateAccessKey(&iam.UpdateAccessKeyInput{
		AccessKeyId: okey.AccessKeyId,
		Status:      aws.String(iam.StatusTypeInactive),
		UserName:    aws.String(p.User),
	})
	if err != nil {
		return fmt.Errorf("failed to update access key status to inactive: %w", err)
	}

	return nil
}

// disableOldSecrets examines all the IAM keys and disables any of the
// non-active keys that have surpassed the maxActiveAge.
func disableOldSecrets(ctx context.Context, svcIam *iam.IAM, ps []Project) error {
	for i := range ps {
		err := disableSecret(ctx, svcIam, &ps[i])
		if err != nil {
			return fmt.Errorf("failed to disable secret: %w", err)
		}
	}

	return nil
}

func main() {
	githubAccessToken = os.Getenv("GITHUB_ACCESS_TOKEN")

	ctx := context.Background()
	gc, err := githubClient(ctx, githubAccessToken)
	if err != nil {
		panic(fmt.Sprintf("unable to authorize with github: %v", err))
	}

	ps, err := listReposWithSecrets(ctx, gc)
	if err != nil {
		panic(fmt.Sprintf("unable list repositories with secrets: %v", err))
	}

	session := session.Must(session.NewSession())
	svcIam := iam.New(session)

	err = rotateSecrets(ctx, gc, svcIam, ps)
	if err != nil {
		panic(fmt.Sprintf("unable to rotate secrets: %v", err))
	}

	err = disableOldSecrets(ctx, svcIam, ps)
	if err != nil {
		panic(fmt.Sprintf("unable to disable expired secrets: %v", err))
	}
}
