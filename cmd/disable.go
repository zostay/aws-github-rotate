package cmd

import (
	"github.com/spf13/cobra"
	"github.com/zostay/aws-github-rotate/pkg/config"
	"github.com/zostay/aws-github-rotate/pkg/disable"
	"github.com/zostay/aws-github-rotate/pkg/plugin"
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
	buildMgr := plugin.NewManager(c.Plugins)
	for _, d := range c.Disablements {
		RunDisablement(buildMgr, &d)
	}
}

func RunDisablement(
	buildMgr *plugin.Manager,
	d *config.Disablement,
) {
	slog := logger.Sugar()

	dc, err := buildMgr.Instance(ctx, d.DisableClient)
	disCli, ok := dc.(disable.Client)
	if !ok {
		slog.Errorw(
			"failed to load disable client",
			"client_name", d.DisableClient,
			"error", err,
		)
		return
	}

	secretSet, err := findSecretSet(d.SecretSet)
	if err != nil {
		slog.Errorw(
			"failed to locate the secret set to work with ",
			"client_name", d.DisableClient,
			"client_desc", disCli.Name(),
			"error", err,
		)
		return
	}

	m := disable.New(
		disCli,
		d.DisableAfter,
		dryRun,
		secretSet.Secrets,
	)

	err = m.DisableSecrets(ctx)
	if err != nil {
		slog.Errorw(
			"failed to complete secret disablement",
			"client_name", d.DisableClient,
			"client_desc", dc.Name(),
			"error", err,
		)
	}
}
