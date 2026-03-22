package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/julianStreibel/crib/internal/config"
	"github.com/julianStreibel/crib/internal/sonos"
	"github.com/julianStreibel/crib/internal/spotify"
	"github.com/julianStreibel/crib/internal/tradfri"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
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
	var devicesBuf, speakersBuf, spotifyBuf strings.Builder
	g := new(errgroup.Group)

	// Devices
	g.Go(func() error {
		cfg, err := config.LoadTradfri()
		if err != nil {
			return nil
		}
		client := tradfri.NewClient(cfg.TradfriHost, cfg.TradfriIdentity, cfg.TradfriPSK)
		defer client.Close()
		devices, err := client.GetAllDevices()
		if err != nil {
			fmt.Fprintf(&devicesBuf, "Devices: error (%v)\n\n", err)
			return nil
		}
		if len(devices) == 0 {
			return nil
		}
		fmt.Fprintln(&devicesBuf, "Devices:")
		for _, d := range devices {
			if d.Type == tradfri.DeviceTypeRemote {
				continue
			}
			extra := ""
			if d.Dimmable {
				extra = " [dimmable]"
			}
			fmt.Fprintf(&devicesBuf, "  %-30s %-8s %-15s %s%s\n", d.Name, d.TypeString(), d.StateString(), d.Model, extra)
		}
		fmt.Fprintln(&devicesBuf)
		return nil
	})

	// Speakers
	g.Go(func() error {
		speakers, err := sonos.Discover(1500 * time.Millisecond)
		if err != nil || len(speakers) == 0 {
			return nil
		}

		type result struct {
			speaker *sonos.Speaker
			state   *sonos.PlaybackState
			err     error
		}
		results := make([]result, len(speakers))
		sg := new(errgroup.Group)
		for i, s := range speakers {
			results[i].speaker = s
			sg.Go(func() error {
				state, err := s.GetPlaybackState()
				results[i].state = state
				results[i].err = err
				return nil
			})
		}
		_ = sg.Wait()

		fmt.Fprintln(&speakersBuf, "Speakers:")
		for _, r := range results {
			group := ""
			if !r.speaker.IsCoordinator {
				group = " [grouped]"
			}
			if r.err != nil {
				fmt.Fprintf(&speakersBuf, "  %-20s %-12s %s%s\n", r.speaker.Room, "error", r.speaker.Model, group)
				continue
			}
			track := r.state.TrackString()
			if track != "" {
				track = " | " + track
			}
			fmt.Fprintf(&speakersBuf, "  %-20s %-12s vol:%-3d %s%s%s\n",
				r.speaker.Room, r.state.StateString(), r.state.Volume, r.speaker.Model, group, track)
		}
		fmt.Fprintln(&speakersBuf)
		return nil
	})

	// Spotify
	g.Go(func() error {
		clientID, _, spotErr := config.LoadSpotify()
		if spotErr != nil {
			return nil
		}
		_, refreshToken, expiresAt, tokenErr := config.LoadSpotifyToken()
		if tokenErr != nil {
			return nil
		}
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
		if err != nil || state == nil {
			return nil
		}
		status := "paused"
		if state.IsPlaying {
			status = "playing"
		}
		fmt.Fprintln(&spotifyBuf, "Spotify:")
		fmt.Fprintf(&spotifyBuf, "  %s on %s (vol: %d%%)\n", status, state.Device.Name, state.Device.VolumePercent)
		if state.Item != nil {
			artists := ""
			for i, a := range state.Item.Artists {
				if i > 0 {
					artists += ", "
				}
				artists += a.Name
			}
			fmt.Fprintf(&spotifyBuf, "  %s - %s\n", artists, state.Item.Name)
		}
		return nil
	})

	_ = g.Wait()

	// Print in consistent order
	fmt.Print(devicesBuf.String())
	fmt.Print(speakersBuf.String())
	fmt.Print(spotifyBuf.String())
}
