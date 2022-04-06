package cmd

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/google/go-github/v42/github"
	"golang.org/x/oauth2"
)

// githubClient connects to the github API client and returns it or returns an
// error.
func githubClient(ctx context.Context, gat string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: gat},
	)
	oc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(oc)
	return client
}

// iamClient returns a service object for AWS IAM.
func iamClient(ctx context.Context) *iam.IAM {
	session := session.Must(session.NewSession())
	svcIam := iam.New(session)
	return svcIam
}
