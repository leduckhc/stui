package providers

import "github.com/natevick/stui/internal/config"

func stuiEntries() ([]Entry, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for name, ep := range cfg.Endpoints {
		if ep.EndpointURL == "" {
			continue
		}
		entries = append(entries, Entry{
			Provider:        Stui,
			Name:            name,
			Endpoint:        ep.EndpointURL,
			Region:          ep.Region,
			PathStyle:       ep.PathStyle,
			AccessKeyID:     ep.AccessKeyID,
			SecretAccessKey: ep.SecretAccessKey,
			SessionToken:    ep.SessionToken,
		})
	}
	return entries, nil
}
