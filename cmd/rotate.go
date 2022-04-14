package cmd

import (
	"github.com/spf13/cobra"

	"github.com/zostay/aws-github-rotate/pkg/config"
	"github.com/zostay/aws-github-rotate/pkg/plugin"
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
	buildMgr := plugin.NewManager(c.Clients)
	for _, r := range c.Rotations {
		RunRotations(buildMgr, &r)
	}
}

func RunRotations(
	buildMgr *plugin.Manager,
	r *config.Rotation,
) {
	slog := logger.Sugar()

	rc, err := buildMgr.Build(ctx, c.Client)
	if rotCli, ok := rc.(rotate.Client); !ok {
		slog.Errorw(
			"failed to load rotation client",
			"client_name", c.Client.Name,
			"error", err,
		)
		return
	}

	secretSet, err := findSecretSet(d.SecretSet)
	if err != nil {
		slog.Errorw(
			"failed to locate the secret set to work with ",
			"client_name", c.Client.Name,
			"client_desc", dc.Name(),
			"error", err,
		)
		return
	}

	m := rotate.New(
		rc,
		c.rotateAfter,
		dryRun,
		buildMgr,
		secretSet.Secrets,
	)

	err := m.RotateSecrets(ctx)
	if err != nil {
		slog.Errorw(
			"failed to complete secret rotation",
			"client_name", c.Client.Name,
			"client_desc", dc.Name(),
			"error", err,
		)
	}
}
