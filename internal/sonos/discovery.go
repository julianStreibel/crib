package sonos

import (
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	ssdpAddr    = "239.255.255.250:1900"
	ssdpSearch  = "M-SEARCH * HTTP/1.1\r\nHOST: 239.255.255.250:1900\r\nMAN: \"ssdp:discover\"\r\nMX: 2\r\nST: urn:schemas-upnp-org:device:ZonePlayer:1\r\n\r\n"
	zoneGroupNS = "urn:schemas-upnp-org:service:ZoneGroupTopology:1"
)

type deviceDescription struct {
	XMLName    xml.Name `xml:"root"`
	Device     ddDevice `xml:"device"`
}

type ddDevice struct {
	RoomName    string `xml:"roomName"`
	DisplayName string `xml:"displayName"`
	ModelName   string `xml:"modelName"`
	UDN         string `xml:"UDN"`
}

// Discover finds all Sonos speakers on the local network via SSDP.
func Discover(timeout time.Duration) ([]*Speaker, error) {
	ips, err := ssdpDiscover(timeout)
	if err != nil {
		return nil, err
	}

	if len(ips) == 0 {
		return nil, nil
	}

	// Get zone group topology from the first speaker
	speakers, err := getZoneGroupTopology(ips[0])
	if err != nil {
		// Fallback: build speaker list from SSDP results
		return buildFromSsdp(ips)
	}

	return speakers, nil
}

func ssdpDiscover(timeout time.Duration) ([]string, error) {
	addr, err := net.ResolveUDPAddr("udp4", ssdpAddr)
	if err != nil {
		return nil, err
	}

	// Bind to all interfaces
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(timeout))

	// Send M-SEARCH multiple times for reliability
	for i := 0; i < 3; i++ {
		_, _ = conn.WriteToUDP([]byte(ssdpSearch), addr)
		time.Sleep(100 * time.Millisecond)
	}

	seen := make(map[string]bool)
	var ips []string
	buf := make([]byte, 4096)

	for {
		n, raddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		ip := raddr.IP.String()
		if seen[ip] {
			continue
		}
		resp := string(buf[:n])
		if strings.Contains(resp, "ZonePlayer") || strings.Contains(resp, "Sonos") || strings.Contains(resp, "1400") {
			seen[ip] = true
			ips = append(ips, ip)
		}
	}

	// Fallback: if SSDP found nothing, try scanning known Sonos port
	if len(ips) == 0 {
		ips = scanForSonos()
	}

	return ips, nil
}

// scanForSonos tries to find Sonos speakers by checking port 1400 on the local subnet.
func scanForSonos() []string {
	localIP := getLocalIP()
	if localIP == "" {
		return nil
	}

	// Derive subnet (assumes /24)
	parts := strings.Split(localIP, ".")
	if len(parts) != 4 {
		return nil
	}
	prefix := strings.Join(parts[:3], ".")

	var ips []string
	results := make(chan string, 255)

	for i := 1; i <= 254; i++ {
		go func(ip string) {
			conn, err := net.DialTimeout("tcp", ip+":1400", 300*time.Millisecond)
			if err == nil {
				conn.Close()
				// Verify it's Sonos
				resp, err := http.Get(fmt.Sprintf("http://%s:1400/xml/device_description.xml", ip))
				if err == nil {
					defer resp.Body.Close()
					body, _ := io.ReadAll(resp.Body)
					if strings.Contains(string(body), "Sonos") {
						results <- ip
						return
					}
				}
			}
			results <- ""
		}(fmt.Sprintf("%s.%d", prefix, i))
	}

	for i := 0; i < 254; i++ {
		if ip := <-results; ip != "" {
			ips = append(ips, ip)
		}
	}

	return ips
}

func getLocalIP() string {
	conn, err := net.Dial("udp4", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

type zoneGroupState struct {
	XMLName xml.Name    `xml:"ZoneGroupState"`
	Groups  []zoneGroup `xml:"ZoneGroups>ZoneGroup"`
}

type zoneGroup struct {
	Coordinator string            `xml:"Coordinator,attr"`
	Members     []zoneGroupMember `xml:"ZoneGroupMember"`
}

type zoneGroupMember struct {
	UUID       string            `xml:"UUID,attr"`
	Location   string            `xml:"Location,attr"`
	ZoneName   string            `xml:"ZoneName,attr"`
	Invisible  string            `xml:"Invisible,attr"`
	Satellites []zoneGroupMember `xml:"Satellite"`
}

func getZoneGroupTopology(speakerIP string) ([]*Speaker, error) {
	resp, err := soapRequest(speakerIP,
		"/ZoneGroupTopology/Control",
		zoneGroupNS,
		"GetZoneGroupState",
		"")
	if err != nil {
		return nil, err
	}

	// Extract ZoneGroupState content (it's HTML-encoded XML inside the SOAP response)
	stateXML := extractTag(resp, "ZoneGroupState")
	stateXML = html.UnescapeString(stateXML)

	var state zoneGroupState
	if err := xml.Unmarshal([]byte(stateXML), &state); err != nil {
		return nil, fmt.Errorf("parsing zone groups: %w", err)
	}

	var speakers []*Speaker
	for _, group := range state.Groups {
		// Find coordinator IP
		var coordIP string
		for _, member := range group.Members {
			if member.UUID == group.Coordinator {
				coordIP = extractIPFromLocation(member.Location)
				break
			}
		}

		for _, member := range group.Members {
			ip := extractIPFromLocation(member.Location)
			if ip == "" {
				continue
			}
			// Skip invisible satellite speakers (surround, sub)
			if member.Invisible == "1" {
				continue
			}

			s := &Speaker{
				IP:            ip,
				UUID:          member.UUID,
				Room:          member.ZoneName,
				IsCoordinator: member.UUID == group.Coordinator,
			}
			if !s.IsCoordinator {
				s.CoordinatorIP = coordIP
			}
			speakers = append(speakers, s)
		}
	}

	// Fetch model info for each speaker
	for _, s := range speakers {
		desc, err := getDeviceDescription(s.IP)
		if err == nil {
			s.Model = desc.Device.DisplayName
			if s.Model == "" {
				s.Model = desc.Device.ModelName
			}
		}
	}

	return speakers, nil
}

func getDeviceDescription(ip string) (*deviceDescription, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:1400/xml/device_description.xml", ip))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var desc deviceDescription
	if err := xml.Unmarshal(body, &desc); err != nil {
		return nil, err
	}
	return &desc, nil
}

func extractIPFromLocation(location string) string {
	// Extract IP from "http://192.168.1.50:1400/xml/device_description.xml"
	location = strings.TrimPrefix(location, "http://")
	location = strings.TrimPrefix(location, "https://")
	parts := strings.Split(location, ":")
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

func buildFromSsdp(ips []string) ([]*Speaker, error) {
	var speakers []*Speaker
	for _, ip := range ips {
		desc, err := getDeviceDescription(ip)
		if err != nil {
			continue
		}
		model := desc.Device.DisplayName
		if model == "" {
			model = desc.Device.ModelName
		}
		speakers = append(speakers, &Speaker{
			IP:            ip,
			UUID:          desc.Device.UDN,
			Room:          desc.Device.RoomName,
			Model:         model,
			IsCoordinator: true,
		})
	}
	return speakers, nil
}
