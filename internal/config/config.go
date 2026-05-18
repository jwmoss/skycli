package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	BaseURL           string    `json:"base_url,omitempty"`
	APIVersion        string    `json:"api_version,omitempty"`
	AuthScheme        string    `json:"auth_scheme,omitempty"`
	SecretsBackend    string    `json:"secrets_backend,omitempty"`
	AccessToken       string    `json:"access_token,omitempty"`
	RefreshToken      string    `json:"refresh_token,omitempty"`
	AccessTokenExpAt  time.Time `json:"access_token_expires_at,omitempty"`
	DeviceFingerprint string    `json:"device_fingerprint,omitempty"`
	DefaultFrameID    int64     `json:"default_frame_id,omitempty"`
}

const (
	DefaultBaseURL    = "https://app.ourskylight.com"
	DefaultAPIVersion = "2026-04-15"
	DefaultAuthScheme = "Bearer"
)

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "skycli", "config.json"), nil
}

func RootDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "skycli"), nil
}

func Load(path string) (*Config, error) {
	if path == "" {
		p, err := DefaultPath()
		if err != nil {
			return nil, err
		}
		path = p
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return applyDefaults(&Config{}), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	c := &Config{}
	if err := json.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return applyDefaults(c), nil
}

func Save(path string, c *Config) error {
	applyDefaults(c)
	if path == "" {
		p, err := DefaultPath()
		if err != nil {
			return err
		}
		path = p
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write config tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename config: %w", err)
	}
	return nil
}

func applyDefaults(c *Config) *Config {
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	if c.APIVersion == "" {
		c.APIVersion = DefaultAPIVersion
	}
	if c.AuthScheme == "" {
		c.AuthScheme = DefaultAuthScheme
	}
	return c
}
