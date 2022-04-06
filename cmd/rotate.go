package cmd

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/google/go-github/v42/github"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/zostay/aws-github-rotate/pkg/rotate"
)

var rotateCmd *cobra.Command

func initRotateCmd() {
	rotateCmd = &cobra.Command{
		Use:   "rotate",
		Short: "rotate an AWS password and update a github aciton secret",
		Run:   RunRotation,
	}

	rootCmd.AddCommand(rotateCmd)
}

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

func RunRotation(cmd *cobra.Command, args []string) {
	c.GithubToken = os.Getenv("GITHUB_TOKEN")

	ctx := context.Background()
	gc := githubClient(ctx, c.GithubToken)

	session := session.Must(session.NewSession())
	svcIam := iam.New(session)

	r := rotate.New(gc, svcIam, c.RotateAfter, c.DisableAfter, dryRun, c.ProjectMap)

	err := r.RotateSecrets(ctx)
	if err != nil {
		fatalf("%v", err)
	}

	err = r.DisableOldSecrets(ctx)
	if err != nil {
		fatalf("%v", err)
	}
}
