package discovery

import (
	"time"

	"github.com/hashicorp/mdns"
)

type Device struct {
	Name string
	Host string
	Port int
}

// FindTradfriGateways scans the local network for TRÅDFRI gateways via mDNS.
func FindTradfriGateways(timeout time.Duration) ([]Device, error) {
	var devices []Device

	entriesCh := make(chan *mdns.ServiceEntry, 10)
	go func() {
		for entry := range entriesCh {
			host := entry.AddrV4.String()
			if host == "" && entry.AddrV6 != nil {
				host = entry.AddrV6.String()
			}
			devices = append(devices, Device{
				Name: entry.Name,
				Host: host,
				Port: entry.Port,
			})
		}
	}()

	params := &mdns.QueryParam{
		Service:             "_coap._udp",
		Domain:              "local",
		Timeout:             timeout,
		Entries:             entriesCh,
		WantUnicastResponse: false,
		DisableIPv6:         true,
	}

	// mDNS query may fail on some networks (e.g. no IPv6) — treat as non-fatal
	_ = mdns.Query(params)

	close(entriesCh)
	// Give goroutine time to process
	time.Sleep(100 * time.Millisecond)

	return devices, nil
}
