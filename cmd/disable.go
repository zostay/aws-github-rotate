package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/zostay/aws-github-rotate/pkg/rotate"
)

var (
	disableCmd *cobra.Command
)

func initDisableCmd() {
	disableCmd = &cobra.Command{
		Use:   "disable",
		Short: "disable previous AWS key/secrets following rotation",
		Run:   RunDisable,
	}

	rootCmd.AddCommand(disableCmd)
}

func RunDisable(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	gc := githubClient(ctx, c.GithubToken)
	svcIam := iamClient(ctx)

	r := rotate.New(
		gc, svcIam,
		c.RotateAfter, c.DisableAfter,
		dryRun, verbose,
		c.ProjectMap,
	)

	err := r.RefreshGithubState(ctx)
	if err != nil {
		fatalf("%v", err)
	}

	err = r.DisableOldSecrets(ctx)
	if err != nil {
		fatalf("%v", err)
	}
}
