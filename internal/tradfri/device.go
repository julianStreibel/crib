package tradfri

import (
	"fmt"
	"math"
)

type DeviceType int

const (
	DeviceTypeUnknown DeviceType = iota
	DeviceTypeLight
	DeviceTypePlug
	DeviceTypeRemote
	DeviceTypeSensor
)

// colorTempPreset maps a hex color string (attr 5706) to a Kelvin value.
type colorTempPreset struct {
	hex    string
	kelvin int
}

// colorTempPresets defines the 4 TRÅDFRI white spectrum presets.
var colorTempPresets = []colorTempPreset{
	{"efd275", 2200}, // warm glow
	{"f1e0b5", 2700}, // warm white
	{"f2eccf", 3000}, // sunrise
	{"f5faf6", 4000}, // cool white
}

// hexToKelvin maps hex color strings to Kelvin values.
var hexToKelvin = func() map[string]int {
	m := make(map[string]int, len(colorTempPresets))
	for _, p := range colorTempPresets {
		m[p.hex] = p.kelvin
	}
	return m
}()

// nearestPreset returns the hex preset closest to the requested Kelvin.
func nearestPreset(kelvin int) colorTempPreset {
	best := colorTempPresets[0]
	bestDist := math.MaxInt
	for _, p := range colorTempPresets {
		dist := kelvin - p.kelvin
		if dist < 0 {
			dist = -dist
		}
		if dist < bestDist {
			bestDist = dist
			best = p
		}
	}
	return best
}

type Device struct {
	ID            int
	Name          string
	Type          DeviceType
	On            bool
	Reachable     bool
	Brightness    int // 0-100 percent
	Dimmable      bool
	ColorTemp     bool   // true if the device supports color temperature
	ColorTempK    int    // current color temperature in Kelvin
	ColorTempMinK int    // minimum supported Kelvin
	ColorTempMaxK int    // maximum supported Kelvin
	ColorHex      string // current hex color (attr 5706), used internally
	Model         string
}

func (d *Device) TypeString() string {
	switch d.Type {
	case DeviceTypeLight:
		return "light"
	case DeviceTypePlug:
		return "plug"
	case DeviceTypeRemote:
		return "remote"
	case DeviceTypeSensor:
		return "sensor"
	default:
		return "unknown"
	}
}

func (d *Device) StateString() string {
	if !d.Reachable {
		return "unreachable"
	}
	if !d.On {
		return "off"
	}
	if d.Dimmable && d.Type == DeviceTypeLight {
		if d.ColorTemp {
			return fmt.Sprintf("on (%d%%, %dK)", d.Brightness, d.ColorTempK)
		}
		return fmt.Sprintf("on (%d%%)", d.Brightness)
	}
	return "on"
}

func (d *Device) controlObject() string {
	switch d.Type {
	case DeviceTypePlug:
		return "3312"
	default:
		return "3311"
	}
}

func parseDevice(raw map[string]interface{}) *Device {
	dev := &Device{}

	if v, ok := raw["9003"].(float64); ok {
		dev.ID = int(v)
	}
	if v, ok := raw["9001"].(string); ok {
		dev.Name = v
	}
	if v, ok := raw["9019"].(float64); ok {
		dev.Reachable = v == 1
	}

	// Determine type from application type (5750)
	if v, ok := raw["5750"].(float64); ok {
		switch int(v) {
		case 2:
			dev.Type = DeviceTypeLight
		case 3:
			dev.Type = DeviceTypePlug
		case 0:
			dev.Type = DeviceTypeRemote
		default:
			dev.Type = DeviceTypeUnknown
		}
	}

	// Get model info
	if info, ok := raw["3"].(map[string]interface{}); ok {
		if v, ok := info["1"].(string); ok {
			dev.Model = v
		}
	}

	// Parse light control (3311)
	if lights, ok := raw["3311"].([]interface{}); ok && len(lights) > 0 {
		if light, ok := lights[0].(map[string]interface{}); ok {
			if v, ok := light["5850"].(float64); ok {
				dev.On = v == 1
			}
			if v, ok := light["5851"].(float64); ok {
				dev.Brightness = int(v / 254.0 * 100.0)
				dev.Dimmable = true
			}
			if v, ok := light["5706"].(string); ok {
				dev.ColorHex = v
				dev.ColorTemp = true
				dev.ColorTempMinK = colorTempPresets[0].kelvin
				dev.ColorTempMaxK = colorTempPresets[len(colorTempPresets)-1].kelvin
				if k, found := hexToKelvin[v]; found {
					dev.ColorTempK = k
				} else if mireds, ok := light["5711"].(float64); ok && mireds > 0 {
					dev.ColorTempK = 1000000 / int(mireds)
				}
			}
		}
	}

	// Parse plug control (3312)
	if plugs, ok := raw["3312"].([]interface{}); ok && len(plugs) > 0 {
		if plug, ok := plugs[0].(map[string]interface{}); ok {
			if v, ok := plug["5850"].(float64); ok {
				dev.On = v == 1
			}
		}
	}

	return dev
}
