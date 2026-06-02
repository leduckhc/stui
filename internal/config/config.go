// Package config handles stui's own configuration, kept separate from the
// standard AWS config files (~/.aws/config, ~/.aws/credentials).
//
// AWS profiles, credentials, regions and SSO are read from ~/.aws as usual via
// the AWS SDK. stui never writes to or extends the AWS config schema. Anything
// that is specific to stui — most importantly custom S3 endpoints for
// S3-compatible servers like SeaweedFS, MinIO or Ceph — lives here, in
// ~/.config/stui/config.json.
//
// This is one of three independent provider configs stui understands (see
// internal/providers): "aws" (~/.aws), "minio" (~/.mc/config.json) and "stui"
// (this file). Providers never fall back to one another.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// EndpointConfig describes how to reach a custom S3-compatible endpoint for a
// given AWS profile. Credentials and region still come from the AWS profile;
// these fields only override what the AWS SDK can't express for non-AWS
// backends.
type EndpointConfig struct {
	// EndpointURL is the base URL of the S3-compatible server, e.g.
	// "http://localhost:8333" (SeaweedFS) or "http://localhost:9000" (MinIO).
	EndpointURL string `json:"endpoint_url"`

	// AccessKeyID and SecretAccessKey are optional static credentials for the
	// endpoint. When set, they are used directly and no ~/.aws profile is
	// required. When empty, credentials fall back to the AWS profile / SDK
	// default chain (env, ~/.aws/credentials, SSO, etc.).
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	// SessionToken is an optional temporary-credential session token.
	SessionToken string `json:"session_token,omitempty"`

	// PathStyle forces path-style ("endpoint/bucket/key") vs virtual-hosted
	// ("bucket.endpoint/key") addressing. If nil, path-style is used
	// automatically whenever EndpointURL is set, which is what most
	// S3-compatible servers require.
	PathStyle *bool `json:"path_style,omitempty"`

	// Region optionally overrides the region for this endpoint. Many
	// S3-compatible servers accept any value (e.g. "us-east-1").
	Region string `json:"region,omitempty"`
}

// Config is stui's persisted configuration.
type Config struct {
	// Endpoints maps an AWS profile name to its custom endpoint settings.
	Endpoints map[string]EndpointConfig `json:"endpoints,omitempty"`

	path string
}

// configFileName is the stui config file inside ~/.config/stui.
const configFileName = "config.json"

// Path returns the location of the stui config file.
func Path() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "stui", configFileName), nil
}

// Load reads ~/.config/stui/config.json. A missing file is not an error; it
// returns an empty config so stui works out of the box.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	cfg := &Config{Endpoints: map[string]EndpointConfig{}, path: path}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read stui config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse stui config %s: %w", path, err)
	}
	if cfg.Endpoints == nil {
		cfg.Endpoints = map[string]EndpointConfig{}
	}
	cfg.path = path
	return cfg, nil
}

// Save writes the config to disk with 0600 permissions, creating the directory
// (0700) if needed.
func (c *Config) Save() error {
	path := c.path
	if path == "" {
		p, err := Path()
		if err != nil {
			return err
		}
		path = p
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stui config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write stui config: %w", err)
	}
	return nil
}

// EndpointFor returns the endpoint configuration for a profile, if one is
// defined.
func (c *Config) EndpointFor(profile string) (EndpointConfig, bool) {
	if c == nil || c.Endpoints == nil || profile == "" {
		return EndpointConfig{}, false
	}
	ep, ok := c.Endpoints[profile]
	if !ok || ep.EndpointURL == "" {
		return EndpointConfig{}, false
	}
	return ep, true
}
