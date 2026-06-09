package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Features lists which optional S3 features the target supports.
type Features struct {
	Versioning       bool `yaml:"versioning"`
	ObjectLock       bool `yaml:"object_lock"`
	Lifecycle        bool `yaml:"lifecycle"`
	CORS             bool `yaml:"cors"`
	ACL              bool `yaml:"acl"`
	Policy           bool `yaml:"policy"`
	Notifications    bool `yaml:"notifications"`
	EncryptionSSES3  bool `yaml:"encryption_sse_s3"`
	EncryptionSSEKMS bool `yaml:"encryption_sse_kms"`
	EncryptionSSEC   bool `yaml:"encryption_sse_c"`
	Replication      bool `yaml:"replication"`
	Select           bool `yaml:"select"`
	Tagging          bool `yaml:"tagging"`
	Multipart        bool `yaml:"multipart"`
}

// Target holds connection and capability settings for one S3 endpoint.
type Target struct {
	Endpoint      string   `yaml:"endpoint"`
	AccessKey     string   `yaml:"access_key"`
	SecretKey     string   `yaml:"secret_key"`
	Region        string   `yaml:"region"`
	UsePathStyle  bool     `yaml:"use_path_style"`
	SkipTLSVerify bool     `yaml:"skip_tls_verify"`
	CACert        string   `yaml:"ca_cert"`
	BucketPrefix  string   `yaml:"bucket_prefix"`
	Features      Features `yaml:"features"`
}

// File is the top-level structure of targets.yml.
type File struct {
	Defaults Target            `yaml:"defaults"`
	Targets  map[string]Target `yaml:"targets"`
}

// Load reads the target config from S3COMPAT_CONFIG (default: configs/targets.yml)
// and returns the target named by S3COMPAT_TARGET (default: minio).
// Env vars S3COMPAT_ENDPOINT, S3COMPAT_ACCESS_KEY, S3COMPAT_SECRET_KEY,
// S3COMPAT_REGION, S3COMPAT_USE_PATH_STYLE, S3COMPAT_SKIP_TLS_VERIFY
// override values from the file.
func Load(targetName string) (*Target, error) {
	cfgPath := os.Getenv("S3COMPAT_CONFIG")
	if cfgPath == "" {
		cfgPath = "configs/targets.yml"
	}
	if targetName == "" {
		targetName = os.Getenv("S3COMPAT_TARGET")
	}
	if targetName == "" {
		targetName = "minio"
	}

	absCfgPath, err := filepath.Abs(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("resolving config path %s: %w", cfgPath, err)
	}
	cfgDir := filepath.Dir(absCfgPath)

	data, err := os.ReadFile(absCfgPath)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", absCfgPath, err)
	}

	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", cfgPath, err)
	}

	t, ok := f.Targets[targetName]
	if !ok {
		names := make([]string, 0, len(f.Targets))
		for k := range f.Targets {
			names = append(names, k)
		}
		sort.Strings(names)
		return nil, fmt.Errorf("target %q not found in %s; available: %v", targetName, cfgPath, names)
	}

	// Merge defaults for zero values.
	if t.Region == "" {
		t.Region = f.Defaults.Region
	}
	if t.BucketPrefix == "" {
		t.BucketPrefix = f.Defaults.BucketPrefix
	}

	// Expand ${VAR} in credentials.
	t.AccessKey = os.ExpandEnv(t.AccessKey)
	t.SecretKey = os.ExpandEnv(t.SecretKey)

	// Env overrides.
	if v := os.Getenv("S3COMPAT_ENDPOINT"); v != "" {
		t.Endpoint = v
	}
	if v := os.Getenv("S3COMPAT_ACCESS_KEY"); v != "" {
		t.AccessKey = v
	}
	if v := os.Getenv("S3COMPAT_SECRET_KEY"); v != "" {
		t.SecretKey = v
	}
	if v := os.Getenv("S3COMPAT_REGION"); v != "" {
		t.Region = v
	}
	if v := os.Getenv("S3COMPAT_USE_PATH_STYLE"); v != "" {
		b, _ := strconv.ParseBool(v)
		t.UsePathStyle = b
	}
	if v := os.Getenv("S3COMPAT_SKIP_TLS_VERIFY"); v != "" {
		b, _ := strconv.ParseBool(v)
		t.SkipTLSVerify = b
	}
	if v := os.Getenv("S3COMPAT_CA_CERT"); v != "" {
		t.CACert = v
	}

	// Merge defaults for ca_cert.
	if t.CACert == "" && f.Defaults.CACert != "" {
		t.CACert = f.Defaults.CACert
	}
	// Resolve ca_cert relative to the config file directory so tests can be run
	// from any working directory.
	if t.CACert != "" && !filepath.IsAbs(t.CACert) {
		t.CACert = filepath.Join(cfgDir, t.CACert)
	}

	return &t, nil
}
