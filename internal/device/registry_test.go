package device

import "testing"

// Mock implementations for testing

type mockDeviceProvider struct {
	name       string
	configured bool
	devices    []Device
}

func (m *mockDeviceProvider) Name() string              { return m.name }
func (m *mockDeviceProvider) IsConfigured() bool        { return m.configured }
func (m *mockDeviceProvider) Devices() ([]Device, error) { return m.devices, nil }

type mockDevice struct {
	id       string
	name     string
	dtype    DeviceType
	provider string
}

func (m *mockDevice) ID() string           { return m.id }
func (m *mockDevice) Name() string         { return m.name }
func (m *mockDevice) Type() DeviceType     { return m.dtype }
func (m *mockDevice) Provider() string     { return m.provider }
func (m *mockDevice) Model() string        { return "mock" }
func (m *mockDevice) State() DeviceState   { return DeviceState{On: true, Reachable: true} }
func (m *mockDevice) TurnOn() error        { return nil }
func (m *mockDevice) TurnOff() error       { return nil }
func (m *mockDevice) Toggle() error        { return nil }
func (m *mockDevice) SetBrightness(int) error { return nil }

type mockSpeakerProvider struct {
	name     string
	speakers []Speaker
}

func (m *mockSpeakerProvider) Name() string                  { return m.name }
func (m *mockSpeakerProvider) Discover() ([]Speaker, error)  { return m.speakers, nil }

func TestRegistryDeviceProviders(t *testing.T) {
	r := NewRegistry()

	d1 := &mockDevice{id: "1", name: "Kitchen", dtype: TypeLight, provider: "hue"}
	d2 := &mockDevice{id: "2", name: "Office", dtype: TypeLight, provider: "tradfri"}

	r.RegisterDeviceProvider(&mockDeviceProvider{
		name:       "hue",
		configured: true,
		devices:    []Device{d1},
	})
	r.RegisterDeviceProvider(&mockDeviceProvider{
		name:       "tradfri",
		configured: true,
		devices:    []Device{d2},
	})

	devices, err := r.AllDevices()
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(devices))
	}
}

func TestRegistrySkipsUnconfigured(t *testing.T) {
	r := NewRegistry()

	r.RegisterDeviceProvider(&mockDeviceProvider{
		name:       "hue",
		configured: false,
		devices:    []Device{&mockDevice{id: "1", name: "Kitchen"}},
	})

	devices, err := r.AllDevices()
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 0 {
		t.Errorf("expected 0 devices (unconfigured), got %d", len(devices))
	}
}

func TestDeviceTypeString(t *testing.T) {
	tests := []struct {
		t    DeviceType
		want string
	}{
		{TypeLight, "light"},
		{TypeSwitch, "switch"},
		{TypeSensor, "sensor"},
		{DeviceType(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.t.String(); got != tt.want {
			t.Errorf("DeviceType(%d).String() = %q, want %q", tt.t, got, tt.want)
		}
	}
}
