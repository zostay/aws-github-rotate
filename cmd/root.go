package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zostay/aws-github-rotate/pkg/config"
)

var (
	rootCmd *cobra.Command
	c       config.Config
	cfgFile string
	dryRun  bool
	verbose bool
	devMode bool
	logger  *zap.Logger
)

func init() {
	cobra.OnInitialize(initLogger, initConfig)

	rootCmd = &cobra.Command{
		Use:   "aws-github-rotate",
		Short: "tools for managing AWS secrets on github",
	}

	viper.SetDefault("RotateAfter", 168*time.Hour)
	viper.SetDefault("DisableAfter", 48*time.Hour)

	rootCmd.PersistentFlags().BoolVar(
		&devMode, "dev-mode",
		"turns on developer mode logging",
	)
	rootCmd.PersistentFlags().StringVar(
		&cfgFile, "config-file", "",
		"config file (default is /aws-github-rotate.yaml)",
	)
	rootCmd.PersistentFlags().Duration(
		"rotate-after", 168*time.Hour,
		"keys older than rotate-after will be rotated",
	)
	rootCmd.PersistentFlags().Duration(
		"disable-after", 48*time.Hour,
		"keys older than rotate-after + disable-after will be disabled",
	)
	rootCmd.PersistentFlags().BoolVar(
		&dryRun, "dry-run", false,
		"a dry-run describes what would happen without doing it",
	)
	rootCmd.PersistentFlags().BoolVarP(
		&verbose, "verbose", "v", false,
		"more verbose logging",
	)

	viper.BindPFlag(
		"rotateAfter", rootCmd.PersistentFlags().Lookup("rotate-after"),
	)
	viper.BindPFlag(
		"disableAfter", rootCmd.PersistentFlags().Lookup("disable-after"),
	)

	viper.SetDefault("rotateAfter", 168*time.Hour)
	viper.SetDefault("disableAfter", 48*time.Hour)

	initRotateCmd()
	initDisableCmd()
}

func initLogger() {
	var err error
	if devMode {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}

	logger = logger.WithOptions(
		zap.IncreaseLevel(zap.DebugLevel),
	)

	if err != nil {
		panic(fmt.Sprintf("failure to setup logger: %w", err))
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("/")
		viper.SetConfigType("yaml")
		viper.SetConfigName("aws-github-rotate")
	}

	viper.AutomaticEnv()

	slog := logger.Sugar()

	err := viper.ReadInConfig()
	if err != nil {
		slog.Fatalf("unable to read configuration: %v", err)
	}

	err = viper.Unmarshal(&c)
	if err != nil {
		slog.Fatalf("unable to unmarshal configuration: %v", err)
	}

	err = c.Prepare()
	if err != nil {
		slog.Fatalf("unable to finish processing configuration: %v", err)
	}

	// if verbose {
	// 	fmt.Printf("%+v\n", c)
	// }
}

func Execute() {
	err := rootCmd.Execute()
	cobra.CheckErr(err)
}
