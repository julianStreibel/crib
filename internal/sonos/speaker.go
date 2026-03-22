package sonos

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"
)

const (
	avTransportNS     = "urn:schemas-upnp-org:service:AVTransport:1"
	avTransportCtrl   = "/MediaRenderer/AVTransport/Control"
	renderControlNS   = "urn:schemas-upnp-org:service:RenderingControl:1"
	renderControlCtrl = "/MediaRenderer/RenderingControl/Control"
)

type Speaker struct {
	IP            string
	UUID          string
	Room          string
	Model         string
	IsCoordinator bool
	CoordinatorIP string // empty if this speaker IS the coordinator
}

type PlaybackState struct {
	State    string // PLAYING, PAUSED_PLAYBACK, STOPPED, TRANSITIONING
	Track    string
	Artist   string
	Album    string
	Duration string
	Position string
	Volume   int
	Muted    bool
}

// controlIP returns the IP to send playback commands to (coordinator for grouped speakers).
func (s *Speaker) controlIP() string {
	if s.CoordinatorIP != "" {
		return s.CoordinatorIP
	}
	return s.IP
}

func (s *Speaker) Play() error {
	_, err := soapRequest(s.controlIP(), avTransportCtrl, avTransportNS, "Play",
		"<InstanceID>0</InstanceID><Speed>1</Speed>")
	return err
}

func (s *Speaker) Pause() error {
	_, err := soapRequest(s.controlIP(), avTransportCtrl, avTransportNS, "Pause",
		"<InstanceID>0</InstanceID>")
	return err
}

// SetPlayMode sets the play mode on the speaker.
// Modes: "NORMAL", "REPEAT_ALL", "REPEAT_ONE", "SHUFFLE_NOREPEAT", "SHUFFLE"
func (s *Speaker) SetPlayMode(mode string) error {
	_, err := soapRequest(s.controlIP(), avTransportCtrl, avTransportNS, "SetPlayMode",
		fmt.Sprintf("<InstanceID>0</InstanceID><NewPlayMode>%s</NewPlayMode>", mode))
	return err
}

// GetPlayMode returns the current play mode.
func (s *Speaker) GetPlayMode() (string, error) {
	resp, err := soapRequest(s.controlIP(), avTransportCtrl, avTransportNS, "GetTransportSettings",
		"<InstanceID>0</InstanceID>")
	if err != nil {
		return "", err
	}
	return extractTag(resp, "PlayMode"), nil
}

// Ungroup removes this speaker from its current group, making it standalone.
func (s *Speaker) Ungroup() error {
	_, err := soapRequest(s.IP, avTransportCtrl, avTransportNS, "BecomeCoordinatorOfStandaloneGroup",
		"<InstanceID>0</InstanceID>")
	return err
}

// JoinGroup adds this speaker to another speaker's group.
func (s *Speaker) JoinGroup(coordinatorUUID string) error {
	uri := fmt.Sprintf("x-rincon:%s", coordinatorUUID)
	_, err := soapRequest(s.IP, avTransportCtrl, avTransportNS, "SetAVTransportURI",
		fmt.Sprintf("<InstanceID>0</InstanceID><CurrentURI>%s</CurrentURI><CurrentURIMetaData></CurrentURIMetaData>", uri))
	return err
}

func (s *Speaker) Stop() error {
	_, err := soapRequest(s.controlIP(), avTransportCtrl, avTransportNS, "Stop",
		"<InstanceID>0</InstanceID>")
	return err
}

func (s *Speaker) Next() error {
	_, err := soapRequest(s.controlIP(), avTransportCtrl, avTransportNS, "Next",
		"<InstanceID>0</InstanceID>")
	return err
}

func (s *Speaker) Previous() error {
	_, err := soapRequest(s.controlIP(), avTransportCtrl, avTransportNS, "Previous",
		"<InstanceID>0</InstanceID>")
	return err
}

func (s *Speaker) GetVolume() (int, error) {
	resp, err := soapRequest(s.IP, renderControlCtrl, renderControlNS, "GetVolume",
		"<InstanceID>0</InstanceID><Channel>Master</Channel>")
	if err != nil {
		return 0, err
	}
	volStr := extractTag(resp, "CurrentVolume")
	vol, _ := strconv.Atoi(volStr)
	return vol, nil
}

func (s *Speaker) SetVolume(vol int) error {
	if vol < 0 {
		vol = 0
	}
	if vol > 100 {
		vol = 100
	}
	_, err := soapRequest(s.IP, renderControlCtrl, renderControlNS, "SetVolume",
		fmt.Sprintf("<InstanceID>0</InstanceID><Channel>Master</Channel><DesiredVolume>%d</DesiredVolume>", vol))
	return err
}

func (s *Speaker) GetMute() (bool, error) {
	resp, err := soapRequest(s.IP, renderControlCtrl, renderControlNS, "GetMute",
		"<InstanceID>0</InstanceID><Channel>Master</Channel>")
	if err != nil {
		return false, err
	}
	return extractTag(resp, "CurrentMute") == "1", nil
}

func (s *Speaker) SetMute(mute bool) error {
	val := "0"
	if mute {
		val = "1"
	}
	_, err := soapRequest(s.IP, renderControlCtrl, renderControlNS, "SetMute",
		fmt.Sprintf("<InstanceID>0</InstanceID><Channel>Master</Channel><DesiredMute>%s</DesiredMute>", val))
	return err
}

func (s *Speaker) GetPlaybackState() (*PlaybackState, error) {
	state := &PlaybackState{}
	g := new(errgroup.Group)

	g.Go(func() error {
		resp, err := soapRequest(s.controlIP(), avTransportCtrl, avTransportNS, "GetTransportInfo",
			"<InstanceID>0</InstanceID>")
		if err != nil {
			return err
		}
		state.State = extractTag(resp, "CurrentTransportState")
		return nil
	})

	g.Go(func() error {
		resp, err := soapRequest(s.controlIP(), avTransportCtrl, avTransportNS, "GetPositionInfo",
			"<InstanceID>0</InstanceID>")
		if err != nil {
			return nil // non-fatal
		}
		state.Duration = extractTag(resp, "TrackDuration")
		state.Position = extractTag(resp, "RelTime")
		metadata := extractTag(resp, "TrackMetaData")
		metadata = html.UnescapeString(metadata)
		if metadata != "" && metadata != "NOT_IMPLEMENTED" {
			state.Track = extractDIDLTag(metadata, "dc:title")
			state.Artist = extractDIDLTag(metadata, "dc:creator")
			state.Album = extractDIDLTag(metadata, "upnp:album")
		}
		return nil
	})

	g.Go(func() error {
		vol, err := s.GetVolume()
		if err == nil {
			state.Volume = vol
		}
		return nil
	})

	g.Go(func() error {
		muted, err := s.GetMute()
		if err == nil {
			state.Muted = muted
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return state, nil
}

// PlaySpotifyURI plays a Spotify URI (track, album, or playlist) on this speaker.
func (s *Speaker) PlaySpotifyURI(spotifyURI string) error {
	// Sonos uses a specific URI format for Spotify content
	// The URI needs to be URL-encoded and wrapped in the Sonos Spotify prefix
	encodedURI := strings.ReplaceAll(spotifyURI, ":", "%3a")

	var sonosURI, metadata string

	if strings.Contains(spotifyURI, ":track:") {
		sonosURI = fmt.Sprintf("x-sonos-spotify:%s?sid=12&amp;flags=8224&amp;sn=5", encodedURI)
		metadata = fmt.Sprintf(`<DIDL-Lite xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/" xmlns:r="urn:schemas-rinconnetworks-com:metadata-1-0/" xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/"><item id="00032020%s" restricted="true"><upnp:class>object.item.audioItem.musicTrack</upnp:class><desc id="cdudn" nameSpace="urn:schemas-rinconnetworks-com:metadata-1-0/">SA_RINCON2311_X_#Svc2311-0-Token</desc></item></DIDL-Lite>`, encodedURI)
	} else if strings.Contains(spotifyURI, ":album:") {
		sonosURI = fmt.Sprintf("x-rincon-cpcontainer:1004206c%s?sid=12&amp;flags=8300&amp;sn=5", encodedURI)
		metadata = fmt.Sprintf(`<DIDL-Lite xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/" xmlns:r="urn:schemas-rinconnetworks-com:metadata-1-0/" xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/"><item id="1004206c%s" restricted="true"><upnp:class>object.container.album.musicAlbum</upnp:class><desc id="cdudn" nameSpace="urn:schemas-rinconnetworks-com:metadata-1-0/">SA_RINCON2311_X_#Svc2311-0-Token</desc></item></DIDL-Lite>`, encodedURI)
	} else if strings.Contains(spotifyURI, ":playlist:") {
		sonosURI = fmt.Sprintf("x-rincon-cpcontainer:1006206c%s?sid=12&amp;flags=8300&amp;sn=5", encodedURI)
		metadata = fmt.Sprintf(`<DIDL-Lite xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/" xmlns:r="urn:schemas-rinconnetworks-com:metadata-1-0/" xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/"><item id="1006206c%s" restricted="true"><upnp:class>object.container.playlistContainer</upnp:class><desc id="cdudn" nameSpace="urn:schemas-rinconnetworks-com:metadata-1-0/">SA_RINCON2311_X_#Svc2311-0-Token</desc></item></DIDL-Lite>`, encodedURI)
	} else {
		return fmt.Errorf("unsupported Spotify URI format: %s", spotifyURI)
	}

	// HTML-escape the metadata for SOAP
	escapedMetadata := strings.ReplaceAll(metadata, "<", "&lt;")
	escapedMetadata = strings.ReplaceAll(escapedMetadata, ">", "&gt;")
	escapedMetadata = strings.ReplaceAll(escapedMetadata, `"`, "&quot;")

	params := fmt.Sprintf(
		"<InstanceID>0</InstanceID><CurrentURI>%s</CurrentURI><CurrentURIMetaData>%s</CurrentURIMetaData>",
		sonosURI, escapedMetadata)

	_, err := soapRequest(s.controlIP(), avTransportCtrl, avTransportNS, "SetAVTransportURI", params)
	if err != nil {
		return fmt.Errorf("setting transport URI: %w", err)
	}

	// Start playback
	return s.Play()
}

func (ps *PlaybackState) StateString() string {
	switch ps.State {
	case "PLAYING":
		return "playing"
	case "PAUSED_PLAYBACK":
		return "paused"
	case "STOPPED":
		return "stopped"
	case "TRANSITIONING":
		return "transitioning"
	default:
		return ps.State
	}
}

func (ps *PlaybackState) TrackString() string {
	parts := []string{}
	if ps.Artist != "" {
		parts = append(parts, ps.Artist)
	}
	if ps.Track != "" {
		parts = append(parts, ps.Track)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " - ")
}

func extractDIDLTag(didl, tagName string) string {
	start := strings.Index(didl, "<"+tagName)
	if start == -1 {
		return ""
	}
	closeTag := strings.Index(didl[start:], ">")
	if closeTag == -1 {
		return ""
	}
	contentStart := start + closeTag + 1
	end := strings.Index(didl[contentStart:], "</"+tagName)
	if end == -1 {
		return ""
	}
	return html.UnescapeString(didl[contentStart : contentStart+end])
}
