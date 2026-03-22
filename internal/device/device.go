// Package device defines the core interfaces for smart home integrations.
//
// Adding a new integration (e.g. Philips Hue) requires:
//  1. Implement DeviceProvider (for lights/switches) or SpeakerProvider (for audio)
//  2. Register it via the registry
//  3. Add CLI commands
package device

// DeviceType categorizes a device.
type DeviceType int

const (
	TypeLight DeviceType = iota
	TypeSwitch
	TypeSensor
)

func (t DeviceType) String() string {
	switch t {
	case TypeLight:
		return "light"
	case TypeSwitch:
		return "switch"
	case TypeSensor:
		return "sensor"
	default:
		return "unknown"
	}
}

// DeviceState holds the current state of a device.
type DeviceState struct {
	On         bool
	Reachable  bool
	Brightness int // 0-100, only meaningful if Dimmable is true
	Dimmable   bool
}

// Device represents a controllable smart home device (light, switch, plug).
type Device interface {
	ID() string
	Name() string
	Type() DeviceType
	Provider() string // "tradfri", "hue", etc.
	Model() string
	State() DeviceState

	TurnOn() error
	TurnOff() error
	Toggle() error
	SetBrightness(percent int) error // returns error if not dimmable
}

// DeviceProvider discovers and manages devices from a single integration.
type DeviceProvider interface {
	Name() string       // provider identifier: "tradfri", "hue"
	IsConfigured() bool // true if credentials/config exist
	Devices() ([]Device, error)
}

// PlaybackState holds the current state of a speaker or music session.
type PlaybackState struct {
	State    string // "playing", "paused", "stopped"
	Track    string
	Artist   string
	Album    string
	Duration string
	Position string
	Volume   int
	Muted    bool
}

// Speaker represents a controllable speaker or speaker group.
type Speaker interface {
	ID() string
	Name() string
	Room() string
	Provider() string // "sonos", "airplay"
	Model() string
	IsCoordinator() bool

	Play() error
	Pause() error
	Stop() error
	Next() error
	Previous() error
	GetVolume() (int, error)
	SetVolume(percent int) error
	GetMute() (bool, error)
	SetMute(muted bool) error
	PlayURI(uri string) error
	GetPlaybackState() (*PlaybackState, error)
}

// SpeakerProvider discovers and manages speakers from a single integration.
type SpeakerProvider interface {
	Name() string // provider identifier: "sonos"
	Discover() ([]Speaker, error)
}

// MusicService provides search and playback control for a music platform.
type MusicService interface {
	Name() string // "spotify", "apple-music"
	IsConfigured() bool
	IsLoggedIn() bool

	Search(query, types string, limit int) (*SearchResults, error)
}

// SearchResults holds search results from a music service.
type SearchResults struct {
	Tracks    []SearchTrack
	Playlists []SearchPlaylist
	Albums    []SearchAlbum
}

type SearchTrack struct {
	Name    string
	URI     string
	Artists string
	Album   string
}

type SearchPlaylist struct {
	Name  string
	URI   string
	Owner string
}

type SearchAlbum struct {
	Name    string
	URI     string
	Artists string
}
