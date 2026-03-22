package sonos

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 5 * time.Second}

type soapEnvelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    soapBody `xml:"Body"`
}

type soapBody struct {
	Content []byte `xml:",innerxml"`
}

func soapRequest(speakerIP, endpoint, serviceURN, action string, params string) ([]byte, error) {
	url := fmt.Sprintf("http://%s:1400%s", speakerIP, endpoint)

	body := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"
            s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
  <s:Body>
    <u:%s xmlns:u="%s">
      %s
    </u:%s>
  </s:Body>
</s:Envelope>`, action, serviceURN, params, action)

	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", `text/xml; charset="utf-8"`)
	req.Header.Set("SOAPAction", fmt.Sprintf(`"%s#%s"`, serviceURN, action))

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", speakerIP, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, parseSoapError(speakerIP, resp.StatusCode, respBody)
	}

	return respBody, nil
}

// Sonos UPnP error codes
var sonosErrorMessages = map[string]string{
	"701": "command not available in current state (e.g. pausing when already stopped)",
	"711": "illegal seek target",
	"712": "illegal MIME type",
	"714": "resource not found",
	"715": "resource not found",
	"716": "resource not found",
	"718": "invalid instance ID",
	"737": "no DNS server",
	"738": "bad domain name",
	"739": "server error",
	"501": "action failed (may need to rediscover speakers with 'crib speakers list')",
}

func parseSoapError(speakerIP string, statusCode int, body []byte) error {
	errorCode := extractTag(body, "errorCode")
	if errorCode != "" {
		if msg, ok := sonosErrorMessages[errorCode]; ok {
			return fmt.Errorf("%s (error %s): %s", speakerIP, errorCode, msg)
		}
		return fmt.Errorf("%s: UPnP error %s", speakerIP, errorCode)
	}
	return fmt.Errorf("SOAP error %d from %s", statusCode, speakerIP)
}

func extractTag(xmlData []byte, tagName string) string {
	s := string(xmlData)
	start := strings.Index(s, "<"+tagName+">")
	if start == -1 {
		// Try with namespace prefix
		start = strings.Index(s, ":"+tagName+">")
		if start == -1 {
			return ""
		}
		// Walk back to find the <
		for start > 0 && s[start] != '<' {
			start--
		}
	}
	startTag := strings.Index(s[start:], ">") + start + 1
	end := strings.Index(s[startTag:], "</")
	if end == -1 {
		return ""
	}
	return s[startTag : startTag+end]
}
