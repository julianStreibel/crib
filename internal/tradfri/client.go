package tradfri

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	piondtls "github.com/pion/dtls/v3"
	"golang.org/x/sync/errgroup"
	coapdtls "github.com/plgd-dev/go-coap/v3/dtls"
	"github.com/plgd-dev/go-coap/v3/message"
	udpClient "github.com/plgd-dev/go-coap/v3/udp/client"
)

const defaultTimeout = 5 * time.Second

type Client struct {
	host     string
	identity string
	psk      string

	mu   sync.Mutex
	conn *udpClient.Conn
}

func NewClient(host, identity, psk string) *Client {
	return &Client{host: host, identity: identity, psk: psk}
}

// connect returns a cached DTLS connection, dialing on first call.
func (c *Client) connect() (*udpClient.Conn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn, nil
	}
	conn, err := c.dial()
	if err != nil {
		return nil, err
	}
	c.conn = conn
	return c.conn, nil
}

// Close closes the cached DTLS connection if one exists.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func (c *Client) dial() (*udpClient.Conn, error) {
	addr := c.host + ":5684"
	return coapdtls.Dial(addr, &piondtls.Config{ //nolint:staticcheck // go-coap requires Config, new API not yet supported
		PSK: func(hint []byte) ([]byte, error) {
			return []byte(c.psk), nil
		},
		PSKIdentityHint: []byte(c.identity),
		CipherSuites:    []piondtls.CipherSuiteID{piondtls.TLS_PSK_WITH_AES_128_CCM_8},
	})
}

func (c *Client) get(path string) ([]byte, error) {
	conn, err := c.connect()
	if err != nil {
		return nil, fmt.Errorf("connecting to gateway: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	resp, err := conn.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", path, err)
	}
	if resp == nil || resp.Body() == nil {
		return nil, fmt.Errorf("GET %s: empty response", path)
	}

	return io.ReadAll(resp.Body())
}

func (c *Client) put(path string, payload string) error {
	conn, err := c.connect()
	if err != nil {
		return fmt.Errorf("connecting to gateway: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	_, err = conn.Put(ctx, path, message.AppJSON, strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("PUT %s: %w", path, err)
	}
	return nil
}

// Register performs the one-time PSK registration with the gateway.
func Register(host, securityCode, clientName string) (identity, psk string, err error) {
	addr := host + ":5684"
	conn, err := coapdtls.Dial(addr, &piondtls.Config{ //nolint:staticcheck // go-coap requires Config
		PSK: func(hint []byte) ([]byte, error) {
			return []byte(securityCode), nil
		},
		PSKIdentityHint: []byte("Client_identity"),
		CipherSuites:    []piondtls.CipherSuiteID{piondtls.TLS_PSK_WITH_AES_128_CCM_8},
	})
	if err != nil {
		return "", "", fmt.Errorf("connecting to gateway: %w", err)
	}
	defer conn.Close()

	payload := fmt.Sprintf(`{"9090":"%s"}`, clientName)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	resp, err := conn.Post(ctx, "/15011/9063", message.AppJSON, strings.NewReader(payload))
	if err != nil {
		return "", "", fmt.Errorf("registering: %w", err)
	}
	if resp == nil || resp.Body() == nil {
		return "", "", fmt.Errorf("gateway returned empty response — check your security code")
	}

	body, err := io.ReadAll(resp.Body())
	if err != nil {
		return "", "", fmt.Errorf("reading response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("parsing response: %w (body=%s)", err, body)
	}

	pskVal, ok := result["9091"]
	if !ok {
		return "", "", fmt.Errorf("no PSK in response: %s", body)
	}

	return clientName, fmt.Sprintf("%v", pskVal), nil
}

// ListDeviceIDs returns all device IDs from the gateway.
func (c *Client) ListDeviceIDs() ([]int, error) {
	body, err := c.get("/15001")
	if err != nil {
		return nil, err
	}

	var ids []int
	if err := json.Unmarshal(body, &ids); err != nil {
		return nil, fmt.Errorf("parsing device list: %w", err)
	}
	return ids, nil
}

// GetDevice returns the parsed device for a given device ID.
func (c *Client) GetDevice(id int) (*Device, error) {
	body, err := c.get(fmt.Sprintf("/15001/%d", id))
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing device: %w", err)
	}
	return parseDevice(raw), nil
}

// GetAllDevices returns all devices from the gateway, fetching them concurrently.
func (c *Client) GetAllDevices() ([]*Device, error) {
	ids, err := c.ListDeviceIDs()
	if err != nil {
		return nil, err
	}

	devices := make([]*Device, len(ids))
	g := new(errgroup.Group)
	for i, id := range ids {
		g.Go(func() error {
			dev, err := c.GetDevice(id)
			if err != nil {
				return nil // skip failures
			}
			devices[i] = dev
			return nil
		})
	}
	_ = g.Wait()

	// Filter out nils (failed fetches).
	result := make([]*Device, 0, len(devices))
	for _, d := range devices {
		if d != nil {
			result = append(result, d)
		}
	}
	return result, nil
}

// TurnOn turns on a light or plug.
func (c *Client) TurnOn(dev *Device) error {
	obj := dev.controlObject()
	payload := fmt.Sprintf(`{"%s":[{"5850":1}]}`, obj)
	return c.put(fmt.Sprintf("/15001/%d", dev.ID), payload)
}

// TurnOff turns off a light or plug.
func (c *Client) TurnOff(dev *Device) error {
	obj := dev.controlObject()
	payload := fmt.Sprintf(`{"%s":[{"5850":0}]}`, obj)
	return c.put(fmt.Sprintf("/15001/%d", dev.ID), payload)
}

// Toggle toggles a device on/off.
func (c *Client) Toggle(dev *Device) error {
	if dev.On {
		return c.TurnOff(dev)
	}
	return c.TurnOn(dev)
}

// SetBrightness sets the brightness of a light (0-100 percent).
func (c *Client) SetBrightness(dev *Device, percent int) error {
	if percent <= 0 {
		return c.TurnOff(dev)
	}
	if percent > 100 {
		percent = 100
	}
	brightness := int(float64(percent) / 100.0 * 254.0)
	if brightness < 1 {
		brightness = 1
	}
	payload := fmt.Sprintf(`{"3311":[{"5850":1,"5851":%d}]}`, brightness)
	return c.put(fmt.Sprintf("/15001/%d", dev.ID), payload)
}

// SetColorTemp sets the color temperature of a light in Kelvin.
// The value is snapped to the nearest supported TRÅDFRI preset.
func (c *Client) SetColorTemp(dev *Device, kelvin int) error {
	preset := nearestPreset(kelvin)
	payload := fmt.Sprintf(`{"3311":[{"5706":"%s","5850":1}]}`, preset.hex)
	return c.put(fmt.Sprintf("/15001/%d", dev.ID), payload)
}

// CheckConnection verifies the gateway is reachable.
func (c *Client) CheckConnection() error {
	_, err := c.ListDeviceIDs()
	return err
}
