package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFileReturnsEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load on missing file should not error: %v", err)
	}
	if cfg == nil || len(cfg.Endpoints) != 0 {
		t.Fatalf("expected empty config, got %#v", cfg)
	}
	if _, ok := cfg.EndpointFor("anything"); ok {
		t.Fatal("expected no endpoint for unknown profile")
	}
}

func TestLoadAndEndpointFor(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".config", "stui")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	body := `{
  "endpoints": {
    "seaweed": { "endpoint_url": "http://localhost:8333" },
    "minio":   { "endpoint_url": "http://localhost:9000", "path_style": false, "region": "us-east-1", "access_key_id": "AKIDLOCAL", "secret_access_key": "secret123" }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	sw, ok := cfg.EndpointFor("seaweed")
	if !ok || sw.EndpointURL != "http://localhost:8333" || sw.PathStyle != nil {
		t.Fatalf("seaweed endpoint wrong: %#v ok=%v", sw, ok)
	}

	mi, ok := cfg.EndpointFor("minio")
	if !ok || mi.PathStyle == nil || *mi.PathStyle != false || mi.Region != "us-east-1" {
		t.Fatalf("minio endpoint wrong: %#v ok=%v", mi, ok)
	}
	if mi.AccessKeyID != "AKIDLOCAL" || mi.SecretAccessKey != "secret123" {
		t.Fatalf("minio credentials wrong: %#v", mi)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	pathStyle := true
	cfg := &Config{Endpoints: map[string]EndpointConfig{
		"local": {EndpointURL: "http://localhost:8333", PathStyle: &pathStyle},
	}}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	ep, ok := loaded.EndpointFor("local")
	if !ok || ep.EndpointURL != "http://localhost:8333" || ep.PathStyle == nil || !*ep.PathStyle {
		t.Fatalf("round trip failed: %#v ok=%v", ep, ok)
	}
}
