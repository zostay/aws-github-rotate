package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/zostay/aws-github-rotate/pkg/rotate"
)

var (
	rotateCmd   *cobra.Command
	alsoDisable bool
)

func initRotateCmd() {
	rotateCmd = &cobra.Command{
		Use:   "rotate",
		Short: "rotate an AWS IAM key/secret pair and update a github action secret",
		Run:   RunRotation,
	}

	rotateCmd.Flags().BoolVar(&alsoDisable, "also-disable", false, "after rotating keys, check for any old access keys that should be disabled")

	rootCmd.AddCommand(rotateCmd)
}

func RunRotation(cmd *cobra.Command, args []string) {
	c.GithubToken = os.Getenv("GITHUB_TOKEN")

	ctx := context.Background()
	gc := githubClient(ctx, c.GithubToken)
	svcIam := iamClient(ctx)

	r := rotate.New(gc, svcIam, c.RotateAfter, c.DisableAfter, dryRun, c.ProjectMap)

	err := r.RotateSecrets(ctx)
	if err != nil {
		fatalf("%v", err)
	}

	if alsoDisable {
		err = r.DisableOldSecrets(ctx)
		if err != nil {
			fatalf("%v", err)
		}
	}
}
