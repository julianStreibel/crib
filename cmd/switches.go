package cmd

import (
	"fmt"
	"os"

	"github.com/julianStreibel/crib/internal/tradfri"
	"github.com/spf13/cobra"
)

var switchesCmd = &cobra.Command{
	Use:   "switches",
	Short: "Control switches and plugs",
}

var switchesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all switches/plugs and their states",
	Run: func(cmd *cobra.Command, args []string) {
		client := mustClient()
		devices, err := client.GetAllDevices()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		found := false
		for _, d := range devices {
			if d.Type != tradfri.DeviceTypePlug {
				continue
			}
			found = true
			fmt.Printf("%-6d %-30s %-15s %s\n", d.ID, d.Name, d.StateString(), d.Model)
		}
		if !found {
			fmt.Println("No switches/plugs found.")
		}
	},
}

var switchesOnCmd = &cobra.Command{
	Use:   "on <device-id>",
	Short: "Turn on a switch/plug",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustClient()
		dev := mustDevice(client, args[0], tradfri.DeviceTypePlug)
		if err := client.TurnOn(dev); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Turned on %s\n", dev.Name)
	},
}

var switchesOffCmd = &cobra.Command{
	Use:   "off <device-id>",
	Short: "Turn off a switch/plug",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustClient()
		dev := mustDevice(client, args[0], tradfri.DeviceTypePlug)
		if err := client.TurnOff(dev); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Turned off %s\n", dev.Name)
	},
}

var switchesToggleCmd = &cobra.Command{
	Use:   "toggle <device-id>",
	Short: "Toggle a switch/plug on/off",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustClient()
		dev := mustDevice(client, args[0], tradfri.DeviceTypePlug)
		if err := client.Toggle(dev); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Toggled %s\n", dev.Name)
	},
}

func init() {
	switchesCmd.AddCommand(switchesListCmd)
	switchesCmd.AddCommand(switchesOnCmd)
	switchesCmd.AddCommand(switchesOffCmd)
	switchesCmd.AddCommand(switchesToggleCmd)
	rootCmd.AddCommand(switchesCmd)
}
