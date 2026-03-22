package spotify

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Client struct {
	clientID     string
	clientSecret string

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

func NewClient(clientID, clientSecret string) *Client {
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

type SearchResult struct {
	Tracks    []Track    `json:"tracks"`
	Playlists []Playlist `json:"playlists"`
	Albums    []Album    `json:"albums"`
}

type Track struct {
	Name    string `json:"name"`
	URI     string `json:"uri"`
	Artists string `json:"artists"`
	Album   string `json:"album"`
}

type Playlist struct {
	Name  string `json:"name"`
	URI   string `json:"uri"`
	Owner string `json:"owner"`
}

type Album struct {
	Name    string `json:"name"`
	URI     string `json:"uri"`
	Artists string `json:"artists"`
}

func (c *Client) authenticate() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return nil
	}

	data := url.Values{"grant_type": {"client_credentials"}}
	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("authenticating with Spotify: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Spotify auth failed (%d): %s", resp.StatusCode, body)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	c.accessToken = result.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn-60) * time.Second)
	return nil
}

func (c *Client) apiGet(endpoint string) ([]byte, error) {
	if err := c.authenticate(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", "https://api.spotify.com/v1"+endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Spotify API error (%d): %s", resp.StatusCode, body)
	}
	return body, nil
}

// Search searches Spotify for tracks, playlists, and albums.
func (c *Client) Search(query string, types string, limit int) (*SearchResult, error) {
	if types == "" {
		types = "track,playlist,album"
	}
	if limit == 0 {
		limit = 5
	}

	endpoint := fmt.Sprintf("/search?q=%s&type=%s&limit=%d",
		url.QueryEscape(query), url.QueryEscape(types), limit)

	body, err := c.apiGet(endpoint)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Tracks struct {
			Items []struct {
				Name    string `json:"name"`
				URI     string `json:"uri"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
				Album struct {
					Name string `json:"name"`
				} `json:"album"`
			} `json:"items"`
		} `json:"tracks"`
		Playlists struct {
			Items []struct {
				Name  string `json:"name"`
				URI   string `json:"uri"`
				Owner struct {
					DisplayName string `json:"display_name"`
				} `json:"owner"`
			} `json:"items"`
		} `json:"playlists"`
		Albums struct {
			Items []struct {
				Name    string `json:"name"`
				URI     string `json:"uri"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
			} `json:"items"`
		} `json:"albums"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing search results: %w", err)
	}

	result := &SearchResult{}

	for _, t := range raw.Tracks.Items {
		artists := make([]string, len(t.Artists))
		for i, a := range t.Artists {
			artists[i] = a.Name
		}
		result.Tracks = append(result.Tracks, Track{
			Name:    t.Name,
			URI:     t.URI,
			Artists: strings.Join(artists, ", "),
			Album:   t.Album.Name,
		})
	}

	for _, p := range raw.Playlists.Items {
		result.Playlists = append(result.Playlists, Playlist{
			Name:  p.Name,
			URI:   p.URI,
			Owner: p.Owner.DisplayName,
		})
	}

	for _, a := range raw.Albums.Items {
		artists := make([]string, len(a.Artists))
		for i, ar := range a.Artists {
			artists[i] = ar.Name
		}
		result.Albums = append(result.Albums, Album{
			Name:    a.Name,
			URI:     a.URI,
			Artists: strings.Join(artists, ", "),
		})
	}

	return result, nil
}
