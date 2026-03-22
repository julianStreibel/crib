# crib

Smart home CLI for humans and LLM agents. Control your IKEA TRÅDFRI lights, Sonos speakers, and Spotify — all from the terminal.

Built to be used directly or by AI agents like Claude Code via the included plugin.

## Install

```bash
# Go
go install github.com/julianStreibel/crib@latest

# Homebrew (macOS)
brew install julianStreibel/tap/crib

# Or download a binary from GitHub Releases
```

## Setup

```bash
crib setup
```

The interactive wizard walks you through each integration. All steps are optional — skip what you don't need.

### Supported integrations

| Integration | What you need | Config required |
|---|---|---|
| **IKEA TRÅDFRI** | Gateway + security code from device | Yes (one-time pairing) |
| **Sonos** | Speakers on your network | No (auto-discovered) |
| **Spotify** | Spotify Developer app + Premium account | Yes (API credentials + login) |

## Usage

### Status

```bash
crib status                              # Everything at a glance
```

### Devices (lights, switches, plugs)

All device types from all providers in one view.

```bash
crib devices list                        # List all devices with state
crib devices on kitchen                  # Turn on (fuzzy name match)
crib devices off kitchen
crib devices toggle office
crib devices dim bedroom 50              # Set brightness (dimmable lights only)
```

### Speakers

Generic speaker commands — currently Sonos, extensible to other providers.

```bash
crib speakers list                       # All speakers with status/volume/track
crib speakers play kitchen "chill vibes" # Search + play on a speaker
crib speakers pause kitchen
crib speakers volume kitchen 30
crib speakers next kitchen
crib speakers prev kitchen
crib speakers repeat kitchen one         # Repeat: one, all, off
crib speakers shuffle kitchen on
crib speakers mute kitchen
crib speakers search "lofi beats"        # Search Spotify, get URIs
crib speakers play-track kitchen spotify:playlist:37i9dQZF1DX2TRYkJECvfC
```

### Sonos-Specific (grouping)

```bash
crib sonos group kitchen living-room     # Add to group
crib sonos ungroup kitchen               # Make standalone
crib sonos group-all                     # Group everything
crib sonos ungroup-all                   # Ungroup everything
```

### Spotify Connect

Control your active Spotify session on any device (phone, laptop, speaker).

```bash
crib spotify login                       # One-time browser auth
crib spotify status                      # What's playing
crib spotify devices                     # List Spotify Connect devices
crib spotify play "bohemian rhapsody"    # Search + play
crib spotify play                        # Resume playback
crib spotify pause
crib spotify next
crib spotify prev
crib spotify volume 50
crib spotify transfer macbook            # Move to another device
crib spotify radio "daft punk"           # Play similar tracks
crib spotify queue "another song"        # Add to queue
crib spotify repeat track                # Loop: track, playlist, off
crib spotify shuffle on
```

## Claude Code Plugin

crib ships as a Claude Code plugin. Install it to let Claude control your home:

```
/plugin
→ Enter: julianStreibel/crib
→ Enable crib
```

Then just ask Claude: *"turn off the lights and play some jazz in the kitchen"*

## Configuration

Config is stored at `~/.config/crib/config.yaml`. Environment variables with the `CRIB_` prefix override config values.

```yaml
# IKEA TRÅDFRI
tradfri_host: 192.168.1.100
tradfri_identity: crib-macbook
tradfri_psk: <generated-during-pairing>

# Spotify
spotify_client_id: <your-client-id>
spotify_client_secret: <your-client-secret>
spotify_access_token: <auto-managed>
spotify_refresh_token: <auto-managed>
```

## Adding new integrations

The codebase is organized by integration:

```
internal/
├── device/     # Core interfaces (Device, Speaker, MusicService)
├── tradfri/    # IKEA TRÅDFRI (CoAP/DTLS)
├── sonos/      # Sonos (UPnP/SOAP)
├── spotify/    # Spotify (Web API + Connect)
├── discovery/  # mDNS, SSDP discovery
├── errors/     # Structured error types with hints
└── config/     # Configuration management
```

Each integration is self-contained. To add a new one:

1. Implement `DeviceProvider` or `SpeakerProvider` from `internal/device/`
2. Create `internal/<name>/` with client code
3. Add commands in `cmd/<name>.go`
4. Add config loading in `internal/config/`
5. Add a setup step in `cmd/setup.go`
6. Update the plugin skill in `plugin/skills/home/SKILL.md`

## License

MIT
