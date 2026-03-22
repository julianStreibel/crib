---
name: home
description: Control smart home devices (lights, switches, Sonos speakers, Spotify). Use when the user asks to turn on/off lights, play music, control volume, or interact with any smart home device.
allowed-tools: Bash(crib *)
---

# Smart Home Control

You have access to `crib` which controls smart home devices. Use it to fulfill user requests about lights, music, speakers, and other devices.

## Quick Reference

### Lights (IKEA TRÅDFRI)

```bash
crib lights list                          # List all lights with state/brightness
crib lights on <id|name>                  # Turn on (by ID or fuzzy name match)
crib lights off <id|name>                 # Turn off
crib lights toggle <id|name>              # Toggle on/off
crib lights dim <id|name> <0-100>         # Set brightness percentage
```

### Switches & Plugs (IKEA TRÅDFRI)

```bash
crib switches list                        # List all plugs with state
crib switches on <id|name>                # Turn on
crib switches off <id|name>               # Turn off
crib switches toggle <id|name>            # Toggle on/off
```

### Sonos Speakers

```bash
crib sonos list                           # List speakers with state/volume/track
crib sonos play <room>                    # Resume playback
crib sonos pause <room>                   # Pause
crib sonos stop <room>                    # Stop
crib sonos next <room>                    # Next track
crib sonos prev <room>                    # Previous track
crib sonos volume <room>                  # Get volume
crib sonos volume <room> <0-100|up|down>  # Set volume
crib sonos mute <room>                    # Toggle mute
crib sonos group <room> <coordinator>      # Add speaker to another's group
crib sonos ungroup <room>                 # Make speaker standalone
crib sonos group-all                      # Group all speakers together
crib sonos ungroup-all                    # Make all speakers standalone
crib sonos repeat <room> <one|all|off>    # Set repeat mode
crib sonos shuffle <room> <on|off>        # Set shuffle mode
crib sonos search "<query>"               # Search Spotify (results with URIs)
crib sonos play-track <room> <spotify-uri> # Play specific Spotify URI on speaker
crib sonos play-search <room> "<query>"   # Search + play top result on speaker
```

### Spotify Connect (controls any Spotify device)

```bash
crib spotify status                       # Current track, device, progress
crib spotify devices                      # List Spotify Connect devices
crib spotify play                         # Resume playback
crib spotify play "<query>"               # Search + play on active device
crib spotify play <spotify:uri>           # Play specific URI
crib spotify pause                        # Pause
crib spotify next                         # Next track
crib spotify prev                         # Previous track
crib spotify volume <0-100|up|down>       # Set volume
crib spotify transfer <device-name>       # Move playback to another device
crib spotify repeat <track|playlist|off>  # Set repeat mode
crib spotify shuffle <on|off>             # Set shuffle mode
crib spotify queue "<query>"              # Add a track to the queue
crib spotify radio "<query>"              # Start radio based on a song (plays similar tracks)
```

## Guidelines

- **Always list first** when unsure about device names or IDs. Run `lights list`, `sonos list`, etc.
- **Fuzzy name matching** works for lights, switches, and Sonos rooms — use readable names like `kugel`, `wohnzimmer`, `küche`.
- **Sonos vs Spotify**: Use `sonos` commands to play on Sonos speakers directly. Use `spotify` commands to control the user's active Spotify session on any device (phone, laptop, etc.).
- **Grouped Sonos speakers**: Playback commands sent to any speaker in a group affect the whole group. Volume is per-speaker.
- **Unreachable devices**: Some lights may show as "unreachable" — these are powered off at the switch, nothing you can do.
- When asked to "play music", prefer `spotify play "<query>"` if the user has an active Spotify session, or `sonos play-search <room> "<query>"` if they want it on a specific speaker.
- When the user says "turn off the lights", turn off all reachable lights that are currently on.
- Be conversational about what you did: "Turned on the Kugel and dimmed it to 50%".

## Setup

If `crib` is not installed or not configured, guide the user:

```bash
# Install
go install github.com/julianStreibel/crib@latest

# Interactive setup (TRÅDFRI pairing, Spotify credentials + login)
crib setup
```
