package cmd

import (
	"fmt"
	"os"

	"github.com/julianStreibel/crib/internal/config"
	"github.com/julianStreibel/crib/internal/tradfri"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage crib configuration",
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		vals := config.Show()
		fmt.Printf("TRÅDFRI Host:        %s\n", vals["tradfri_host"])
		fmt.Printf("TRÅDFRI Identity:    %s\n", vals["tradfri_identity"])
		fmt.Printf("TRÅDFRI PSK:         %s\n", vals["tradfri_psk"])
		fmt.Printf("Spotify Client ID:   %s\n", vals["spotify_client_id"])
		fmt.Printf("Spotify Secret:      %s\n", vals["spotify_client_secret"])
		fmt.Printf("Config file:         %s\n", vals["config_file"])
	},
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test the connection to the TRÅDFRI gateway",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadTradfri()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		client := tradfri.NewClient(cfg.TradfriHost, cfg.TradfriIdentity, cfg.TradfriPSK)
		if err := client.CheckConnection(); err != nil {
			fmt.Fprintf(os.Stderr, "Connection failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Successfully connected to TRÅDFRI gateway.")
	},
}

var pairCmd = &cobra.Command{
	Use:   "pair <gateway-host> <security-code>",
	Short: "Pair with a TRÅDFRI gateway using the security code from the device",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		host := args[0]
		securityCode := args[1]

		hostname, _ := os.Hostname()
		clientName := "crib"
		if hostname != "" {
			clientName = "crib-" + hostname
		}

		fmt.Printf("Registering with gateway at %s...\n", host)
		identity, psk, err := tradfri.Register(host, securityCode, clientName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		if err := config.SetMultiple(map[string]string{
			"tradfri_host":     host,
			"tradfri_identity": identity,
			"tradfri_psk":      psk,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "error saving config: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Paired successfully. Credentials saved.")
	},
}

func init() {
	configCmd.AddCommand(showCmd)
	configCmd.AddCommand(testCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(pairCmd)
}
