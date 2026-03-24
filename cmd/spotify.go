package cmd

import (
	"fmt"
	"strings"

	"github.com/julianStreibel/crib/internal/config"
	cerrors "github.com/julianStreibel/crib/internal/errors"
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
			exitErr(cerrors.NotConfigured("Spotify"))
		}

		token, err := spotify.AuthorizePKCE(clientID)
		if err != nil {
			exitErr(cerrors.Provider("spotify", err))
		}

		if err := config.SaveSpotifyToken(token.AccessToken, token.RefreshToken, token.ExpiresAt); err != nil {
			exitErr(cerrors.Provider("config", err))
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
			exitErr(cerrors.Provider("spotify", err))
		}
		if len(devices) == 0 {
			fmt.Println("No active Spotify devices found.")
			fmt.Println("Hint: open Spotify on a device first.")
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
			exitErr(cerrors.Provider("spotify", err))
		}
		if state == nil {
			exitErr(cerrors.NoSession("Spotify"))
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
				exitSpotifyErr(err)
			}
			fmt.Println("Resumed playback.")
			return
		}

		query := strings.Join(args, " ")

		if strings.HasPrefix(query, "spotify:") {
			if err := client.PlayURI(query, ""); err != nil {
				exitSpotifyErr(err)
			}
			fmt.Printf("Playing %s\n", query)
			return
		}

		searchClient := mustSpotifySearchClient()
		results, err := searchClient.Search(query, "track,playlist,album", 1)
		if err != nil {
			exitErr(cerrors.Provider("spotify", err))
		}

		if len(results.Tracks) > 0 {
			t := results.Tracks[0]
			if err := client.PlayURI(t.URI, ""); err != nil {
				exitSpotifyErr(err)
			}
			fmt.Printf("Playing %s - %s\n", t.Artists, t.Name)
		} else if len(results.Playlists) > 0 {
			p := results.Playlists[0]
			if err := client.PlayURI(p.URI, ""); err != nil {
				exitSpotifyErr(err)
			}
			fmt.Printf("Playing playlist: %s\n", p.Name)
		} else if len(results.Albums) > 0 {
			a := results.Albums[0]
			if err := client.PlayURI(a.URI, ""); err != nil {
				exitSpotifyErr(err)
			}
			fmt.Printf("Playing album: %s - %s\n", a.Artists, a.Name)
		} else {
			exitErr(cerrors.NotFound("track", query, nil))
		}
	},
}

var spotifyPauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pause playback",
	Run: func(cmd *cobra.Command, args []string) {
		client := mustPlayerClient()
		if err := client.Pause(); err != nil {
			exitSpotifyErr(err)
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
			exitSpotifyErr(err)
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
			exitSpotifyErr(err)
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

		vol := parseVolume(args[0], func() int {
			state, err := client.GetPlayerState()
			if err != nil || state == nil {
				exitErr(cerrors.NoSession("Spotify"))
			}
			return state.Device.VolumePercent
		})

		if vol < 0 {
			vol = 0
		}
		if vol > 100 {
			vol = 100
		}

		if err := client.SetVolume(vol); err != nil {
			exitSpotifyErr(err)
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
			exitErr(cerrors.Provider("spotify", err))
		}

		for _, d := range devices {
			if strings.Contains(strings.ToLower(d.Name), query) {
				if err := client.TransferPlayback(d.ID, true); err != nil {
					exitSpotifyErr(err)
				}
				fmt.Printf("Transferred playback to %s\n", d.Name)
				return
			}
		}

		available := make([]string, len(devices))
		for i, d := range devices {
			available[i] = fmt.Sprintf("%s (%s)", d.Name, d.Type)
		}
		exitErr(cerrors.NotFound("device", query, available))
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
			exitErr(cerrors.InvalidArgWithHint(
				fmt.Sprintf("unknown repeat mode '%s'", mode),
				"usage: crib spotify repeat <track|playlist|off>",
			))
		}
		if err := client.SetRepeat(mode); err != nil {
			exitSpotifyErr(err)
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
			exitSpotifyErr(err)
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
				exitErr(cerrors.Provider("spotify", err))
			}
			if len(results.Tracks) == 0 {
				exitErr(cerrors.NotFound("track", query, nil))
			}
			uri = results.Tracks[0].URI
			fmt.Printf("Adding %s - %s to queue\n", results.Tracks[0].Artists, results.Tracks[0].Name)
		}

		if err := client.AddToQueue(uri); err != nil {
			exitSpotifyErr(err)
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

		var trackID string
		if strings.HasPrefix(query, "spotify:track:") {
			trackID = strings.TrimPrefix(query, "spotify:track:")
		} else {
			results, err := searchClient.Search(query, "track", 1)
			if err != nil {
				exitErr(cerrors.Provider("spotify", err))
			}
			if len(results.Tracks) == 0 {
				exitErr(cerrors.NotFound("track", query, nil))
			}
			trackID = strings.TrimPrefix(results.Tracks[0].URI, "spotify:track:")
			fmt.Printf("Starting radio based on: %s - %s\n", results.Tracks[0].Artists, results.Tracks[0].Name)
		}

		uris, err := client.GetRecommendations(trackID, 30)
		if err != nil {
			exitErr(cerrors.Provider("spotify", err))
		}
		if len(uris) == 0 {
			exitErr(&cerrors.Error{
				Code:    cerrors.CodeProviderError,
				Message: "no recommendations found for this track",
				Hint:    "try a different seed track",
			})
		}

		uris = append([]string{"spotify:track:" + trackID}, uris...)

		if err := client.PlayURIs(uris, ""); err != nil {
			exitSpotifyErr(err)
		}
		fmt.Printf("Playing radio (%d tracks)\n", len(uris))
	},
}

var spotifyPlaylistCmd = &cobra.Command{
	Use:   "playlist",
	Short: "Manage Spotify playlists",
}

var spotifyPlaylistListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your playlists",
	Run: func(cmd *cobra.Command, args []string) {
		client := mustPlayerClient()
		playlists, err := client.GetMyPlaylists(50)
		if err != nil {
			exitSpotifyErr(err)
		}
		if len(playlists) == 0 {
			fmt.Println("No playlists found.")
			return
		}
		for _, p := range playlists {
			fmt.Printf("%-40s %3d tracks  %s\n", p.Name, p.Items.Total, p.URI)
		}
	},
}

var spotifyPlaylistPlayCmd = &cobra.Command{
	Use:   "play <playlist name>",
	Short: "Play a playlist by name",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustPlayerClient()
		playlist := findPlaylist(client, strings.Join(args, " "))
		if err := client.PlayURI(playlist.URI, ""); err != nil {
			exitSpotifyErr(err)
		}
		fmt.Printf("Playing playlist: %s\n", playlist.Name)
	},
}

var spotifyPlaylistCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new playlist",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustPlayerClient()
		name := strings.Join(args, " ")

		pl, err := client.CreatePlaylist(name, false)
		if err != nil {
			exitSpotifyErr(err)
		}
		fmt.Printf("Created playlist: %s (%s)\n", pl.Name, pl.URI)
	},
}

var spotifyPlaylistAddCmd = &cobra.Command{
	Use:   "add <playlist name> <search query or spotify URI>",
	Short: "Add a track to a playlist",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustPlayerClient()
		playlist := findPlaylist(client, args[0])
		uri := resolveTrackURI(strings.Join(args[1:], " "))

		if err := client.AddToPlaylist(playlist.ID, []string{uri}); err != nil {
			exitSpotifyErr(err)
		}
		fmt.Printf("Added to %s.\n", playlist.Name)
	},
}

var spotifyPlaylistRemoveCmd = &cobra.Command{
	Use:   "remove <playlist name> <search query or spotify URI>",
	Short: "Remove a track from a playlist",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustPlayerClient()
		playlist := findPlaylist(client, args[0])
		uri := resolveTrackURI(strings.Join(args[1:], " "))

		if err := client.RemoveFromPlaylist(playlist.ID, []string{uri}); err != nil {
			exitSpotifyErr(err)
		}
		fmt.Printf("Removed from %s.\n", playlist.Name)
	},
}

var spotifyPlaylistShowCmd = &cobra.Command{
	Use:   "show <playlist name>",
	Short: "Show tracks in a playlist",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustPlayerClient()
		playlist := findPlaylist(client, strings.Join(args, " "))

		items, err := client.GetPlaylistItems(playlist.ID, 50)
		if err != nil {
			exitSpotifyErr(err)
		}

		fmt.Printf("%s (%d tracks)\n\n", playlist.Name, playlist.Items.Total)
		for i, t := range items {
			artists := make([]string, len(t.Item.Artists))
			for j, a := range t.Item.Artists {
				artists[j] = a.Name
			}
			fmt.Printf("%3d. %s - %s\n", i+1, strings.Join(artists, ", "), t.Item.Name)
		}
	},
}

// findPlaylist finds a user's playlist by fuzzy name match.
func findPlaylist(client *spotify.PlayerClient, query string) *spotify.UserPlaylist {
	playlists, err := client.GetMyPlaylists(50)
	if err != nil {
		exitSpotifyErr(err)
	}
	q := strings.ToLower(query)
	for _, p := range playlists {
		if strings.Contains(strings.ToLower(p.Name), q) {
			return &p
		}
	}
	available := make([]string, len(playlists))
	for i, p := range playlists {
		available[i] = p.Name
	}
	exitErr(&cerrors.Error{
		Code:      cerrors.CodeNotFound,
		Message:   fmt.Sprintf("no playlist matching '%s'", query),
		Available: available,
		Hint:      "use 'crib spotify playlist list' to see all playlists",
	})
	return nil
}

// resolveTrackURI resolves a search query or Spotify URI to a track URI.
func resolveTrackURI(query string) string {
	if strings.HasPrefix(query, "spotify:") {
		return query
	}
	searchClient := mustSpotifySearchClient()
	results, err := searchClient.Search(query, "track", 1)
	if err != nil {
		exitErr(cerrors.Provider("spotify", err))
	}
	if len(results.Tracks) == 0 {
		exitErr(&cerrors.Error{
			Code:    cerrors.CodeNotFound,
			Message: fmt.Sprintf("no track matching '%s'", query),
			Hint:    "try a different search query, or use a spotify:track:URI directly",
		})
	}
	fmt.Printf("Found: %s - %s\n", results.Tracks[0].Artists, results.Tracks[0].Name)
	return results.Tracks[0].URI
}

// exitSpotifyErr converts common Spotify API errors to structured errors.
func exitSpotifyErr(err error) {
	msg := err.Error()
	if strings.Contains(msg, "NO_ACTIVE_DEVICE") || strings.Contains(msg, "No active device") {
		exitErr(cerrors.NoSession("Spotify"))
	}
	if strings.Contains(msg, "401") {
		exitErr(cerrors.AuthExpired("spotify"))
	}
	if strings.Contains(msg, "403") && strings.Contains(msg, "Premium") {
		exitErr(&cerrors.Error{
			Code:    cerrors.CodeProviderError,
			Message: "Spotify Premium is required for playback control",
		})
	}
	exitErr(cerrors.Provider("spotify", err))
}

func mustPlayerClient() *spotify.PlayerClient {
	clientID, _, err := config.LoadSpotify()
	if err != nil {
		exitErr(cerrors.NotConfigured("Spotify"))
	}

	accessToken, refreshToken, expiresAt, err := config.LoadSpotifyToken()
	if err != nil {
		exitErr(cerrors.AuthExpired("spotify"))
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
	spotifyPlaylistCmd.AddCommand(spotifyPlaylistListCmd)
	spotifyPlaylistCmd.AddCommand(spotifyPlaylistPlayCmd)
	spotifyPlaylistCmd.AddCommand(spotifyPlaylistCreateCmd)
	spotifyPlaylistCmd.AddCommand(spotifyPlaylistAddCmd)
	spotifyPlaylistCmd.AddCommand(spotifyPlaylistRemoveCmd)
	spotifyPlaylistCmd.AddCommand(spotifyPlaylistShowCmd)
	spotifyCmd.AddCommand(spotifyPlaylistCmd)
	rootCmd.AddCommand(spotifyCmd)
}
