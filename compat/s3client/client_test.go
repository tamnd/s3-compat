package s3client_test

import (
	"path/filepath"
	"testing"

	"github.com/tamnd/s3-compat/compat/s3client"
)

func TestLoadMissingConfig(t *testing.T) {
	nonexistent := filepath.Join(t.TempDir(), "does-not-exist.yml")
	t.Setenv("S3COMPAT_CONFIG", nonexistent)
	t.Setenv("S3COMPAT_TARGET", "minio")

	_, err := s3client.Load("")
	if err == nil {
		t.Fatal("expected error when config file does not exist, got nil")
	}
}
