// Package skip provides helpers that skip tests for unsupported S3 features.
package skip

import (
	"strings"
	"testing"

	"github.com/tamnd/s3-compat/compat/config"
)

// normalize maps feature names to their canonical key in the Features struct.
func normalize(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, "-", "_"))
}

// Feature skips t if the named feature is disabled in features.
// Unknown feature names are considered enabled (no skip).
func Feature(t testing.TB, name string, features config.Features) {
	t.Helper()
	key := normalize(name)
	enabled := featureEnabled(key, features)
	if !enabled {
		t.Skipf("target does not support feature %q", name)
	}
}

func featureEnabled(key string, f config.Features) bool {
	switch key {
	case "versioning":
		return f.Versioning
	case "object_lock", "objectlock":
		return f.ObjectLock
	case "lifecycle":
		return f.Lifecycle
	case "cors":
		return f.CORS
	case "acl":
		return f.ACL
	case "policy":
		return f.Policy
	case "notifications":
		return f.Notifications
	case "encryption_sse_s3":
		return f.EncryptionSSES3
	case "encryption_sse_kms":
		return f.EncryptionSSEKMS
	case "encryption_sse_c":
		return f.EncryptionSSEC
	case "replication":
		return f.Replication
	case "select":
		return f.Select
	case "tagging":
		return f.Tagging
	case "multipart":
		return f.Multipart
	case "strict_auth":
		return f.StrictAuth
	case "strict_delete_bucket":
		return f.StrictDeleteBucket
	case "list_buckets_consistent":
		return f.ListBucketsConsistent
	case "list_multipart":
		return f.ListMultipart
	default:
		// Unknown feature: assume enabled, do not skip.
		return true
	}
}
