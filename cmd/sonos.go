package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/julianStreibel/crib/internal/config"
	"github.com/julianStreibel/crib/internal/sonos"
	"github.com/julianStreibel/crib/internal/spotify"
	"github.com/spf13/cobra"
)

var sonosCmd = &cobra.Command{
	Use:   "sonos",
	Short: "Control Sonos speakers",
}

var sonosListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all Sonos speakers and their states",
	Run: func(cmd *cobra.Command, args []string) {
		speakers := mustDiscoverSonos()

		for _, s := range speakers {
			group := ""
			if !s.IsCoordinator {
				group = " [grouped]"
			}
			state, err := s.GetPlaybackState()
			if err != nil {
				fmt.Printf("%-20s %-12s vol:?   %s%s\n", s.Room, "error", s.Model, group)
				continue
			}
			track := state.TrackString()
			if track != "" {
				track = " | " + track
			}
			fmt.Printf("%-20s %-12s vol:%-3d %s%s%s\n",
				s.Room, state.StateString(), state.Volume, s.Model, group, track)
		}
	},
}

var sonosPlayCmd = &cobra.Command{
	Use:   "play [room]",
	Short: "Resume playback",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSonos(args[0])
		if err := speaker.Play(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Playing on %s\n", speaker.Room)
	},
}

var sonosPauseCmd = &cobra.Command{
	Use:   "pause [room]",
	Short: "Pause playback",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSonos(args[0])
		if err := speaker.Pause(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Paused %s\n", speaker.Room)
	},
}

var sonosStopCmd = &cobra.Command{
	Use:   "stop [room]",
	Short: "Stop playback",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSonos(args[0])
		if err := speaker.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Stopped %s\n", speaker.Room)
	},
}

var sonosNextCmd = &cobra.Command{
	Use:   "next [room]",
	Short: "Skip to next track",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSonos(args[0])
		if err := speaker.Next(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Next track on %s\n", speaker.Room)
	},
}

var sonosPrevCmd = &cobra.Command{
	Use:   "prev [room]",
	Short: "Go to previous track",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSonos(args[0])
		if err := speaker.Previous(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Previous track on %s\n", speaker.Room)
	},
}

var sonosVolumeCmd = &cobra.Command{
	Use:   "volume [room] [level|up|down]",
	Short: "Get or set volume (0-100, up, down)",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSonos(args[0])

		if len(args) == 1 {
			vol, err := speaker.GetVolume()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("%s volume: %d\n", speaker.Room, vol)
			return
		}

		action := args[1]
		var targetVol int

		switch action {
		case "up":
			vol, err := speaker.GetVolume()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			targetVol = vol + 5
		case "down":
			vol, err := speaker.GetVolume()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			targetVol = vol - 5
		default:
			v, err := strconv.Atoi(action)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: volume must be 0-100, 'up', or 'down'\n")
				os.Exit(1)
			}
			targetVol = v
		}

		if err := speaker.SetVolume(targetVol); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Set %s volume to %d\n", speaker.Room, targetVol)
	},
}

var sonosMuteCmd = &cobra.Command{
	Use:   "mute [room]",
	Short: "Toggle mute on a speaker",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSonos(args[0])
		muted, err := speaker.GetMute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if err := speaker.SetMute(!muted); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if muted {
			fmt.Printf("Unmuted %s\n", speaker.Room)
		} else {
			fmt.Printf("Muted %s\n", speaker.Room)
		}
	},
}

var sonosGroupCmd = &cobra.Command{
	Use:   "group <room> <coordinator-room>",
	Short: "Add a speaker to another speaker's group",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		speakers := mustDiscoverSonos()
		speaker := findSonosIn(speakers, args[0])
		coordinator := findSonosIn(speakers, args[1])

		if err := speaker.JoinGroup(coordinator.UUID); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Added %s to %s's group\n", speaker.Room, coordinator.Room)
	},
}

var sonosUngroupCmd = &cobra.Command{
	Use:   "ungroup <room>",
	Short: "Remove a speaker from its group (make standalone)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSonos(args[0])
		if err := speaker.Ungroup(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Ungrouped %s\n", speaker.Room)
	},
}

var sonosGroupAllCmd = &cobra.Command{
	Use:   "group-all",
	Short: "Group all speakers together",
	Run: func(cmd *cobra.Command, args []string) {
		speakers := mustDiscoverSonos()
		if len(speakers) < 2 {
			fmt.Println("Need at least 2 speakers to group.")
			return
		}

		// Use the first coordinator we find, or the first speaker
		coordinator := speakers[0]
		for _, s := range speakers {
			if s.IsCoordinator {
				coordinator = s
				break
			}
		}

		count := 0
		for _, s := range speakers {
			if s.IP == coordinator.IP {
				continue
			}
			if err := s.JoinGroup(coordinator.UUID); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not group %s: %v\n", s.Room, err)
				continue
			}
			count++
		}
		fmt.Printf("Grouped %d speakers with %s\n", count, coordinator.Room)
	},
}

var sonosUngroupAllCmd = &cobra.Command{
	Use:   "ungroup-all",
	Short: "Ungroup all speakers (make all standalone)",
	Run: func(cmd *cobra.Command, args []string) {
		speakers := mustDiscoverSonos()
		count := 0
		for _, s := range speakers {
			if !s.IsCoordinator {
				if err := s.Ungroup(); err != nil {
					fmt.Fprintf(os.Stderr, "warning: could not ungroup %s: %v\n", s.Room, err)
					continue
				}
				count++
			}
		}
		fmt.Printf("Ungrouped %d speakers\n", count)
	},
}

var sonosRepeatCmd = &cobra.Command{
	Use:   "repeat <room> <one|all|off>",
	Short: "Set repeat mode on a Sonos speaker",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSonos(args[0])
		var mode string
		switch args[1] {
		case "one", "track", "song":
			mode = "REPEAT_ONE"
		case "all", "playlist":
			mode = "REPEAT_ALL"
		case "off", "none":
			mode = "NORMAL"
		default:
			fmt.Fprintf(os.Stderr, "error: mode must be one, all, or off\n")
			os.Exit(1)
		}
		if err := speaker.SetPlayMode(mode); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Set %s repeat to %s\n", speaker.Room, args[1])
	},
}

var sonosShuffleCmd = &cobra.Command{
	Use:   "shuffle <room> <on|off>",
	Short: "Set shuffle mode on a Sonos speaker",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSonos(args[0])

		// Get current mode to preserve repeat state
		currentMode, err := speaker.GetPlayMode()
		if err != nil {
			currentMode = "NORMAL"
		}

		on := args[1] == "on" || args[1] == "true"
		var mode string
		if on {
			if currentMode == "REPEAT_ALL" || currentMode == "SHUFFLE" {
				mode = "SHUFFLE"
			} else {
				mode = "SHUFFLE_NOREPEAT"
			}
		} else {
			if currentMode == "SHUFFLE" {
				mode = "REPEAT_ALL"
			} else {
				mode = "NORMAL"
			}
		}

		if err := speaker.SetPlayMode(mode); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if on {
			fmt.Printf("Shuffle on for %s\n", speaker.Room)
		} else {
			fmt.Printf("Shuffle off for %s\n", speaker.Room)
		}
	},
}

var sonosSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search Spotify for tracks, playlists, and albums",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := strings.Join(args, " ")
		client := mustSpotifyClient()

		results, err := client.Search(query, "", 5)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		if len(results.Tracks) > 0 {
			fmt.Println("Tracks:")
			for _, t := range results.Tracks {
				fmt.Printf("  %-50s %-30s %s\n", t.Artists+" - "+t.Name, t.Album, t.URI)
			}
		}
		if len(results.Playlists) > 0 {
			fmt.Println("Playlists:")
			for _, p := range results.Playlists {
				fmt.Printf("  %-50s by %-25s %s\n", p.Name, p.Owner, p.URI)
			}
		}
		if len(results.Albums) > 0 {
			fmt.Println("Albums:")
			for _, a := range results.Albums {
				fmt.Printf("  %-50s %-30s %s\n", a.Artists+" - "+a.Name, "", a.URI)
			}
		}
	},
}

var sonosPlayTrackCmd = &cobra.Command{
	Use:   "play-track <room> <spotify-uri>",
	Short: "Play a Spotify track, album, or playlist on a speaker",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSonos(args[0])
		uri := args[1]

		if err := speaker.PlaySpotifyURI(uri); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Playing %s on %s\n", uri, speaker.Room)
	},
}

var sonosPlaySearchCmd = &cobra.Command{
	Use:   "play-search <room> <query>",
	Short: "Search Spotify and play the top result on a speaker",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSonos(args[0])
		query := strings.Join(args[1:], " ")
		client := mustSpotifyClient()

		results, err := client.Search(query, "track", 1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		if len(results.Tracks) == 0 {
			fmt.Fprintf(os.Stderr, "No tracks found for %q\n", query)
			os.Exit(1)
		}

		track := results.Tracks[0]
		if err := speaker.PlaySpotifyURI(track.URI); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Playing %s - %s on %s\n", track.Artists, track.Name, speaker.Room)
	},
}

func mustSpotifyClient() *spotify.Client {
	clientID, clientSecret, err := config.LoadSpotify()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return spotify.NewClient(clientID, clientSecret)
}

func mustDiscoverSonos() []*sonos.Speaker {
	speakers, err := sonos.Discover(3 * time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error discovering Sonos: %v\n", err)
		os.Exit(1)
	}
	if len(speakers) == 0 {
		fmt.Fprintf(os.Stderr, "No Sonos speakers found on the network.\n")
		os.Exit(1)
	}
	return speakers
}

func mustFindSonos(name string) *sonos.Speaker {
	speakers := mustDiscoverSonos()
	return findSonosIn(speakers, name)
}

func findSonosIn(speakers []*sonos.Speaker, name string) *sonos.Speaker {
	name = strings.ToLower(name)

	// Exact match first
	for _, s := range speakers {
		if strings.ToLower(s.Room) == name {
			return s
		}
	}
	// Substring match
	for _, s := range speakers {
		if strings.Contains(strings.ToLower(s.Room), name) {
			return s
		}
	}

	fmt.Fprintf(os.Stderr, "error: no Sonos speaker matching %q\nAvailable rooms:", name)
	for _, s := range speakers {
		fmt.Fprintf(os.Stderr, "  %s\n", s.Room)
	}
	os.Exit(1)
	return nil
}

func init() {
	sonosCmd.AddCommand(sonosListCmd)
	sonosCmd.AddCommand(sonosPlayCmd)
	sonosCmd.AddCommand(sonosPauseCmd)
	sonosCmd.AddCommand(sonosStopCmd)
	sonosCmd.AddCommand(sonosNextCmd)
	sonosCmd.AddCommand(sonosPrevCmd)
	sonosCmd.AddCommand(sonosVolumeCmd)
	sonosCmd.AddCommand(sonosMuteCmd)
	sonosCmd.AddCommand(sonosGroupCmd)
	sonosCmd.AddCommand(sonosUngroupCmd)
	sonosCmd.AddCommand(sonosGroupAllCmd)
	sonosCmd.AddCommand(sonosUngroupAllCmd)
	sonosCmd.AddCommand(sonosRepeatCmd)
	sonosCmd.AddCommand(sonosShuffleCmd)
	sonosCmd.AddCommand(sonosSearchCmd)
	sonosCmd.AddCommand(sonosPlayTrackCmd)
	sonosCmd.AddCommand(sonosPlaySearchCmd)
	rootCmd.AddCommand(sonosCmd)
}
