package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/google/go-github/github"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/zostay/internal/rotate"
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
func githubClient(ctx context.Context, gat string) (*github.Client, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubAccessToken},
	)
	oc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(oc)
	return client, nil
}

func RunRotation(cmd *cobra.Command, args []string) {
	githubToken = os.Getenv("GITHUB_TOKEN")

	ctx := context.Background()
	gc := githubClient(ctx, githubToken)

	session := session.Must(session.NewSession())
	svcIam := iam.New(session)

	r := rotate.New(gc, svcIam, c.RotateAfter, dryRun, c.ProjectMap())

	r.RotateSecrets(ctx)
	r.DisableSecrets(ctx)

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
