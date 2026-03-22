package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/julianStreibel/crib/internal/config"
	"github.com/julianStreibel/crib/internal/spotify"
	"github.com/spf13/cobra"
)

var spotifyCmd = &cobra.Command{
	Use:   "spotify",
	Short: "Control Spotify playback on any device",
}

var spotifyLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to Spotify (opens browser for authorization)",
	Run: func(cmd *cobra.Command, args []string) {
		clientID, _, err := config.LoadSpotify()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		token, err := spotify.AuthorizePKCE(clientID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		if err := config.SaveSpotifyToken(token.AccessToken, token.RefreshToken, token.ExpiresAt); err != nil {
			fmt.Fprintf(os.Stderr, "error saving token: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Logged in to Spotify successfully.")
	},
}

var spotifyDevicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "List available Spotify Connect devices",
	Run: func(cmd *cobra.Command, args []string) {
		client := mustPlayerClient()
		devices, err := client.GetDevices()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if len(devices) == 0 {
			fmt.Println("No active Spotify devices found.")
			return
		}
		for _, d := range devices {
			active := ""
			if d.IsActive {
				active = " [active]"
			}
			fmt.Printf("%-30s %-12s vol:%-3d %s%s\n", d.Name, d.Type, d.VolumePercent, d.ID, active)
		}
	},
}

var spotifyStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current Spotify playback status",
	Run: func(cmd *cobra.Command, args []string) {
		client := mustPlayerClient()
		state, err := client.GetPlayerState()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if state == nil {
			fmt.Println("Nothing currently playing.")
			return
		}

		status := "paused"
		if state.IsPlaying {
			status = "playing"
		}

		track := ""
		if state.Item != nil {
			artists := make([]string, len(state.Item.Artists))
			for i, a := range state.Item.Artists {
				artists[i] = a.Name
			}
			track = strings.Join(artists, ", ") + " - " + state.Item.Name
		}

		fmt.Printf("Status:  %s\n", status)
		if track != "" {
			fmt.Printf("Track:   %s\n", track)
			if state.Item != nil {
				fmt.Printf("Album:   %s\n", state.Item.Album.Name)
				progressSec := state.ProgressMs / 1000
				durationSec := state.Item.DurationMs / 1000
				fmt.Printf("Time:    %d:%02d / %d:%02d\n",
					progressSec/60, progressSec%60,
					durationSec/60, durationSec%60)
			}
		}
		fmt.Printf("Device:  %s (vol: %d%%)\n", state.Device.Name, state.Device.VolumePercent)
	},
}

var spotifyPlayCmd = &cobra.Command{
	Use:   "play [search query or spotify URI]",
	Short: "Play or resume. Optionally search and play a track/album/playlist",
	Run: func(cmd *cobra.Command, args []string) {
		client := mustPlayerClient()

		if len(args) == 0 {
			if err := client.Play(""); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Resumed playback.")
			return
		}

		query := strings.Join(args, " ")

		// If it's a Spotify URI, play it directly
		if strings.HasPrefix(query, "spotify:") {
			if err := client.PlayURI(query, ""); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Playing %s\n", query)
			return
		}

		// Search and play top result
		searchClient := mustSpotifySearchClient()
		results, err := searchClient.Search(query, "track,playlist,album", 1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error searching: %v\n", err)
			os.Exit(1)
		}

		// Prefer tracks, then playlists, then albums
		if len(results.Tracks) > 0 {
			t := results.Tracks[0]
			if err := client.PlayURI(t.URI, ""); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Playing %s - %s\n", t.Artists, t.Name)
		} else if len(results.Playlists) > 0 {
			p := results.Playlists[0]
			if err := client.PlayURI(p.URI, ""); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Playing playlist: %s\n", p.Name)
		} else if len(results.Albums) > 0 {
			a := results.Albums[0]
			if err := client.PlayURI(a.URI, ""); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Playing album: %s - %s\n", a.Artists, a.Name)
		} else {
			fmt.Fprintf(os.Stderr, "No results found for %q\n", query)
			os.Exit(1)
		}
	},
}

var spotifyPauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pause playback",
	Run: func(cmd *cobra.Command, args []string) {
		client := mustPlayerClient()
		if err := client.Pause(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Paused.")
	},
}

var spotifyNextCmd = &cobra.Command{
	Use:   "next",
	Short: "Skip to next track",
	Run: func(cmd *cobra.Command, args []string) {
		client := mustPlayerClient()
		if err := client.Next(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Skipped to next track.")
	},
}

var spotifyPrevCmd = &cobra.Command{
	Use:   "prev",
	Short: "Go to previous track",
	Run: func(cmd *cobra.Command, args []string) {
		client := mustPlayerClient()
		if err := client.Previous(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Previous track.")
	},
}

var spotifyVolumeCmd = &cobra.Command{
	Use:   "volume <0-100|up|down>",
	Short: "Set Spotify volume",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustPlayerClient()

		var vol int
		switch args[0] {
		case "up":
			state, err := client.GetPlayerState()
			if err != nil || state == nil {
				fmt.Fprintf(os.Stderr, "error: cannot get current volume\n")
				os.Exit(1)
			}
			vol = state.Device.VolumePercent + 5
		case "down":
			state, err := client.GetPlayerState()
			if err != nil || state == nil {
				fmt.Fprintf(os.Stderr, "error: cannot get current volume\n")
				os.Exit(1)
			}
			vol = state.Device.VolumePercent - 5
		default:
			v, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: volume must be 0-100, 'up', or 'down'\n")
				os.Exit(1)
			}
			vol = v
		}

		if vol < 0 {
			vol = 0
		}
		if vol > 100 {
			vol = 100
		}

		if err := client.SetVolume(vol); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Volume set to %d%%\n", vol)
	},
}

var spotifyTransferCmd = &cobra.Command{
	Use:   "transfer <device-name>",
	Short: "Transfer playback to another device",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustPlayerClient()
		query := strings.ToLower(strings.Join(args, " "))

		devices, err := client.GetDevices()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		for _, d := range devices {
			if strings.Contains(strings.ToLower(d.Name), query) {
				if err := client.TransferPlayback(d.ID, true); err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("Transferred playback to %s\n", d.Name)
				return
			}
		}

		fmt.Fprintf(os.Stderr, "No device matching %q found. Available:\n", query)
		for _, d := range devices {
			fmt.Fprintf(os.Stderr, "  %s (%s)\n", d.Name, d.Type)
		}
		os.Exit(1)
	},
}

var spotifyRepeatCmd = &cobra.Command{
	Use:   "repeat <track|playlist|off>",
	Short: "Set repeat mode",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustPlayerClient()
		mode := args[0]
		switch mode {
		case "track", "song":
			mode = "track"
		case "playlist", "album", "context", "all":
			mode = "context"
		case "off", "none":
			mode = "off"
		default:
			fmt.Fprintf(os.Stderr, "error: mode must be track, playlist, or off\n")
			os.Exit(1)
		}
		if err := client.SetRepeat(mode); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Repeat set to %s\n", mode)
	},
}

var spotifyShuffleCmd = &cobra.Command{
	Use:   "shuffle <on|off>",
	Short: "Set shuffle mode",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustPlayerClient()
		on := args[0] == "on" || args[0] == "true"
		if err := client.SetShuffle(on); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if on {
			fmt.Println("Shuffle on.")
		} else {
			fmt.Println("Shuffle off.")
		}
	},
}

var spotifyQueueCmd = &cobra.Command{
	Use:   "queue <search query or spotify URI>",
	Short: "Add a track to the queue",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustPlayerClient()
		query := strings.Join(args, " ")

		var uri string
		if strings.HasPrefix(query, "spotify:") {
			uri = query
		} else {
			searchClient := mustSpotifySearchClient()
			results, err := searchClient.Search(query, "track", 1)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			if len(results.Tracks) == 0 {
				fmt.Fprintf(os.Stderr, "No tracks found for %q\n", query)
				os.Exit(1)
			}
			uri = results.Tracks[0].URI
			fmt.Printf("Adding %s - %s to queue\n", results.Tracks[0].Artists, results.Tracks[0].Name)
		}

		if err := client.AddToQueue(uri); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Added to queue.")
	},
}

var spotifyRadioCmd = &cobra.Command{
	Use:   "radio <search query or spotify URI>",
	Short: "Start a radio based on a song (plays similar tracks)",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustPlayerClient()
		searchClient := mustSpotifySearchClient()
		query := strings.Join(args, " ")

		// Find the seed track
		var trackID string
		if strings.HasPrefix(query, "spotify:track:") {
			trackID = strings.TrimPrefix(query, "spotify:track:")
		} else {
			results, err := searchClient.Search(query, "track", 1)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			if len(results.Tracks) == 0 {
				fmt.Fprintf(os.Stderr, "No tracks found for %q\n", query)
				os.Exit(1)
			}
			trackID = strings.TrimPrefix(results.Tracks[0].URI, "spotify:track:")
			fmt.Printf("Starting radio based on: %s - %s\n", results.Tracks[0].Artists, results.Tracks[0].Name)
		}

		// Get recommendations
		uris, err := client.GetRecommendations(trackID, 30)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error getting recommendations: %v\n", err)
			os.Exit(1)
		}
		if len(uris) == 0 {
			fmt.Fprintf(os.Stderr, "No recommendations found\n")
			os.Exit(1)
		}

		// Prepend the seed track
		uris = append([]string{"spotify:track:" + trackID}, uris...)

		if err := client.PlayURIs(uris, ""); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Playing radio (%d tracks)\n", len(uris))
	},
}

func mustPlayerClient() *spotify.PlayerClient {
	clientID, _, err := config.LoadSpotify()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	accessToken, refreshToken, expiresAt, err := config.LoadSpotifyToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\nRun: crib spotify login\n", err)
		os.Exit(1)
	}

	token := &spotify.TokenData{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}

	return spotify.NewPlayerClient(clientID, token, func(t *spotify.TokenData) {
		_ = config.SaveSpotifyToken(t.AccessToken, t.RefreshToken, t.ExpiresAt)
	})
}

func init() {
	spotifyCmd.AddCommand(spotifyLoginCmd)
	spotifyCmd.AddCommand(spotifyDevicesCmd)
	spotifyCmd.AddCommand(spotifyStatusCmd)
	spotifyCmd.AddCommand(spotifyPlayCmd)
	spotifyCmd.AddCommand(spotifyPauseCmd)
	spotifyCmd.AddCommand(spotifyNextCmd)
	spotifyCmd.AddCommand(spotifyPrevCmd)
	spotifyCmd.AddCommand(spotifyVolumeCmd)
	spotifyCmd.AddCommand(spotifyTransferCmd)
	spotifyCmd.AddCommand(spotifyRepeatCmd)
	spotifyCmd.AddCommand(spotifyShuffleCmd)
	spotifyCmd.AddCommand(spotifyQueueCmd)
	spotifyCmd.AddCommand(spotifyRadioCmd)
	rootCmd.AddCommand(spotifyCmd)
}
