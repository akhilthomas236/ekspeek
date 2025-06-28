package cmd

import "github.com/spf13/cobra"

// Variables used across commands
var (
	profile     string
	region      string
	debug       bool
	clusterName string
)

// AddGlobalFlags adds global flags to the root command
func AddGlobalFlags(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "AWS profile to use")
	rootCmd.PersistentFlags().StringVar(&region, "region", "us-west-2", "AWS region to use")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
}
