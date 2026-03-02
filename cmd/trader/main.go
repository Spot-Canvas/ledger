package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var version = "dev"

func init() {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}
}

var rootCmd = &cobra.Command{
	Use:           "trader",
	Short:         "Trader CLI",
	Long:          `trader is the command-line interface for the Signal Ngn trader service.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().String("trader-url", "", "Trader service URL (overrides config/env)")
	rootCmd.PersistentFlags().Bool("json", false, "Output as JSON")

	rootCmd.Version = version
}

func initConfig() {
	loadConfig()
	_ = viper.BindPFlag("trader_url", rootCmd.PersistentFlags().Lookup("trader-url"))
}
