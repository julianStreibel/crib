package device

import (
	"sync"

	"golang.org/x/sync/errgroup"
)

// Registry holds all registered providers.
type Registry struct {
	mu               sync.RWMutex
	deviceProviders  []DeviceProvider
	speakerProviders []SpeakerProvider
	musicServices    []MusicService
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// RegisterDeviceProvider adds a device provider (e.g. TRÅDFRI, Hue).
func (r *Registry) RegisterDeviceProvider(p DeviceProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deviceProviders = append(r.deviceProviders, p)
}

// RegisterSpeakerProvider adds a speaker provider (e.g. Sonos).
func (r *Registry) RegisterSpeakerProvider(p SpeakerProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.speakerProviders = append(r.speakerProviders, p)
}

// RegisterMusicService adds a music service (e.g. Spotify).
func (r *Registry) RegisterMusicService(s MusicService) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.musicServices = append(r.musicServices, s)
}

// DeviceProviders returns all registered device providers.
func (r *Registry) DeviceProviders() []DeviceProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.deviceProviders
}

// SpeakerProviders returns all registered speaker providers.
func (r *Registry) SpeakerProviders() []SpeakerProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.speakerProviders
}

// MusicServices returns all registered music services.
func (r *Registry) MusicServices() []MusicService {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.musicServices
}

// AllDevices returns devices from all configured providers, querying them concurrently.
func (r *Registry) AllDevices() ([]Device, error) {
	r.mu.RLock()
	providers := make([]DeviceProvider, len(r.deviceProviders))
	copy(providers, r.deviceProviders)
	r.mu.RUnlock()

	results := make([][]Device, len(providers))
	g := new(errgroup.Group)
	for i, p := range providers {
		if !p.IsConfigured() {
			continue
		}
		g.Go(func() error {
			devices, err := p.Devices()
			if err != nil {
				return nil // skip failing providers
			}
			results[i] = devices
			return nil
		})
	}
	_ = g.Wait()

	var all []Device
	for _, r := range results {
		all = append(all, r...)
	}
	return all, nil
}

// AllSpeakers returns speakers from all providers, discovering them concurrently.
func (r *Registry) AllSpeakers() ([]Speaker, error) {
	r.mu.RLock()
	providers := make([]SpeakerProvider, len(r.speakerProviders))
	copy(providers, r.speakerProviders)
	r.mu.RUnlock()

	results := make([][]Speaker, len(providers))
	g := new(errgroup.Group)
	for i, p := range providers {
		g.Go(func() error {
			speakers, err := p.Discover()
			if err != nil {
				return nil
			}
			results[i] = speakers
			return nil
		})
	}
	_ = g.Wait()

	var all []Speaker
	for _, r := range results {
		all = append(all, r...)
	}
	return all, nil
}
