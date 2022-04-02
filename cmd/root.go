package cmd

import (
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	rootCmd *cobra.Command

	githubToken string
)

func init() {
	rootCmd = &cobra.Command{
		Use:   "aws-github-rotate",
		Short: "tools for managing AWS secrets on github",
	}

	rootCmd.PersistentFlags().DurationVar(&rotateAge, "rotate-age", 168*time.Hour, "keys older than rotate-age will be rotated")
	rootCmd.PersistentFlags().DurationVar(&activeAge, "active-age", 48*time.Hour, "keys older than rotate-age + active-age will be disabled")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "a dry-run describes what would happen without doing it")

	initRotateCmd()
}

func Execute() {
	githubToken = os.Getenv("GITHUB_TOKEN")

	err := rootCmd.Execute()
	cobra.CheckErr(err)
}
