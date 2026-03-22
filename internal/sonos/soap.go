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
		return nil, fmt.Errorf("SOAP error %d from %s: %s", resp.StatusCode, speakerIP, string(respBody))
	}

	return respBody, nil
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
