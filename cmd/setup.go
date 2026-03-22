package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/julianStreibel/crib/internal/config"
	"github.com/julianStreibel/crib/internal/discovery"
	"github.com/julianStreibel/crib/internal/spotify"
	"github.com/julianStreibel/crib/internal/tradfri"
	"github.com/spf13/cobra"
)

const maxRetries = 3

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard for connecting to smart home devices",
	Run:   runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("crib setup")
	fmt.Println("================")
	fmt.Println("Each step is optional — press Enter to skip.\n")

	setupTradfri(reader)
	fmt.Println()
	setupSpotify(reader)
	fmt.Println()
	setupSpotifyLogin(reader)

	fmt.Println()
	fmt.Println("Setup complete!")
}

func setupTradfri(reader *bufio.Reader) {
	fmt.Println("[TRÅDFRI Gateway] (optional)")
	fmt.Println()

	// Check if already configured
	cfg, err := config.LoadTradfri()
	if err == nil {
		client := tradfri.NewClient(cfg.TradfriHost, cfg.TradfriIdentity, cfg.TradfriPSK)
		if client.CheckConnection() == nil {
			fmt.Printf("Already paired with gateway at %s (working)\n", cfg.TradfriHost)
			if !confirm(reader, "Reconfigure?") {
				return
			}
		}
	}

	if !confirm(reader, "Set up IKEA TRÅDFRI gateway?") {
		fmt.Println("Skipped.")
		return
	}

	// Auto-discover
	fmt.Print("Scanning for TRÅDFRI gateways...")
	gateways, _ := discovery.FindTradfriGateways(3 * time.Second)
	fmt.Println()

	var host string
	if len(gateways) == 1 {
		host = gateways[0].Host
		fmt.Printf("Found gateway at %s\n", host)
		if !confirm(reader, "Use this gateway?") {
			host = prompt(reader, "Enter gateway IP address")
		}
	} else if len(gateways) > 1 {
		fmt.Println("Found multiple gateways:")
		for i, gw := range gateways {
			fmt.Printf("  %d) %s (%s)\n", i+1, gw.Host, gw.Name)
		}
		choice := prompt(reader, "Enter number or IP address")
		idx := 0
		if _, err := fmt.Sscanf(choice, "%d", &idx); err == nil && idx >= 1 && idx <= len(gateways) {
			host = gateways[idx-1].Host
		} else {
			host = choice
		}
	} else {
		fmt.Println("No gateways found automatically.")
		host = prompt(reader, "Enter gateway IP address (or Enter to skip)")
	}

	if host == "" {
		fmt.Println("Skipped.")
		return
	}

	// Retry loop for pairing
	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Println()
		fmt.Println("Enter the security code from the bottom of your TRÅDFRI gateway.")
		fmt.Println("It's a 16-character code next to the QR code.")
		securityCode := prompt(reader, "Security code (or Enter to skip)")

		if securityCode == "" {
			fmt.Println("Skipped.")
			return
		}

		// Use a unique identity per machine to avoid conflicts
		hostname, _ := os.Hostname()
		clientName := "crib"
		if hostname != "" {
			clientName = "crib-" + hostname
		}

		fmt.Printf("Pairing with gateway at %s...\n", host)
		identity, psk, err := tradfri.Register(host, securityCode, clientName)
		if err != nil {
			fmt.Printf("Pairing failed: %v\n", err)
			if attempt < maxRetries {
				fmt.Printf("Retrying (%d/%d)...\n", attempt, maxRetries)
				continue
			}
			fmt.Println("All attempts failed. You can try again later with: crib pair <host> <code>")
			return
		}

		if err := config.SetMultiple(map[string]string{
			"tradfri_host":     host,
			"tradfri_identity": identity,
			"tradfri_psk":      psk,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			return
		}

		// Verify
		client := tradfri.NewClient(host, identity, psk)
		devices, err := client.GetAllDevices()
		if err != nil {
			fmt.Printf("Paired but could not list devices: %v\n", err)
			return
		}

		lights := 0
		plugs := 0
		for _, d := range devices {
			switch d.Type {
			case tradfri.DeviceTypeLight:
				lights++
			case tradfri.DeviceTypePlug:
				plugs++
			}
		}

		fmt.Printf("Paired successfully! Found %d lights and %d plugs.\n", lights, plugs)
		return
	}
}

func setupSpotify(reader *bufio.Reader) {
	fmt.Println("[Spotify] (optional)")
	fmt.Println()

	// Check if already configured
	clientID, _, err := config.LoadSpotify()
	if err == nil && clientID != "" {
		fmt.Printf("Spotify already configured (client ID: %s...)\n", clientID[:8])
		if !confirm(reader, "Reconfigure?") {
			return
		}
	}

	fmt.Println("To search and play music, you need Spotify API credentials.")
	fmt.Println("Create an app at https://developer.spotify.com/dashboard")
	fmt.Println("Add http://127.0.0.1:8089/callback as a redirect URI.")
	fmt.Println()

	id := prompt(reader, "Spotify Client ID (or Enter to skip)")
	if id == "" {
		fmt.Println("Skipped.")
		return
	}

	secret := prompt(reader, "Spotify Client Secret")
	if secret == "" {
		fmt.Println("Skipped.")
		return
	}

	if err := config.SetMultiple(map[string]string{
		"spotify_client_id":     id,
		"spotify_client_secret": secret,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		return
	}

	fmt.Println("Spotify credentials saved.")
}

func setupSpotifyLogin(reader *bufio.Reader) {
	fmt.Println("[Spotify Login] (optional)")
	fmt.Println()

	// Check if already logged in
	_, refreshToken, _, err := config.LoadSpotifyToken()
	if err == nil && refreshToken != "" {
		fmt.Println("Already logged in to Spotify.")
		if !confirm(reader, "Re-authenticate?") {
			return
		}
	}

	// Check if client ID is configured
	clientID, _, err := config.LoadSpotify()
	if err != nil {
		fmt.Println("Skipping — Spotify API credentials not configured.")
		return
	}

	if !confirm(reader, "Log in to Spotify for playback control? (opens browser)") {
		fmt.Println("Skipped. You can log in later with: crib spotify login")
		return
	}

	token, err := spotify.AuthorizePKCE(clientID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Login failed: %v\n", err)
		fmt.Println("You can try again later with: crib spotify login")
		return
	}

	if err := config.SaveSpotifyToken(token.AccessToken, token.RefreshToken, token.ExpiresAt); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving token: %v\n", err)
		return
	}

	fmt.Println("Logged in to Spotify successfully.")
}

func prompt(reader *bufio.Reader, label string) string {
	fmt.Printf("%s: ", label)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func confirm(reader *bufio.Reader, question string) bool {
	fmt.Printf("%s [Y/n] ", question)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "" || input == "y" || input == "yes"
}
