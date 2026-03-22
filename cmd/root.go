package cmd

import (
	"fmt"
	"os"

	"github.com/julianStreibel/crib/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "crib",
	Short: "Smart home CLI for humans and LLM agents",
	Long:  "crib is a command-line tool to control smart home devices. Built for both human use and LLM agent automation. Supports IKEA TRÅDFRI lights/switches, Sonos speakers, and Spotify.",
}

func Execute() {
	config.Init()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
