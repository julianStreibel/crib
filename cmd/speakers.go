package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/julianStreibel/crib/internal/config"
	cerrors "github.com/julianStreibel/crib/internal/errors"
	"github.com/julianStreibel/crib/internal/sonos"
	"github.com/julianStreibel/crib/internal/spotify"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var speakersCmd = &cobra.Command{
	Use:   "speakers",
	Short: "Control speakers (Sonos)",
}

var speakersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all speakers and their states",
	Run: func(cmd *cobra.Command, args []string) {
		speakers := mustDiscoverSpeakers()

		type result struct {
			speaker *sonos.Speaker
			state   *sonos.PlaybackState
			err     error
		}
		results := make([]result, len(speakers))
		g := new(errgroup.Group)
		for i, s := range speakers {
			results[i].speaker = s
			g.Go(func() error {
				state, err := s.GetPlaybackState()
				results[i].state = state
				results[i].err = err
				return nil
			})
		}
		_ = g.Wait()

		for _, r := range results {
			group := ""
			if !r.speaker.IsCoordinator {
				group = " [grouped]"
			}
			if r.err != nil {
				fmt.Printf("%-20s %-12s vol:?   %s%s\n", r.speaker.Room, "error", r.speaker.Model, group)
				continue
			}
			track := r.state.TrackString()
			if track != "" {
				track = " | " + track
			}
			fmt.Printf("%-20s %-12s vol:%-3d %s%s%s\n",
				r.speaker.Room, r.state.StateString(), r.state.Volume, r.speaker.Model, group, track)
		}
	},
}

var speakersPlayCmd = &cobra.Command{
	Use:   "play <room> [query]",
	Short: "Resume playback, or search and play a track",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSpeaker(args[0])

		if len(args) == 1 {
			if err := speaker.Play(); err != nil {
				exitErr(cerrors.Provider("sonos", err))
			}
			fmt.Printf("Playing on %s\n", speaker.Room)
			return
		}

		// Search and play
		query := strings.Join(args[1:], " ")
		client := mustSpotifySearchClient()

		results, err := client.Search(query, "track", 1)
		if err != nil {
			exitErr(cerrors.Provider("spotify", err))
		}
		if len(results.Tracks) == 0 {
			exitErr(cerrors.NotFound("track", query, nil))
		}

		track := results.Tracks[0]
		if err := speaker.PlaySpotifyURI(track.URI); err != nil {
			exitErr(cerrors.Provider("sonos", err))
		}
		fmt.Printf("Playing %s - %s on %s\n", track.Artists, track.Name, speaker.Room)
	},
}

var speakersPauseCmd = &cobra.Command{
	Use:   "pause <room>",
	Short: "Pause playback",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSpeaker(args[0])
		if err := speaker.Pause(); err != nil {
			exitErr(cerrors.Provider("sonos", err))
		}
		fmt.Printf("Paused %s\n", speaker.Room)
	},
}

var speakersStopCmd = &cobra.Command{
	Use:   "stop <room>",
	Short: "Stop playback",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSpeaker(args[0])
		if err := speaker.Stop(); err != nil {
			exitErr(cerrors.Provider("sonos", err))
		}
		fmt.Printf("Stopped %s\n", speaker.Room)
	},
}

var speakersNextCmd = &cobra.Command{
	Use:   "next <room>",
	Short: "Skip to next track",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSpeaker(args[0])
		if err := speaker.Next(); err != nil {
			exitErr(cerrors.Provider("sonos", err))
		}
		fmt.Printf("Next track on %s\n", speaker.Room)
	},
}

var speakersPrevCmd = &cobra.Command{
	Use:   "prev <room>",
	Short: "Go to previous track",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSpeaker(args[0])
		if err := speaker.Previous(); err != nil {
			exitErr(cerrors.Provider("sonos", err))
		}
		fmt.Printf("Previous track on %s\n", speaker.Room)
	},
}

var speakersVolumeCmd = &cobra.Command{
	Use:   "volume <room> [0-100|up|down]",
	Short: "Get or set volume",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSpeaker(args[0])

		if len(args) == 1 {
			vol, err := speaker.GetVolume()
			if err != nil {
				exitErr(cerrors.Provider("sonos", err))
			}
			fmt.Printf("%s volume: %d\n", speaker.Room, vol)
			return
		}

		targetVol := parseVolume(args[1], func() int {
			vol, err := speaker.GetVolume()
			if err != nil {
				exitErr(cerrors.Provider("sonos", err))
			}
			return vol
		})

		if err := speaker.SetVolume(targetVol); err != nil {
			exitErr(cerrors.Provider("sonos", err))
		}
		fmt.Printf("Set %s volume to %d\n", speaker.Room, targetVol)
	},
}

var speakersMuteCmd = &cobra.Command{
	Use:   "mute <room>",
	Short: "Toggle mute on a speaker",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSpeaker(args[0])
		muted, err := speaker.GetMute()
		if err != nil {
			exitErr(cerrors.Provider("sonos", err))
		}
		if err := speaker.SetMute(!muted); err != nil {
			exitErr(cerrors.Provider("sonos", err))
		}
		if muted {
			fmt.Printf("Unmuted %s\n", speaker.Room)
		} else {
			fmt.Printf("Muted %s\n", speaker.Room)
		}
	},
}

var speakersRepeatCmd = &cobra.Command{
	Use:   "repeat <room> <one|all|off>",
	Short: "Set repeat mode",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSpeaker(args[0])
		var mode string
		switch args[1] {
		case "one", "track", "song":
			mode = "REPEAT_ONE"
		case "all", "playlist":
			mode = "REPEAT_ALL"
		case "off", "none":
			mode = "NORMAL"
		default:
			exitErr(cerrors.InvalidArgWithHint(
				fmt.Sprintf("unknown repeat mode '%s'", args[1]),
				"usage: crib speakers repeat <room> <one|all|off>",
			))
		}
		if err := speaker.SetPlayMode(mode); err != nil {
			exitErr(cerrors.Provider("sonos", err))
		}
		fmt.Printf("Set %s repeat to %s\n", speaker.Room, args[1])
	},
}

var speakersShuffleCmd = &cobra.Command{
	Use:   "shuffle <room> <on|off>",
	Short: "Set shuffle mode",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSpeaker(args[0])

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
			exitErr(cerrors.Provider("sonos", err))
		}
		if on {
			fmt.Printf("Shuffle on for %s\n", speaker.Room)
		} else {
			fmt.Printf("Shuffle off for %s\n", speaker.Room)
		}
	},
}

var speakersSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search Spotify for tracks, playlists, and albums",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := strings.Join(args, " ")
		client := mustSpotifySearchClient()

		results, err := client.Search(query, "", 5)
		if err != nil {
			exitErr(cerrors.Provider("spotify", err))
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
				fmt.Printf("  %-50s %s\n", a.Artists+" - "+a.Name, a.URI)
			}
		}
	},
}

var speakersPlayTrackCmd = &cobra.Command{
	Use:   "play-track <room> <spotify-uri>",
	Short: "Play a Spotify track, album, or playlist on a speaker",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSpeaker(args[0])
		uri := args[1]

		if err := speaker.PlaySpotifyURI(uri); err != nil {
			exitErr(cerrors.Provider("sonos", err))
		}
		fmt.Printf("Playing %s on %s\n", uri, speaker.Room)
	},
}

// Helpers

func mustSpotifySearchClient() *spotify.Client {
	clientID, clientSecret, err := config.LoadSpotify()
	if err != nil {
		exitErr(cerrors.NotConfigured("Spotify"))
	}
	return spotify.NewClient(clientID, clientSecret)
}

func mustDiscoverSpeakers() []*sonos.Speaker {
	speakers, err := sonos.Discover(3 * time.Second)
	if err != nil {
		exitErr(cerrors.Provider("sonos", err))
	}
	if len(speakers) == 0 {
		exitErr(&cerrors.Error{
			Code:    cerrors.CodeNetwork,
			Message: "no speakers found on the network",
			Hint:    "make sure your speakers are powered on and on the same network",
		})
	}
	return speakers
}

func mustFindSpeaker(name string) *sonos.Speaker {
	speakers := mustDiscoverSpeakers()
	return findSpeakerIn(speakers, name)
}

func findSpeakerIn(speakers []*sonos.Speaker, name string) *sonos.Speaker {
	lower := strings.ToLower(name)

	for _, s := range speakers {
		if strings.ToLower(s.Room) == lower {
			return s
		}
	}
	for _, s := range speakers {
		if strings.Contains(strings.ToLower(s.Room), lower) {
			return s
		}
	}

	available := make([]string, len(speakers))
	for i, s := range speakers {
		available[i] = fmt.Sprintf("%s (%s)", s.Room, s.Model)
	}
	exitErr(cerrors.NotFound("speaker", name, available))
	return nil
}

func parseVolume(arg string, getCurrentVol func() int) int {
	switch arg {
	case "up":
		return getCurrentVol() + 5
	case "down":
		return getCurrentVol() - 5
	default:
		v, err := strconv.Atoi(arg)
		if err != nil {
			exitErr(cerrors.InvalidArgWithHint(
				fmt.Sprintf("invalid volume '%s'", arg),
				"usage: volume <room> <0-100|up|down>",
			))
		}
		return v
	}
}

func init() {
	speakersCmd.AddCommand(speakersListCmd)
	speakersCmd.AddCommand(speakersPlayCmd)
	speakersCmd.AddCommand(speakersPauseCmd)
	speakersCmd.AddCommand(speakersStopCmd)
	speakersCmd.AddCommand(speakersNextCmd)
	speakersCmd.AddCommand(speakersPrevCmd)
	speakersCmd.AddCommand(speakersVolumeCmd)
	speakersCmd.AddCommand(speakersMuteCmd)
	speakersCmd.AddCommand(speakersRepeatCmd)
	speakersCmd.AddCommand(speakersShuffleCmd)
	speakersCmd.AddCommand(speakersSearchCmd)
	speakersCmd.AddCommand(speakersPlayTrackCmd)
	rootCmd.AddCommand(speakersCmd)
}
