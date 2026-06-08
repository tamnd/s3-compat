package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tamnd/s3-compat/compat/config"
)

const testYAML = `
defaults:
  region: us-east-1
  bucket_prefix: s3compat-

targets:
  minio:
    endpoint: http://localhost:9000
    access_key: minioadmin
    secret_key: minioadmin
    use_path_style: true
    features:
      versioning: true
      multipart: true
`

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "targets.yml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing temp yaml: %v", err)
	}
	return path
}

func TestLoadKnownTarget(t *testing.T) {
	path := writeTempYAML(t, testYAML)
	t.Setenv("S3COMPAT_CONFIG", path)
	t.Setenv("S3COMPAT_TARGET", "minio")

	target, err := config.Load("")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if target.Endpoint != "http://localhost:9000" {
		t.Errorf("unexpected endpoint %q", target.Endpoint)
	}
	if target.Region != "us-east-1" {
		t.Errorf("expected default region us-east-1, got %q", target.Region)
	}
}

func TestLoadUnknownTarget(t *testing.T) {
	path := writeTempYAML(t, testYAML)
	t.Setenv("S3COMPAT_CONFIG", path)

	_, err := config.Load("doesnotexist")
	if err == nil {
		t.Fatal("expected error for unknown target, got nil")
	}
	// Error must mention the target name.
	errStr := err.Error()
	if len(errStr) == 0 {
		t.Fatal("error message is empty")
	}
	// Quick substring check without importing strings.
	found := false
	needle := "doesnotexist"
	for i := 0; i <= len(errStr)-len(needle); i++ {
		if errStr[i:i+len(needle)] == needle {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("error %q does not mention target name %q", errStr, needle)
	}
}

func TestLoadEnvOverride(t *testing.T) {
	path := writeTempYAML(t, testYAML)
	t.Setenv("S3COMPAT_CONFIG", path)
	t.Setenv("S3COMPAT_TARGET", "minio")
	t.Setenv("S3COMPAT_ENDPOINT", "http://custom-host:9000")

	target, err := config.Load("")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if target.Endpoint != "http://custom-host:9000" {
		t.Errorf("expected overridden endpoint, got %q", target.Endpoint)
	}
}
