package device

import "sync"

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

// AllDevices returns devices from all configured providers.
func (r *Registry) AllDevices() ([]Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var all []Device
	for _, p := range r.deviceProviders {
		if !p.IsConfigured() {
			continue
		}
		devices, err := p.Devices()
		if err != nil {
			continue // skip failing providers
		}
		all = append(all, devices...)
	}
	return all, nil
}

// AllSpeakers returns speakers from all providers.
func (r *Registry) AllSpeakers() ([]Speaker, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var all []Speaker
	for _, p := range r.speakerProviders {
		speakers, err := p.Discover()
		if err != nil {
			continue
		}
		all = append(all, speakers...)
	}
	return all, nil
}
