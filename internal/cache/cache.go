// Package cache provides a local name-to-ID/IP cache for fast device and speaker lookups.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type DeviceEntry struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type SpeakerEntry struct {
	IP            string `json:"ip"`
	UUID          string `json:"uuid"`
	Room          string `json:"room"`
	Model         string `json:"model"`
	IsCoordinator bool   `json:"is_coordinator"`
	CoordinatorIP string `json:"coordinator_ip,omitempty"`
}

type Cache struct {
	Devices  []DeviceEntry  `json:"devices"`
	Speakers []SpeakerEntry `json:"speakers"`
}

func cachePath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "crib", "cache.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "crib", "cache.json")
}

// Load reads the cache from disk. Returns an error if the file is missing or corrupt.
func Load() (*Cache, error) {
	path := cachePath()
	if path == "" {
		return nil, fmt.Errorf("cannot determine cache path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Cache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// Save writes the cache to disk atomically.
func Save(c *Cache) error {
	path := cachePath()
	if path == "" {
		return fmt.Errorf("cannot determine cache path")
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Dir(path), 0700)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// FindDevice searches for a device by name (case-insensitive exact match, then substring).
func (c *Cache) FindDevice(name string) *DeviceEntry {
	lower := strings.ToLower(name)
	for i := range c.Devices {
		if strings.ToLower(c.Devices[i].Name) == lower {
			return &c.Devices[i]
		}
	}
	for i := range c.Devices {
		if strings.Contains(strings.ToLower(c.Devices[i].Name), lower) {
			return &c.Devices[i]
		}
	}
	return nil
}

// FindSpeaker searches for a speaker by room name (case-insensitive exact match, then substring).
func (c *Cache) FindSpeaker(name string) *SpeakerEntry {
	lower := strings.ToLower(name)
	for i := range c.Speakers {
		if strings.ToLower(c.Speakers[i].Room) == lower {
			return &c.Speakers[i]
		}
	}
	for i := range c.Speakers {
		if strings.Contains(strings.ToLower(c.Speakers[i].Room), lower) {
			return &c.Speakers[i]
		}
	}
	return nil
}
