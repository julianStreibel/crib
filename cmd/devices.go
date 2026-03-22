package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/julianStreibel/crib/internal/config"
	cerrors "github.com/julianStreibel/crib/internal/errors"
	"github.com/julianStreibel/crib/internal/tradfri"
	"github.com/spf13/cobra"
)

var devicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "Control lights, switches, and plugs",
}

var devicesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all devices and their states",
	Run: func(cmd *cobra.Command, args []string) {
		client := mustTradfriClient()
		devices, err := client.GetAllDevices()
		if err != nil {
			exitErr(cerrors.Provider("tradfri", err))
		}

		if len(devices) == 0 {
			fmt.Println("No devices found.")
			return
		}

		for _, d := range devices {
			extra := ""
			if d.Dimmable {
				extra = " [dimmable]"
			}
			fmt.Printf("%-6d %-30s %-8s %-15s %s%s\n",
				d.ID, d.Name, d.TypeString(), d.StateString(), d.Model, extra)
		}
	},
}

var devicesOnCmd = &cobra.Command{
	Use:   "on <name>",
	Short: "Turn on a device",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustTradfriClient()
		dev := mustFindDevice(client, args[0])
		checkReachable(dev)
		if err := client.TurnOn(dev); err != nil {
			exitErr(cerrors.Provider("tradfri", err))
		}
		fmt.Printf("Turned on %s\n", dev.Name)
	},
}

var devicesOffCmd = &cobra.Command{
	Use:   "off <name>",
	Short: "Turn off a device",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustTradfriClient()
		dev := mustFindDevice(client, args[0])
		checkReachable(dev)
		if err := client.TurnOff(dev); err != nil {
			exitErr(cerrors.Provider("tradfri", err))
		}
		fmt.Printf("Turned off %s\n", dev.Name)
	},
}

var devicesToggleCmd = &cobra.Command{
	Use:   "toggle <name>",
	Short: "Toggle a device on/off",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustTradfriClient()
		dev := mustFindDevice(client, args[0])
		checkReachable(dev)
		if err := client.Toggle(dev); err != nil {
			exitErr(cerrors.Provider("tradfri", err))
		}
		fmt.Printf("Toggled %s\n", dev.Name)
	},
}

var devicesDimCmd = &cobra.Command{
	Use:   "dim <name> <0-100>",
	Short: "Set device brightness (0-100)",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustTradfriClient()
		dev := mustFindDevice(client, args[0])
		checkReachable(dev)

		if !dev.Dimmable {
			exitErr(cerrors.InvalidArgWithHint(
				fmt.Sprintf("'%s' is not dimmable", dev.Name),
				"only lights with [dimmable] support brightness control",
			))
		}

		pct, err := strconv.Atoi(args[1])
		if err != nil || pct < 0 || pct > 100 {
			exitErr(cerrors.InvalidArgWithHint(
				fmt.Sprintf("brightness must be 0-100, got '%s'", args[1]),
				"usage: crib devices dim <name> <0-100>",
			))
		}

		if err := client.SetBrightness(dev, pct); err != nil {
			exitErr(cerrors.Provider("tradfri", err))
		}
		fmt.Printf("Set %s brightness to %d%%\n", dev.Name, pct)
	},
}

func mustTradfriClient() *tradfri.Client {
	cfg, err := config.LoadTradfri()
	if err != nil {
		exitErr(cerrors.NotConfigured("TRÅDFRI"))
	}
	return tradfri.NewClient(cfg.TradfriHost, cfg.TradfriIdentity, cfg.TradfriPSK)
}

func mustFindDevice(client *tradfri.Client, query string) *tradfri.Device {
	// Try by ID first
	if id, err := strconv.Atoi(query); err == nil {
		dev, err := client.GetDevice(id)
		if err != nil {
			exitErr(cerrors.Provider("tradfri", err))
		}
		return dev
	}

	// Search by name
	devices, err := client.GetAllDevices()
	if err != nil {
		exitErr(cerrors.Provider("tradfri", err))
	}

	query = strings.ToLower(query)

	// Exact match
	for _, d := range devices {
		if strings.ToLower(d.Name) == query {
			return d
		}
	}
	// Substring match
	for _, d := range devices {
		if strings.Contains(strings.ToLower(d.Name), query) {
			return d
		}
	}

	// Not found — build available list
	available := make([]string, 0, len(devices))
	for _, d := range devices {
		available = append(available, fmt.Sprintf("%s (%s, %s)", d.Name, d.TypeString(), d.StateString()))
	}
	exitErr(cerrors.NotFound("device", query, available))
	return nil
}

func checkReachable(dev *tradfri.Device) {
	if !dev.Reachable {
		exitErr(cerrors.Unreachable(dev.Name))
	}
}

func exitErr(err *cerrors.Error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(err.ExitCode())
}

func init() {
	devicesCmd.AddCommand(devicesListCmd)
	devicesCmd.AddCommand(devicesOnCmd)
	devicesCmd.AddCommand(devicesOffCmd)
	devicesCmd.AddCommand(devicesToggleCmd)
	devicesCmd.AddCommand(devicesDimCmd)
	rootCmd.AddCommand(devicesCmd)
}
