package aws

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestEndpointFromFlatProfile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config")
	os.WriteFile(cfgPath, []byte("[profile sw]\nregion = us-east-1\nendpoint_url = http://localhost:8333\n"), 0600)
	t.Setenv("AWS_CONFIG_FILE", cfgPath)
	t.Setenv("AWS_ACCESS_KEY_ID", "x")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "y")

	c, err := NewClient(context.Background(), "sw", "")
	if err != nil {
		t.Fatal(err)
	}
	if c.Endpoint != "http://localhost:8333" {
		t.Fatalf("endpoint not resolved, got %q", c.Endpoint)
	}
	if !c.S3.Options().UsePathStyle {
		t.Fatal("expected path-style addressing for custom endpoint")
	}
}

func TestServicesSectionEndpoint(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config")
	os.WriteFile(cfgPath, []byte("[profile sw]\nregion = us-east-1\nservices = local\n\n[services local]\ns3 =\n  endpoint_url = http://localhost:9000\n"), 0600)
	t.Setenv("AWS_CONFIG_FILE", cfgPath)
	t.Setenv("AWS_ACCESS_KEY_ID", "x")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "y")

	c, err := NewClient(context.Background(), "sw", "")
	if err != nil {
		t.Fatal(err)
	}
	if c.Endpoint != "http://localhost:9000" {
		t.Fatalf("services-section endpoint not resolved, got %q", c.Endpoint)
	}
}


func TestStaticCredentialsSkipProfile(t *testing.T) {
	// Point AWS config/credentials at empty temp files so the default chain
	// has nothing; static creds from ClientOptions must still work.
	dir := t.TempDir()
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_PROFILE", "")

	c, err := NewClientWithOptions(context.Background(), ClientOptions{
		Profile:         "minio", // does not exist in ~/.aws
		Region:          "us-east-1",
		Endpoint:        "http://localhost:9000",
		AccessKeyID:     "AKIDLOCAL",
		SecretAccessKey: "secret123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.Endpoint != "http://localhost:9000" {
		t.Fatalf("endpoint wrong: %q", c.Endpoint)
	}
	if !c.S3.Options().UsePathStyle {
		t.Fatal("expected path-style for custom endpoint")
	}
	creds, err := c.Config.Credentials.Retrieve(context.Background())
	if err != nil {
		t.Fatalf("retrieve creds: %v", err)
	}
	if creds.AccessKeyID != "AKIDLOCAL" || creds.SecretAccessKey != "secret123" {
		t.Fatalf("static creds not applied: %#v", creds)
	}
}

func TestWithRegionPreservesStaticCreds(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_REGION", "")

	c, err := NewClientWithOptions(context.Background(), ClientOptions{
		Profile:         "minio",
		Region:          "us-east-1",
		Endpoint:        "http://localhost:9000",
		AccessKeyID:     "AKIDLOCAL",
		SecretAccessKey: "secret123",
	})
	if err != nil {
		t.Fatal(err)
	}

	c2, err := c.WithRegion(context.Background(), "eu-west-1")
	if err != nil {
		t.Fatalf("WithRegion: %v", err)
	}
	if c2.Endpoint != "http://localhost:9000" || !c2.S3.Options().UsePathStyle {
		t.Fatalf("WithRegion lost endpoint/path-style: %#v", c2.Endpoint)
	}
	creds, err := c2.Config.Credentials.Retrieve(context.Background())
	if err != nil {
		t.Fatalf("retrieve creds after WithRegion: %v", err)
	}
	if creds.AccessKeyID != "AKIDLOCAL" || creds.SecretAccessKey != "secret123" {
		t.Fatalf("WithRegion lost static creds: %#v", creds)
	}
}
