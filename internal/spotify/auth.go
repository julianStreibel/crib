package spotify

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	authURL     = "https://accounts.spotify.com/authorize"
	tokenURL    = "https://accounts.spotify.com/api/token"
	redirectURI = "http://127.0.0.1:8089/callback"
	scopes      = "user-read-playback-state user-modify-playback-state user-read-currently-playing playlist-read-private playlist-modify-public playlist-modify-private"
)

type TokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	ExpiresAt    int64  `json:"expires_at"`
}

func (t *TokenData) IsExpired() bool {
	return time.Now().Unix() >= t.ExpiresAt
}

// AuthorizePKCE performs the full PKCE authorization flow:
// opens a browser, listens for callback, exchanges code for tokens.
func AuthorizePKCE(clientID string) (*TokenData, error) {
	verifier, challenge, err := generatePKCE()
	if err != nil {
		return nil, err
	}

	state, err := randomString(16)
	if err != nil {
		return nil, err
	}

	// Build authorization URL
	authParams := url.Values{
		"client_id":             {clientID},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI},
		"scope":                 {scopes},
		"code_challenge_method": {"S256"},
		"code_challenge":        {challenge},
		"state":                 {state},
	}

	authLink := authURL + "?" + authParams.Encode()

	// Try to start local callback server
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	serverRunning := false

	listener, listenErr := net.Listen("tcp", "127.0.0.1:8089")
	if listenErr == nil {
		serverRunning = true
		mux := http.NewServeMux()
		mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("state") != state {
				errCh <- fmt.Errorf("state mismatch")
				fmt.Fprint(w, "Error: state mismatch. Close this tab.")
				return
			}
			if errParam := r.URL.Query().Get("error"); errParam != "" {
				errCh <- fmt.Errorf("authorization denied: %s", errParam)
				fmt.Fprint(w, "Authorization denied. Close this tab.")
				return
			}
			code := r.URL.Query().Get("code")
			codeCh <- code
			fmt.Fprint(w, "Authorization successful! You can close this tab.")
		})

		srv := &http.Server{Handler: mux}
		go func() { _ = srv.Serve(listener) }()
		defer srv.Close()
	}

	fmt.Println("Visit this URL to authorize Spotify:")
	fmt.Printf("\n%s\n\n", authLink)

	if serverRunning {
		openBrowser(authLink)
		fmt.Println("Waiting for callback... (or paste the redirect URL below)")
	} else {
		fmt.Println("After authorizing, you'll be redirected to a URL starting with")
		fmt.Println("http://127.0.0.1:8089/callback?code=...")
		fmt.Println("Copy that full URL and paste it below.")
	}

	// Wait for either the callback server or manual paste
	manualCh := make(chan string, 1)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("\nPaste redirect URL (or wait for automatic callback): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			manualCh <- input
		}
	}()

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return nil, err
	case pastedURL := <-manualCh:
		// Parse code from pasted URL
		parsed, parseErr := url.Parse(pastedURL)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid URL: %w", parseErr)
		}
		code = parsed.Query().Get("code")
		if code == "" {
			return nil, fmt.Errorf("no authorization code found in URL")
		}
	case <-time.After(300 * time.Second):
		return nil, fmt.Errorf("authorization timed out (5 minutes)")
	}

	// Exchange code for tokens
	return exchangeCode(clientID, code, verifier)
}

// RefreshAccessToken refreshes an expired access token.
func RefreshAccessToken(clientID string, refreshToken string) (*TokenData, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	}

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("token refresh failed (%d): %s", resp.StatusCode, body)
	}

	var token TokenData
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	token.ExpiresAt = time.Now().Unix() + int64(token.ExpiresIn) - 60

	// Keep old refresh token if new one not provided
	if token.RefreshToken == "" {
		token.RefreshToken = refreshToken
	}

	return &token, nil
}

func exchangeCode(clientID, code, verifier string) (*TokenData, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {clientID},
		"code_verifier": {verifier},
	}

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("token exchange failed (%d): %s", resp.StatusCode, body)
	}

	var token TokenData
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	token.ExpiresAt = time.Now().Unix() + int64(token.ExpiresIn) - 60

	return &token, nil
}

func generatePKCE() (verifier, challenge string, err error) {
	verifier, err = randomString(64)
	if err != nil {
		return "", "", err
	}

	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(hash[:])
	return verifier, challenge, nil
}

func randomString(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b)[:length], nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		cmd = exec.Command("open", url)
	}
	_ = cmd.Start()
}

// PlayerClient is a Spotify client authenticated with user tokens for playback control.
type PlayerClient struct {
	clientID string
	token    *TokenData
	onSave   func(*TokenData) // called when token is refreshed
}

func NewPlayerClient(clientID string, token *TokenData, onSave func(*TokenData)) *PlayerClient {
	return &PlayerClient{clientID: clientID, token: token, onSave: onSave}
}

func (p *PlayerClient) ensureToken() error {
	if !p.token.IsExpired() {
		return nil
	}
	newToken, err := RefreshAccessToken(p.clientID, p.token.RefreshToken)
	if err != nil {
		return fmt.Errorf("refreshing token: %w", err)
	}
	p.token = newToken
	if p.onSave != nil {
		p.onSave(newToken)
	}
	return nil
}

func (p *PlayerClient) request(method, endpoint string, body io.Reader) ([]byte, int, error) {
	if err := p.ensureToken(); err != nil {
		return nil, 0, err
	}

	req, err := http.NewRequest(method, "https://api.spotify.com/v1"+endpoint, body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+p.token.AccessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 401 {
		// Token might have just expired, try refresh
		newToken, err := RefreshAccessToken(p.clientID, p.token.RefreshToken)
		if err != nil {
			return nil, resp.StatusCode, fmt.Errorf("token expired and refresh failed: %w", err)
		}
		p.token = newToken
		if p.onSave != nil {
			p.onSave(newToken)
		}
		// Retry
		return p.request(method, endpoint, body)
	}

	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, fmt.Errorf("spotify API error (%d): %s", resp.StatusCode, respBody)
	}

	return respBody, resp.StatusCode, nil
}

type PlayerState struct {
	IsPlaying bool   `json:"is_playing"`
	Device    Device `json:"device"`
	Item      *struct {
		Name    string `json:"name"`
		URI     string `json:"uri"`
		Artists []struct {
			Name string `json:"name"`
		} `json:"artists"`
		Album struct {
			Name string `json:"name"`
		} `json:"album"`
		DurationMs int `json:"duration_ms"`
	} `json:"item"`
	ProgressMs int `json:"progress_ms"`
}

type Device struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	IsActive       bool   `json:"is_active"`
	VolumePercent  int    `json:"volume_percent"`
	SupportsVolume bool   `json:"supports_volume"`
}

func (p *PlayerClient) GetDevices() ([]Device, error) {
	body, _, err := p.request("GET", "/me/player/devices", nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Devices []Device `json:"devices"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Devices, nil
}

func (p *PlayerClient) GetPlayerState() (*PlayerState, error) {
	body, status, err := p.request("GET", "/me/player", nil)
	if err != nil {
		return nil, err
	}
	if status == 204 {
		return nil, nil // nothing playing
	}
	var state PlayerState
	if err := json.Unmarshal(body, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (p *PlayerClient) Play(deviceID string) error {
	endpoint := "/me/player/play"
	if deviceID != "" {
		endpoint += "?device_id=" + deviceID
	}
	_, _, err := p.request("PUT", endpoint, nil)
	return err
}

func (p *PlayerClient) PlayURI(uri string, deviceID string) error {
	endpoint := "/me/player/play"
	if deviceID != "" {
		endpoint += "?device_id=" + deviceID
	}

	var payload string
	if strings.Contains(uri, ":track:") {
		payload = fmt.Sprintf(`{"uris":["%s"]}`, uri)
	} else {
		payload = fmt.Sprintf(`{"context_uri":"%s"}`, uri)
	}

	_, _, err := p.request("PUT", endpoint, strings.NewReader(payload))
	return err
}

func (p *PlayerClient) Pause() error {
	_, _, err := p.request("PUT", "/me/player/pause", nil)
	return err
}

func (p *PlayerClient) Next() error {
	_, _, err := p.request("POST", "/me/player/next", nil)
	return err
}

func (p *PlayerClient) Previous() error {
	_, _, err := p.request("POST", "/me/player/previous", nil)
	return err
}

func (p *PlayerClient) SetVolume(percent int) error {
	_, _, err := p.request("PUT", fmt.Sprintf("/me/player/volume?volume_percent=%d", percent), nil)
	return err
}

func (p *PlayerClient) TransferPlayback(deviceID string, play bool) error {
	payload := fmt.Sprintf(`{"device_ids":["%s"],"play":%t}`, deviceID, play)
	_, _, err := p.request("PUT", "/me/player", strings.NewReader(payload))
	return err
}

// SetRepeat sets repeat mode: "track", "context" (playlist/album), or "off".
func (p *PlayerClient) SetRepeat(state string) error {
	_, _, err := p.request("PUT", fmt.Sprintf("/me/player/repeat?state=%s", state), nil)
	return err
}

// SetShuffle sets shuffle mode.
func (p *PlayerClient) SetShuffle(on bool) error {
	_, _, err := p.request("PUT", fmt.Sprintf("/me/player/shuffle?state=%t", on), nil)
	return err
}

// AddToQueue adds a track to the playback queue.
func (p *PlayerClient) AddToQueue(uri string) error {
	_, _, err := p.request("POST", fmt.Sprintf("/me/player/queue?uri=%s", url.QueryEscape(uri)), nil)
	return err
}

// GetRecommendations returns recommended tracks based on a seed track.
func (p *PlayerClient) GetRecommendations(seedTrackID string, limit int) ([]string, error) {
	if limit == 0 {
		limit = 20
	}
	body, _, err := p.request("GET", fmt.Sprintf("/recommendations?seed_tracks=%s&limit=%d", seedTrackID, limit), nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Tracks []struct {
			URI string `json:"uri"`
		} `json:"tracks"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	uris := make([]string, len(result.Tracks))
	for i, t := range result.Tracks {
		uris[i] = t.URI
	}
	return uris, nil
}

// UserPlaylist represents a Spotify playlist with full details.
type UserPlaylist struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	URI   string `json:"uri"`
	Owner struct {
		DisplayName string `json:"display_name"`
	} `json:"owner"`
	Items struct {
		Total int `json:"total"`
	} `json:"items"`
}

// PlaylistItem represents an item inside a playlist.
type PlaylistItem struct {
	Item struct {
		Name    string `json:"name"`
		URI     string `json:"uri"`
		Artists []struct {
			Name string `json:"name"`
		} `json:"artists"`
	} `json:"item"`
}

// GetMyPlaylists returns the current user's playlists.
func (p *PlayerClient) GetMyPlaylists(limit int) ([]UserPlaylist, error) {
	if limit == 0 {
		limit = 50
	}
	body, _, err := p.request("GET", fmt.Sprintf("/me/playlists?limit=%d", limit), nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Items []UserPlaylist `json:"items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Items, nil
}

// GetPlaylistItems returns the items in a playlist.
func (p *PlayerClient) GetPlaylistItems(playlistID string, limit int) ([]PlaylistItem, error) {
	if limit == 0 {
		limit = 50
	}
	body, _, err := p.request("GET", fmt.Sprintf("/playlists/%s/items?limit=%d", playlistID, limit), nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Items []PlaylistItem `json:"items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Items, nil
}

// CreatePlaylist creates a new playlist for the current user.
func (p *PlayerClient) CreatePlaylist(name string, public bool) (*UserPlaylist, error) {
	payload := fmt.Sprintf(`{"name":%q,"public":%t}`, name, public)
	body, _, err := p.request("POST", "/me/playlists", strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	var pl UserPlaylist
	if err := json.Unmarshal(body, &pl); err != nil {
		return nil, err
	}
	return &pl, nil
}

// AddToPlaylist adds track URIs to a playlist.
func (p *PlayerClient) AddToPlaylist(playlistID string, uris []string) error {
	uriJSON, _ := json.Marshal(uris)
	payload := fmt.Sprintf(`{"uris":%s}`, string(uriJSON))
	_, _, err := p.request("POST", fmt.Sprintf("/playlists/%s/items", playlistID), strings.NewReader(payload))
	return err
}

// RemoveFromPlaylist removes track URIs from a playlist.
func (p *PlayerClient) RemoveFromPlaylist(playlistID string, uris []string) error {
	items := make([]map[string]string, len(uris))
	for i, uri := range uris {
		items[i] = map[string]string{"uri": uri}
	}
	itemsJSON, _ := json.Marshal(items)
	payload := fmt.Sprintf(`{"items":%s}`, string(itemsJSON))
	_, _, err := p.request("DELETE", fmt.Sprintf("/playlists/%s/items", playlistID), strings.NewReader(payload))
	return err
}

// PlayURIs plays a list of track URIs.
func (p *PlayerClient) PlayURIs(uris []string, deviceID string) error {
	endpoint := "/me/player/play"
	if deviceID != "" {
		endpoint += "?device_id=" + deviceID
	}

	uriJSON, _ := json.Marshal(uris)
	payload := fmt.Sprintf(`{"uris":%s}`, string(uriJSON))
	_, _, err := p.request("PUT", endpoint, strings.NewReader(payload))
	return err
}
