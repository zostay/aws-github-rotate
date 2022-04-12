package cmd

import (
	"context"

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
	ctx := context.Background()
	gc := githubClient(ctx, c.GithubToken)
	svcIam := iamClient(ctx)

	r := rotate.New(
		gc, svcIam,
		c.RotateAfter, c.DisableAfter,
		dryRun, verbose,
		c.ProjectMap,
	)

	slog := logger.Sugar()

	err := r.RefreshGithubState(ctx)
	if err != nil {
		slog.Fatal(err)
	}

	err = r.RotateSecrets(ctx)
	if err != nil {
		slog.Fatal(err)
	}

	if alsoDisable {
		err = r.DisableOldSecrets(ctx)
		if err != nil {
			slog.Fatal(err)
		}
	}
}
