package tradfri

import "fmt"

type DeviceType int

const (
	DeviceTypeUnknown DeviceType = iota
	DeviceTypeLight
	DeviceTypePlug
	DeviceTypeRemote
	DeviceTypeSensor
)

type Device struct {
	ID         int
	Name       string
	Type       DeviceType
	On         bool
	Reachable  bool
	Brightness int // 0-100 percent
	Dimmable   bool
	Model      string
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
