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
	githubAccessKeyKey = "AWS_ACCESS_KEY_ID"
	githubSecretKeyKey = "AWS_SECRET_ACCESS_KEY"

	dryRun = false
)

var (
	githubAccessToken string

	maxAge       = 336 * time.Hour
	maxActiveAge = 168 * time.Hour
	allowListOrg = map[string]struct{}{
		"zostay": struct{}{},
	}

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

func githubClient(ctx context.Context, gat string) (*github.Client, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubAccessToken},
	)
	oc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(oc)
	return client, nil
}

type Project struct {
	Owner string
	Repo  string

	User string

	SecretUpdatedAt time.Time
}

func ptrString(p *string) string {
	if p != nil {
		return *p
	} else {
		return ""
	}
}

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

func getOlderApiKeyAge(ctx context.Context, svcIam *iam.IAM, user string) (time.Time, error) {
	oldKey, newKey, err := getAccessKeys(ctx, svcIam, user)
	if err != nil {
		return time.Time{}, fmt.Errorf("getAccessKeys(%s): %w", user, err)
	}

	if oldKey == nil {
		return aws.TimeValue(newKey.CreateDate), nil
	}

	return aws.TimeValue(oldKey.CreateDate), nil
}

func getAccessKeys(ctx context.Context, svcIam *iam.IAM, user string) (*iam.AccessKeyMetadata, *iam.AccessKeyMetadata, error) {
	ak, err := svcIam.ListAccessKeys(&iam.ListAccessKeysInput{
		UserName: aws.String(user),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("svcIam.ListAccessKeys(%q): %w", user, err)
	}

	oldKey, newKey := examineKeys(ak.AccessKeyMetadata)
	return oldKey, newKey, nil
}

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

func rotateSecretIam(ctx context.Context, svcIam *iam.IAM, p Project) (string, string, error) {
	fmt.Printf("Rotating IAM account for %s/%s\n", p.Owner, p.Repo)

	oldKey, newKey, err := getAccessKeys(ctx, svcIam, p.User)
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

func rotateSecret(ctx context.Context, gc *github.Client, svcIam *iam.IAM, p Project) error {
	ak, sk, err := rotateSecretIam(ctx, svcIam, p)
	if err != nil {
		return fmt.Errorf("rotateSecretIam(): %w", err)
	}

	err = updateSecrets(ctx, gc, ak, sk, p)
	if err != nil {
		return fmt.Errorf("updateSecrets(): %w", err)
	}

	return nil
}

func rotateOldSecrets(ctx context.Context, gc *github.Client, ps []Project) error {
	session := session.Must(session.NewSession())
	svcIam := iam.New(session)

	for _, p := range ps {
		if time.Since(p.SecretUpdatedAt) > maxAge {
			// secret is too old: rotate
			err := rotateSecret(ctx, gc, svcIam, p)
			if err != nil {
				return fmt.Errorf("failed to rotate old secret: %w", err)
			}
			// fmt.Println("Quitting 1 for testing.")
			// os.Exit(0)
		} else {
			// the github secret is too old to be the current secret: rotate
			keyAge, err := getOlderApiKeyAge(ctx, svcIam, p.User)
			if err != nil {
				return fmt.Errorf("failed to retrieve key age: %w", err)
			}

			if p.SecretUpdatedAt.Before(keyAge) {
				err := rotateSecret(ctx, gc, svcIam, p)
				if err != nil {
					return fmt.Errorf("failed to rotate expired secret: %w", err)
				}
				// fmt.Println("Quitting 2 for testing.")
				// os.Exit(0)
			}
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

	err = rotateOldSecrets(ctx, gc, ps)
	if err != nil {
		panic(fmt.Sprintf("unable to rotate secrets: %v", err))
	}
}
