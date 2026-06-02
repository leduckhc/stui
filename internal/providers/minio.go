package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// mcConfig is the subset of ~/.mc/config.json that stui reads.
type mcConfig struct {
	Aliases map[string]mcAlias `json:"aliases"`
}

type mcAlias struct {
	URL       string `json:"url"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	API       string `json:"api"`  // e.g. "s3v4"
	Path      string `json:"path"` // "auto" | "on" | "off"
}

// mcConfigPath returns the location of the mc client config, honoring
// $MC_CONFIG_DIR like mc itself does.
func mcConfigPath() (string, error) {
	if dir := os.Getenv("MC_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "config.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mc", "config.json"), nil
}

func minioEntries() ([]Entry, error) {
	path, err := mcConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read mc config: %w", err)
	}

	var cfg mcConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse mc config %s: %w", path, err)
	}

	var entries []Entry
	for name, a := range cfg.Aliases {
		if a.URL == "" {
			continue
		}
		e := Entry{
			Provider:        MinIO,
			Name:            name,
			Endpoint:        a.URL,
			AccessKeyID:     a.AccessKey,
			SecretAccessKey: a.SecretKey,
		}
		switch strings.ToLower(a.Path) {
		case "on":
			t := true
			e.PathStyle = &t
		case "off":
			f := false
			e.PathStyle = &f
		}
		entries = append(entries, e)
	}
	return entries, nil
}
