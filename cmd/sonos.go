package cmd

import (
	"fmt"

	cerrors "github.com/julianStreibel/crib/internal/errors"
	"github.com/spf13/cobra"
)

// Sonos-specific commands that don't generalize to other speaker providers.

var sonosCmd = &cobra.Command{
	Use:   "sonos",
	Short: "Sonos-specific commands (grouping)",
}

var sonosGroupCmd = &cobra.Command{
	Use:   "group <room> <coordinator-room>",
	Short: "Add a speaker to another speaker's group",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		speakers := mustDiscoverSpeakers()
		speaker := findSpeakerIn(speakers, args[0])
		coordinator := findSpeakerIn(speakers, args[1])

		if err := speaker.JoinGroup(coordinator.UUID); err != nil {
			exitErr(cerrors.Provider("sonos", err))
		}
		fmt.Printf("Added %s to %s's group\n", speaker.Room, coordinator.Room)
	},
}

var sonosUngroupCmd = &cobra.Command{
	Use:   "ungroup <room>",
	Short: "Remove a speaker from its group (make standalone)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		speaker := mustFindSpeaker(args[0])
		if err := speaker.Ungroup(); err != nil {
			exitErr(cerrors.Provider("sonos", err))
		}
		fmt.Printf("Ungrouped %s\n", speaker.Room)
	},
}

var sonosGroupAllCmd = &cobra.Command{
	Use:   "group-all",
	Short: "Group all speakers together",
	Run: func(cmd *cobra.Command, args []string) {
		speakers := mustDiscoverSpeakers()
		if len(speakers) < 2 {
			fmt.Println("Need at least 2 speakers to group.")
			return
		}

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
				fmt.Printf("warning: could not group %s: %v\n", s.Room, err)
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
		speakers := mustDiscoverSpeakers()
		count := 0
		for _, s := range speakers {
			if !s.IsCoordinator {
				if err := s.Ungroup(); err != nil {
					fmt.Printf("warning: could not ungroup %s: %v\n", s.Room, err)
					continue
				}
				count++
			}
		}
		fmt.Printf("Ungrouped %d speakers\n", count)
	},
}

func init() {
	sonosCmd.AddCommand(sonosGroupCmd)
	sonosCmd.AddCommand(sonosUngroupCmd)
	sonosCmd.AddCommand(sonosGroupAllCmd)
	sonosCmd.AddCommand(sonosUngroupAllCmd)
	rootCmd.AddCommand(sonosCmd)
}
