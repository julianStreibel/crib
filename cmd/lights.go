package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/julianStreibel/crib/internal/config"
	"github.com/julianStreibel/crib/internal/tradfri"
	"github.com/spf13/cobra"
)

var lightsCmd = &cobra.Command{
	Use:   "lights",
	Short: "Control lights",
}

var lightsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all lights and their states",
	Run: func(cmd *cobra.Command, args []string) {
		client := mustClient()
		devices, err := client.GetAllDevices()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		found := false
		for _, d := range devices {
			if d.Type != tradfri.DeviceTypeLight {
				continue
			}
			found = true
			dimmable := ""
			if d.Dimmable {
				dimmable = " [dimmable]"
			}
			fmt.Printf("%-6d %-30s %-15s %s%s\n", d.ID, d.Name, d.StateString(), d.Model, dimmable)
		}
		if !found {
			fmt.Println("No lights found.")
		}
	},
}

var lightsOnCmd = &cobra.Command{
	Use:   "on <device-id>",
	Short: "Turn on a light",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustClient()
		dev := mustDevice(client, args[0], tradfri.DeviceTypeLight)
		if err := client.TurnOn(dev); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Turned on %s\n", dev.Name)
	},
}

var lightsOffCmd = &cobra.Command{
	Use:   "off <device-id>",
	Short: "Turn off a light",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustClient()
		dev := mustDevice(client, args[0], tradfri.DeviceTypeLight)
		if err := client.TurnOff(dev); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Turned off %s\n", dev.Name)
	},
}

var lightsToggleCmd = &cobra.Command{
	Use:   "toggle <device-id>",
	Short: "Toggle a light on/off",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustClient()
		dev := mustDevice(client, args[0], tradfri.DeviceTypeLight)
		if err := client.Toggle(dev); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Toggled %s\n", dev.Name)
	},
}

var lightsDimCmd = &cobra.Command{
	Use:   "dim <device-id> <percent>",
	Short: "Set light brightness (0-100)",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustClient()
		dev := mustDevice(client, args[0], tradfri.DeviceTypeLight)

		pct, err := strconv.Atoi(args[1])
		if err != nil || pct < 0 || pct > 100 {
			fmt.Fprintf(os.Stderr, "error: brightness must be a number between 0 and 100\n")
			os.Exit(1)
		}

		if err := client.SetBrightness(dev, pct); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Set %s brightness to %d%%\n", dev.Name, pct)
	},
}

func mustClient() *tradfri.Client {
	cfg, err := config.LoadTradfri()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return tradfri.NewClient(cfg.TradfriHost, cfg.TradfriIdentity, cfg.TradfriPSK)
}

func mustDevice(client *tradfri.Client, idStr string, expectedType tradfri.DeviceType) *tradfri.Device {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		// Try finding by name
		devices, err := client.GetAllDevices()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		for _, d := range devices {
			if d.Type == expectedType && matchesName(d.Name, idStr) {
				return d
			}
		}
		fmt.Fprintf(os.Stderr, "error: device %q not found\n", idStr)
		os.Exit(1)
	}

	dev, err := client.GetDevice(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return dev
}

// matchesName does a case-insensitive substring match.
func matchesName(name, query string) bool {
	name = strings.ToLower(name)
	query = strings.ToLower(query)
	return strings.Contains(name, query)
}

func init() {
	lightsCmd.AddCommand(lightsListCmd)
	lightsCmd.AddCommand(lightsOnCmd)
	lightsCmd.AddCommand(lightsOffCmd)
	lightsCmd.AddCommand(lightsToggleCmd)
	lightsCmd.AddCommand(lightsDimCmd)
	rootCmd.AddCommand(lightsCmd)
}
