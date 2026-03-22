package cmd

import (
	"fmt"
	"time"

	"github.com/julianStreibel/crib/internal/config"
	"github.com/julianStreibel/crib/internal/sonos"
	"github.com/julianStreibel/crib/internal/spotify"
	"github.com/julianStreibel/crib/internal/tradfri"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of all devices, speakers, and music",
	Run:   runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) {
	// Devices
	cfg, err := config.LoadTradfri()
	if err == nil {
		client := tradfri.NewClient(cfg.TradfriHost, cfg.TradfriIdentity, cfg.TradfriPSK)
		devices, err := client.GetAllDevices()
		if err == nil && len(devices) > 0 {
			fmt.Println("Devices:")
			for _, d := range devices {
				if d.Type == tradfri.DeviceTypeRemote {
					continue
				}
				extra := ""
				if d.Dimmable {
					extra = " [dimmable]"
				}
				fmt.Printf("  %-30s %-8s %-15s %s%s\n", d.Name, d.TypeString(), d.StateString(), d.Model, extra)
			}
			fmt.Println()
		} else if err != nil {
			fmt.Printf("Devices: error (%v)\n\n", err)
		}
	}

	// Speakers
	speakers, err := sonos.Discover(3 * time.Second)
	if err == nil && len(speakers) > 0 {
		fmt.Println("Speakers:")
		for _, s := range speakers {
			group := ""
			if !s.IsCoordinator {
				group = " [grouped]"
			}
			state, err := s.GetPlaybackState()
			if err != nil {
				fmt.Printf("  %-20s %-12s %s%s\n", s.Room, "error", s.Model, group)
				continue
			}
			track := state.TrackString()
			if track != "" {
				track = " | " + track
			}
			fmt.Printf("  %-20s %-12s vol:%-3d %s%s%s\n",
				s.Room, state.StateString(), state.Volume, s.Model, group, track)
		}
		fmt.Println()
	}

	// Spotify
	clientID, _, spotErr := config.LoadSpotify()
	if spotErr == nil {
		_, refreshToken, expiresAt, tokenErr := config.LoadSpotifyToken()
		if tokenErr == nil {
			token := &spotify.TokenData{
				AccessToken:  "",
				RefreshToken: refreshToken,
				ExpiresAt:    expiresAt,
			}
			// Force refresh to get a valid token
			token.ExpiresAt = 0
			player := spotify.NewPlayerClient(clientID, token, func(t *spotify.TokenData) {
				_ = config.SaveSpotifyToken(t.AccessToken, t.RefreshToken, t.ExpiresAt)
			})
			state, err := player.GetPlayerState()
			if err == nil && state != nil {
				status := "paused"
				if state.IsPlaying {
					status = "playing"
				}
				fmt.Println("Spotify:")
				fmt.Printf("  %s on %s (vol: %d%%)\n", status, state.Device.Name, state.Device.VolumePercent)
				if state.Item != nil {
					artists := ""
					for i, a := range state.Item.Artists {
						if i > 0 {
							artists += ", "
						}
						artists += a.Name
					}
					fmt.Printf("  %s - %s\n", artists, state.Item.Name)
				}
			}
		}
	}
}
