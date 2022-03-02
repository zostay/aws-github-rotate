package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/google/go-github/v42/github"
	"golang.org/x/oauth2"
)

const (
	githubAccessToken  = "***REMOVED***"
	githubAccessKeyKey = "AWS_ACCESS_KEY_ID"
	githubSecretKeyKey = "AWS_SECRET_ACCESS_KEY"
)

var (
	maxAge       = 336 * time.Hours
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
			var updated time.Time
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
					recordEarliesstUpdateDate(secret.UpdatedAt.Time)
				}
				if ak && sk {
					break
				}
			}

			if ak && sk {
				if user, ok := iamUsers[strings.Join([]string{owner, name}, "/")]; ok {
					p := Project{
						Owner: owner,
						Name:  name,

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

func getAccessKeys(ctx context.Context, user string) (*iam.AccessKeyMetadata, *iam.AccessKeyMetadata, error) {
	ak, err := c.svc.LisAccessKeys(&iam.ListAccessKeysInput{
		UserName: aws.String(user),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("c.svc.ListAccessKeys(%q): %w", user, err)
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

func rotateSecret(ctx context.Context, p Project) error {
	fmt.Printf("Rotating IAM account for %s/%s\n", p.Owner, p.Name)

	oldKey, newKey, err := getAccessKeys(ctx, p.User)
	if err != nil {
		return fmt.Errorf("getAccessKeys(%q): %w", p.User, err)
	}

	return nil
}

func rotateOldSecrets(ctx context.Context, ps []Project) error {
	session := session.Must(session.NewSession())
	svcIam := iam.New(session)

	for _, p := range ps {
		if time.Since(p.SecretUpdatedAt) > maxAge {
			// secret is too old: rotate
			err := rotateSecret(ctx, p)
			if err != nil {
				return fmt.Errorf("failed to rotate old secret: %w", err)
			}
		} else {
			// the github secret is too old to be the current secret: rotate
			keyAge, err := getOlderApiKeyAge(ctx, p.User)
			if err != nil {
				return fmt.Errorf("failed to retrieve key age: %w", err)
			}

			if p.SecretUpdatedAt.Before(keyAge) {
				err := rotateSecret(ctx, p)
				if err != nil {
					return fmt.Errorf("failed to rotate expired secret: %w", err)
				}
			}
		}
	}

	return nil
}

func main() {
	ctx := context.Background()
	gc, err := githubClient(ctx, githubAccessToken)
	if err != nil {
		panic(fmt.Sprintf("unable to authorize with github: %v", err))
	}

	ps, err := listReposWithSecrets(ctx, gc)
	if err != nil {
		panic(fmt.Sprintf("unable list repositories with secrets: %v", err))
	}

	err = rotateOldSecrets(ctx, ps)
	if err != nil {
		panic(fmt.Sprintf("unable to rotate secrets: %v", err))
	}
}
