package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/julianStreibel/crib/internal/cache"
	"github.com/julianStreibel/crib/internal/config"
	cerrors "github.com/julianStreibel/crib/internal/errors"
	"github.com/julianStreibel/crib/internal/tradfri"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
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
		defer client.Close()
		devices, err := client.GetAllDevices()
		if err != nil {
			exitErr(cerrors.Provider("tradfri", err))
		}

		updateDeviceCache(devices)

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
	Use:   "on <name> [name...]",
	Short: "Turn on one or more devices",
	Args:  cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		runBulkDeviceAction(cmd, args, "on", func(client *tradfri.Client, dev *tradfri.Device) error {
			return client.TurnOn(dev)
		})
	},
}

var devicesOffCmd = &cobra.Command{
	Use:   "off <name> [name...]",
	Short: "Turn off one or more devices",
	Args:  cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		runBulkDeviceAction(cmd, args, "off", func(client *tradfri.Client, dev *tradfri.Device) error {
			return client.TurnOff(dev)
		})
	},
}

var devicesToggleCmd = &cobra.Command{
	Use:   "toggle <name> [name...]",
	Short: "Toggle one or more devices on/off",
	Args:  cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		runBulkDeviceAction(cmd, args, "toggled", func(client *tradfri.Client, dev *tradfri.Device) error {
			return client.Toggle(dev)
		})
	},
}

var devicesDimCmd = &cobra.Command{
	Use:   "dim <name> <0-100>",
	Short: "Set device brightness (0-100)",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustTradfriClient()
		defer client.Close()
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

var colorTempAliases = map[string]int{
	"warm":    2200,
	"neutral": 2700,
	"sunrise": 3000,
	"cool":    4000,
}

var devicesTempCmd = &cobra.Command{
	Use:   "temp <name> <kelvin|warm|neutral|sunrise|cool>",
	Short: "Set device color temperature",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustTradfriClient()
		defer client.Close()
		dev := mustFindDevice(client, args[0])
		checkReachable(dev)

		if !dev.ColorTemp {
			exitErr(cerrors.InvalidArgWithHint(
				fmt.Sprintf("'%s' does not support color temperature", dev.Name),
				"only lights with color temperature support this command",
			))
		}

		kelvin, ok := colorTempAliases[strings.ToLower(args[1])]
		if !ok {
			var err error
			kelvin, err = strconv.Atoi(args[1])
			if err != nil || kelvin < 1000 || kelvin > 10000 {
				exitErr(cerrors.InvalidArgWithHint(
					fmt.Sprintf("invalid temperature '%s'", args[1]),
					"usage: crib devices temp <name> <kelvin|warm|neutral|sunrise|cool>",
				))
			}
		}

		if err := client.SetColorTemp(dev, kelvin); err != nil {
			exitErr(cerrors.Provider("tradfri", err))
		}
		fmt.Printf("Set %s color temperature to %dK\n", dev.Name, kelvin)
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

	// Try cache for fast lookup by name
	if c, err := cache.Load(); err == nil {
		if entry := c.FindDevice(query); entry != nil {
			dev, err := client.GetDevice(entry.ID)
			if err == nil {
				return dev
			}
			// Cache hit but device fetch failed — fall through to full discovery
		}
	}

	// Full discovery (cache miss or stale cache)
	devices, err := client.GetAllDevices()
	if err != nil {
		exitErr(cerrors.Provider("tradfri", err))
	}

	updateDeviceCache(devices)

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

func runBulkDeviceAction(cmd *cobra.Command, args []string, verb string, action func(*tradfri.Client, *tradfri.Device) error) {
	all, _ := cmd.Flags().GetBool("all")
	if !all && len(args) == 0 {
		fmt.Fprintf(os.Stderr, "error: provide device name(s) or use --all\n")
		os.Exit(1)
	}

	client := mustTradfriClient()
	defer client.Close()

	var devices []*tradfri.Device
	if all {
		// Full discovery
		var err error
		devices, err = client.GetAllDevices()
		if err != nil {
			exitErr(cerrors.Provider("tradfri", err))
		}
		updateDeviceCache(devices)
		// Filter to controllable devices (lights and plugs)
		var controllable []*tradfri.Device
		for _, d := range devices {
			if d.Reachable && (d.Type == tradfri.DeviceTypeLight || d.Type == tradfri.DeviceTypePlug) {
				controllable = append(controllable, d)
			}
		}
		devices = controllable
	} else if len(args) == 1 {
		// Single device — use existing mustFindDevice (with cache)
		dev := mustFindDevice(client, args[0])
		checkReachable(dev)
		if err := action(client, dev); err != nil {
			exitErr(cerrors.Provider("tradfri", err))
		}
		fmt.Printf("Turned %s %s\n", verb, dev.Name)
		return
	} else {
		// Multiple named devices — resolve all via cache, then fallback
		for _, name := range args {
			dev := mustFindDevice(client, name)
			checkReachable(dev)
			devices = append(devices, dev)
		}
	}

	if len(devices) == 0 {
		fmt.Println("No controllable devices found.")
		return
	}

	// Fan out operations concurrently
	type result struct {
		name string
		err  error
	}
	results := make([]result, len(devices))
	g := new(errgroup.Group)
	for i, dev := range devices {
		results[i].name = dev.Name
		g.Go(func() error {
			results[i].err = action(client, dev)
			return nil
		})
	}
	_ = g.Wait()

	var names []string
	for _, r := range results {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", r.name, r.err)
		} else {
			names = append(names, r.name)
		}
	}
	if len(names) > 0 {
		fmt.Printf("Turned %s %s\n", verb, strings.Join(names, ", "))
	}
}

func updateDeviceCache(devices []*tradfri.Device) {
	c, _ := cache.Load()
	if c == nil {
		c = &cache.Cache{}
	}
	entries := make([]cache.DeviceEntry, 0, len(devices))
	for _, d := range devices {
		entries = append(entries, cache.DeviceEntry{ID: d.ID, Name: d.Name})
	}
	c.Devices = entries
	_ = cache.Save(c)
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
	devicesOnCmd.Flags().BoolP("all", "a", false, "Apply to all reachable lights and plugs")
	devicesOffCmd.Flags().BoolP("all", "a", false, "Apply to all reachable lights and plugs")
	devicesToggleCmd.Flags().BoolP("all", "a", false, "Apply to all reachable lights and plugs")

	devicesCmd.AddCommand(devicesListCmd)
	devicesCmd.AddCommand(devicesOnCmd)
	devicesCmd.AddCommand(devicesOffCmd)
	devicesCmd.AddCommand(devicesToggleCmd)
	devicesCmd.AddCommand(devicesDimCmd)
	devicesCmd.AddCommand(devicesTempCmd)
	rootCmd.AddCommand(devicesCmd)
}
