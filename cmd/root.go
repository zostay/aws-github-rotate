package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zostay/aws-github-rotate/pkg/config"
)

const (
	DefaultAccessKey = "AWS_ACCESS_KEY_ID"
	DefaultSecretKey = "AWS_SECRET_ACCESS_KEY"
)

var (
	rootCmd *cobra.Command
	c       config.Config
	cfgFile string
	dryRun  bool
	verbose bool
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd = &cobra.Command{
		Use:   "aws-github-rotate",
		Short: "tools for managing AWS secrets on github",
	}

	viper.SetDefault("RotateAfter", 168*time.Hour)
	viper.SetDefault("DisableAfter", 48*time.Hour)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config-file", "", "config file (default is /aws-github-rotate.yaml)")
	rootCmd.PersistentFlags().Duration("rotate-after", 168*time.Hour, "keys older than rotate-after will be rotated")
	rootCmd.PersistentFlags().Duration("disable-after", 48*time.Hour, "keys older than rotate-after + disable-after will be disabled")
	rootCmd.PersistentFlags().String("access-key", DefaultAccessKey, "set the default key to use to store the access key in github")
	rootCmd.PersistentFlags().String("secret-key", DefaultSecretKey, "set the default key to use to store the secret key in github")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "a dry-run describes what would happen without doing it")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "more verbose logging")

	viper.BindPFlag("rotateAfter", rootCmd.PersistentFlags().Lookup("rotate-after"))
	viper.BindPFlag("disableAfter", rootCmd.PersistentFlags().Lookup("disable-after"))
	viper.BindPFlag("defaultAccessKey", rootCmd.PersistentFlags().Lookup("access-key"))
	viper.BindPFlag("defaultSecretKey", rootCmd.PersistentFlags().Lookup("secret-key"))

	viper.SetDefault("rotateAfter", 168*time.Hour)
	viper.SetDefault("disableAfter", 48*time.Hour)
	viper.SetDefault("defaultAccessKey", DefaultAccessKey)
	viper.SetDefault("defaultSecretKey", DefaultSecretKey)

	initRotateCmd()
	initDisableCmd()
}

func fatalf(f string, args ...any) {
	fmt.Fprintf(os.Stderr, f, args...)
	os.Exit(1)
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

	err := viper.ReadInConfig()
	if err != nil {
		fatalf("unable to read configuration: %v", err)
	}

	err = viper.Unmarshal(&c)
	if err != nil {
		fatalf("unable to unmarshal configuration: %v", err)
	}

	// if verbose {
	// 	fmt.Printf("%+v\n", c)
	// }
}

func Execute() {
	err := rootCmd.Execute()
	cobra.CheckErr(err)
}
