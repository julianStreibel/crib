---
name: home
description: Control smart home devices (lights, switches, speakers, Spotify). Use when the user asks to turn on/off lights, play music, control volume, or interact with any smart home device.
allowed-tools: Bash(crib *)
---

# Smart Home Control

You have access to `crib` which controls smart home devices. Use it to fulfill user requests about lights, music, speakers, and other devices.

## Quick Start

```bash
crib status                                  # Everything at a glance
```

## Devices (lights, switches, plugs)

All device types from all providers in one unified view.

```bash
crib devices list                            # List all devices with state
crib devices on <name> [name...]             # Turn on (fuzzy name match)
crib devices off <name> [name...]            # Turn off
crib devices toggle <name> [name...]         # Toggle on/off
crib devices on --all                        # Turn on all reachable lights and plugs
crib devices off --all                       # Turn off all reachable lights and plugs
crib devices dim <name> <0-100>              # Set brightness (dimmable lights only)
crib devices temp <name> <kelvin|warm|neutral|cool>  # Set color temperature
```

## Speakers

Generic speaker commands — works for any speaker provider (currently Sonos).

```bash
crib speakers list                           # All speakers with status/volume/track
crib speakers play <room> [query]            # Resume, or search+play a track
crib speakers pause <room>                   # Pause
crib speakers stop <room>                    # Stop
crib speakers next <room>                    # Next track
crib speakers prev <room>                    # Previous track
crib speakers volume <room> [0-100|up|down]  # Get or set volume
crib speakers mute <room>                    # Toggle mute
crib speakers repeat <room> <one|all|off>    # Set repeat mode
crib speakers shuffle <room> <on|off>        # Set shuffle mode
crib speakers search <query>                 # Search Spotify, get URIs
crib speakers play-track <room> <spotify-uri> # Play specific Spotify URI
```

## Sonos-Specific (grouping)

```bash
crib sonos group <room> <coordinator>        # Add speaker to a group
crib sonos ungroup <room>                    # Make speaker standalone
crib sonos group-all                         # Group all speakers together
crib sonos ungroup-all                       # Ungroup all speakers
```

## Spotify Connect

Control your active Spotify session on any device (phone, laptop, speaker).

```bash
crib spotify status                          # Current track, device, progress
crib spotify devices                         # List Spotify Connect devices
crib spotify play [query or spotify:uri]     # Resume, search+play, or play URI
crib spotify pause                           # Pause
crib spotify next                            # Next track
crib spotify prev                            # Previous track
crib spotify volume <0-100|up|down>          # Set volume
crib spotify transfer <device-name>          # Move playback to another device
crib spotify repeat <track|playlist|off>     # Set repeat mode
crib spotify shuffle <on|off>                # Set shuffle mode
crib spotify queue <query>                   # Add track to queue
crib spotify radio <query>                   # Play similar tracks based on a song
```

## Guidelines

- **Start with `crib status`** to understand the current state before taking action.
- **Fuzzy name matching** works everywhere — use readable names like `kugel`, `wohnzimmer`, `küche`.
- **`speakers` vs `spotify`**: Use `speakers` to play on specific Sonos speakers. Use `spotify` to control the user's active Spotify session on any device.
- **Grouped speakers**: Playback commands on any grouped speaker affect the whole group. Volume is per-speaker. To play on one room only, ungroup it first with `crib sonos ungroup <room>`.
- **Unreachable devices**: Some show as "unreachable" — they're powered off at the switch, nothing you can do remotely.
- When the user says "turn off the lights", use `crib devices off --all`. For multiple specific devices, pass all names in one command: `crib devices off Flur Kugel Stehlampe`.
- When asked to "play music", prefer `spotify play` if there's an active session, or `speakers play <room>` for a specific room.
- **Errors include hints** — read them to know what to do next (e.g. "run crib setup", "device is unreachable").
- Be conversational: "Turned on the Kugel and dimmed it to 50%", not "Executed crib devices on kugel".

## Setup

If `crib` is not installed or configured:

```bash
# Install
go install github.com/julianStreibel/crib@latest

# Or via Homebrew
brew install julianStreibel/tap/crib

# Interactive setup
crib setup
```
