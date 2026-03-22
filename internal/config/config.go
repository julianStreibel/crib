package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	appDir     = "crib"
	configName = "config"
	configType = "yaml"
)

type Config struct {
	TradfriHost     string `mapstructure:"tradfri_host"`
	TradfriIdentity string `mapstructure:"tradfri_identity"`
	TradfriPSK      string `mapstructure:"tradfri_psk"`
}

func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, appDir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot find home directory: %v\n", err)
		os.Exit(1)
	}
	return filepath.Join(home, ".config", appDir)
}

func configPath() string {
	return filepath.Join(configDir(), configName+"."+configType)
}

func Init() {
	dir := configDir()
	_ = os.MkdirAll(dir, 0700)

	viper.SetConfigName(configName)
	viper.SetConfigType(configType)
	viper.AddConfigPath(dir)

	viper.SetEnvPrefix("CRIB")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	_ = viper.ReadInConfig()
}

func LoadSpotify() (clientID, clientSecret string, err error) {
	clientID = viper.GetString("spotify_client_id")
	clientSecret = viper.GetString("spotify_client_secret")
	if clientID == "" || clientSecret == "" {
		return "", "", fmt.Errorf("spotify credentials not configured, run: crib setup")
	}
	return clientID, clientSecret, nil
}

func LoadSpotifyToken() (accessToken, refreshToken string, expiresAt int64, err error) {
	accessToken = viper.GetString("spotify_access_token")
	refreshToken = viper.GetString("spotify_refresh_token")
	expiresAt = viper.GetInt64("spotify_token_expires_at")
	if refreshToken == "" {
		return "", "", 0, fmt.Errorf("spotify not logged in, run: crib spotify login")
	}
	return accessToken, refreshToken, expiresAt, nil
}

func SaveSpotifyToken(accessToken, refreshToken string, expiresAt int64) error {
	return SetMultiple(map[string]string{
		"spotify_access_token":     accessToken,
		"spotify_refresh_token":    refreshToken,
		"spotify_token_expires_at": fmt.Sprintf("%d", expiresAt),
	})
}

func LoadTradfri() (*Config, error) {
	cfg := &Config{
		TradfriHost:     viper.GetString("tradfri_host"),
		TradfriIdentity: viper.GetString("tradfri_identity"),
		TradfriPSK:      viper.GetString("tradfri_psk"),
	}

	if cfg.TradfriHost == "" {
		return nil, fmt.Errorf("TRÅDFRI gateway host not configured. Run: crib pair <host> <security-code>")
	}
	if cfg.TradfriIdentity == "" || cfg.TradfriPSK == "" {
		return nil, fmt.Errorf("TRÅDFRI credentials not configured. Run: crib pair <host> <security-code>")
	}

	return cfg, nil
}

func Set(key, value string) error {
	viper.Set(key, value)
	return viper.WriteConfigAs(configPath())
}

func SetMultiple(values map[string]string) error {
	for k, v := range values {
		viper.Set(k, v)
	}
	return viper.WriteConfigAs(configPath())
}

func Show() map[string]string {
	return map[string]string{
		"tradfri_host":          viper.GetString("tradfri_host"),
		"tradfri_identity":      viper.GetString("tradfri_identity"),
		"tradfri_psk":           maskToken(viper.GetString("tradfri_psk")),
		"spotify_client_id":     viper.GetString("spotify_client_id"),
		"spotify_client_secret": maskToken(viper.GetString("spotify_client_secret")),
		"config_file":           viper.ConfigFileUsed(),
	}
}

func maskToken(token string) string {
	if token == "" {
		return "(not set)"
	}
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
