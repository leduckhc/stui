package providers

import (
	"os"
	"path/filepath"
	"testing"
)

// setupConfigs writes isolated AWS, mc and stui configs into a temp HOME and
// points the relevant env vars at them.
func setupConfigs(t *testing.T, awsCfg, mcCfg, stuiCfg string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MC_CONFIG_DIR", "")
	t.Setenv("AWS_CONFIG_FILE", "")

	if awsCfg != "" {
		dir := filepath.Join(home, ".aws")
		os.MkdirAll(dir, 0700)
		os.WriteFile(filepath.Join(dir, "config"), []byte(awsCfg), 0600)
	}
	if mcCfg != "" {
		dir := filepath.Join(home, ".mc")
		os.MkdirAll(dir, 0700)
		os.WriteFile(filepath.Join(dir, "config.json"), []byte(mcCfg), 0600)
	}
	if stuiCfg != "" {
		dir := filepath.Join(home, ".config", "stui")
		os.MkdirAll(dir, 0700)
		os.WriteFile(filepath.Join(dir, "config.json"), []byte(stuiCfg), 0600)
	}
}

const mcJSON = `{
  "version": "10",
  "aliases": {
    "local": { "url": "http://localhost:9000", "accessKey": "ak", "secretKey": "sk", "api": "s3v4", "path": "on" },
    "shared": { "url": "http://minio.example.com", "accessKey": "a", "secretKey": "b" }
  }
}`

const stuiJSON = `{ "endpoints": {
  "seaweed": { "endpoint_url": "http://localhost:8333" },
  "shared":  { "endpoint_url": "http://stui.example.com", "access_key_id": "x", "secret_access_key": "y" }
} }`

const awsCfg = "[profile work]\nregion = us-east-1\n[profile shared]\nregion = eu-west-1\n"

func TestResolveUniquePerProvider(t *testing.T) {
	setupConfigs(t, awsCfg, mcJSON, stuiJSON)

	// "local" exists only in minio
	e, err := Resolve("local", "")
	if err != nil {
		t.Fatal(err)
	}
	if e.Provider != MinIO || e.Endpoint != "http://localhost:9000" || e.PathStyle == nil || !*e.PathStyle {
		t.Fatalf("local resolved wrong: %#v", e)
	}

	// "work" exists only in aws
	e, err = Resolve("work", "")
	if err != nil {
		t.Fatal(err)
	}
	if e.Provider != AWS || e.Endpoint != "" {
		t.Fatalf("work resolved wrong: %#v", e)
	}

	// "seaweed" only in stui
	e, err = Resolve("seaweed", "")
	if err != nil || e.Provider != Stui {
		t.Fatalf("seaweed resolved wrong: %#v err=%v", e, err)
	}
}

func TestResolveAmbiguous(t *testing.T) {
	setupConfigs(t, awsCfg, mcJSON, stuiJSON)

	// "shared" exists in all three providers
	_, err := Resolve("shared", "")
	amb, ok := err.(*AmbiguousError)
	if !ok {
		t.Fatalf("expected AmbiguousError, got %v", err)
	}
	if len(amb.Providers) != 3 {
		t.Fatalf("expected 3 providers, got %v", amb.Providers)
	}
}

func TestResolveWithProviderHintNoFallback(t *testing.T) {
	setupConfigs(t, awsCfg, mcJSON, stuiJSON)

	// Force minio for the ambiguous name.
	e, err := Resolve("shared", MinIO)
	if err != nil || e.Provider != MinIO || e.AccessKeyID != "a" {
		t.Fatalf("forced minio wrong: %#v err=%v", e, err)
	}

	// A name that exists elsewhere but not in the forced provider must NOT
	// fall back.
	if _, err := Resolve("work", MinIO); err == nil {
		t.Fatal("expected not-found for work in minio (no fallback)")
	}
}

func TestResolveNotFound(t *testing.T) {
	setupConfigs(t, awsCfg, mcJSON, stuiJSON)
	if _, err := Resolve("nope", ""); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestListSurfacesProviderErrors(t *testing.T) {
	// Valid stui + aws, but a corrupt mc config.
	setupConfigs(t, awsCfg, "{ this is not json", stuiJSON)

	entries, err := List()
	if err == nil {
		t.Fatal("expected error from corrupt mc config")
	}
	// Healthy providers must still contribute entries.
	var sawAWS, sawStui bool
	for _, e := range entries {
		if e.Provider == AWS {
			sawAWS = true
		}
		if e.Provider == Stui {
			sawStui = true
		}
	}
	if !sawAWS || !sawStui {
		t.Fatalf("healthy providers dropped: aws=%v stui=%v", sawAWS, sawStui)
	}
}
