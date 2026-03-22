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

### Supported devices

| Integration | What you need | Config required |
|---|---|---|
| **IKEA TRÅDFRI** | Gateway + security code from device | Yes (one-time pairing) |
| **Sonos** | Speakers on your network | No (auto-discovered) |
| **Spotify** | Spotify Developer app + Premium account | Yes (API credentials + login) |

## Usage

### Lights & Switches (IKEA TRÅDFRI)

```bash
crib lights list
crib lights on kitchen
crib lights dim bedroom 50
crib lights toggle office

crib switches list
crib switches on "salt lamp"
```

Devices can be targeted by ID or fuzzy name match.

### Sonos

```bash
crib sonos list                          # All speakers with status
crib sonos play-search kitchen "chill"   # Search Spotify + play on speaker
crib sonos volume kitchen 30
crib sonos pause kitchen

# Grouping
crib sonos group kitchen living-room     # Add to group
crib sonos ungroup kitchen               # Make standalone
crib sonos group-all                     # Group everything

# Playback modes
crib sonos repeat kitchen one            # Repeat track
crib sonos shuffle kitchen on
```

### Spotify Connect

Control your active Spotify session on any device (phone, laptop, speaker).

```bash
crib spotify login                       # One-time browser auth
crib spotify status                      # What's playing
crib spotify play "bohemian rhapsody"    # Search + play
crib spotify pause
crib spotify next
crib spotify volume 50
crib spotify transfer macbook            # Move to another device
crib spotify radio "daft punk"           # Play similar tracks
crib spotify queue "another song"        # Add to queue
crib spotify repeat track                # Loop current song
crib spotify shuffle on
```

### Search

```bash
crib sonos search "lofi beats"           # Search Spotify, get URIs
crib sonos play-track kitchen spotify:playlist:37i9dQZF1DX2TRYkJECvfC
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
├── tradfri/    # IKEA TRÅDFRI (CoAP/DTLS)
├── sonos/      # Sonos (UPnP/SOAP)
└── spotify/    # Spotify (Web API + Connect)
```

Each integration is self-contained. To add a new one:

1. Create `internal/<name>/` with client code
2. Add commands in `cmd/<name>.go`
3. Add config loading in `internal/config/`
4. Add a setup step in `cmd/setup.go`
5. Update the plugin skill in `plugin/skills/home/SKILL.md`

## License

MIT
