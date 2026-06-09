package s3client

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/tamnd/s3-compat/compat/config"
)

// Client wraps an S3 client with the loaded target configuration.
type Client struct {
	S3         *s3.Client
	Target     *config.Target
	HTTPClient *http.Client // use for raw HTTP calls (presigned URLs, etc.)
}

// Load builds an S3 client for the named target.
func Load(targetName string) (*Client, error) {
	target, err := config.Load(targetName)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	var httpClient *http.Client
	switch {
	case target.CACert != "":
		pem, err := os.ReadFile(target.CACert)
		if err != nil {
			return nil, fmt.Errorf("reading CA cert %s: %w", target.CACert, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("parsing CA cert %s: no valid PEM block found", target.CACert)
		}
		httpClient = &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool},
		}}
	case target.SkipTLSVerify:
		httpClient = &http.Client{Transport: &http.Transport{ //nolint:gosec
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}}
	}

	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(target.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(target.AccessKey, target.SecretKey, ""),
		),
	}
	if httpClient != nil {
		opts = append(opts, awsconfig.WithHTTPClient(httpClient))
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("building aws config: %w", err)
	}

	s3Opts := []func(*s3.Options){
		func(o *s3.Options) {
			o.UsePathStyle = target.UsePathStyle
			if target.Endpoint != "" {
				o.BaseEndpoint = aws.String(target.Endpoint)
			}
		},
	}

	rawHTTP := httpClient
	if rawHTTP == nil {
		rawHTTP = http.DefaultClient
	}
	return &Client{
		S3:         s3.NewFromConfig(cfg, s3Opts...),
		Target:     target,
		HTTPClient: rawHTTP,
	}, nil
}

// MustLoad calls Load and fails the test on error.
func MustLoad(t testing.TB) *Client {
	t.Helper()
	c, err := Load("")
	if err != nil {
		t.Fatalf("s3client.Load: %v", err)
	}
	return c
}

// RandBucket returns a bucket name of the form "<prefix><suite>-<hex8>".
func (c *Client) RandBucket(suite string) string {
	return fmt.Sprintf("%s%s-%08x", c.Target.BucketPrefix, suite, rand.Uint32())
}

// CreateBucket creates a bucket and registers cleanup to delete it.
func (c *Client) CreateBucket(t testing.TB, name string) {
	t.Helper()
	ctx := context.Background()
	_, err := c.S3.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(name),
	})
	if err != nil {
		t.Fatalf("CreateBucket %q: %v", name, err)
	}
	t.Cleanup(func() {
		c.DeleteBucketForce(context.Background(), name)
	})
}

// DeleteBucketForce drains all objects and versions, then deletes the bucket.
// Errors are swallowed so cleanup never fails the test suite.
func (c *Client) DeleteBucketForce(ctx context.Context, bucket string) {
	// Delete all object versions (handles versioned buckets).
	vPager := s3.NewListObjectVersionsPaginator(c.S3, &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
	})
	for vPager.HasMorePages() {
		page, err := vPager.NextPage(ctx)
		if err != nil {
			break
		}
		var objs []types.ObjectIdentifier
		for _, v := range page.Versions {
			objs = append(objs, types.ObjectIdentifier{
				Key:       v.Key,
				VersionId: v.VersionId,
			})
		}
		for _, dm := range page.DeleteMarkers {
			objs = append(objs, types.ObjectIdentifier{
				Key:       dm.Key,
				VersionId: dm.VersionId,
			})
		}
		if len(objs) > 0 {
			_, _ = c.S3.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(bucket),
				Delete: &types.Delete{Objects: objs},
			})
		}
	}

	// Delete any remaining unversioned objects.
	oPager := s3.NewListObjectsV2Paginator(c.S3, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	for oPager.HasMorePages() {
		page, err := oPager.NextPage(ctx)
		if err != nil {
			break
		}
		var objs []types.ObjectIdentifier
		for _, obj := range page.Contents {
			objs = append(objs, types.ObjectIdentifier{Key: obj.Key})
		}
		if len(objs) > 0 {
			_, _ = c.S3.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(bucket),
				Delete: &types.Delete{Objects: objs},
			})
		}
	}

	_, _ = c.S3.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	})
}

// PutObject uploads body to bucket/key, failing the test on error.
func (c *Client) PutObject(t testing.TB, bucket, key string, body []byte) {
	t.Helper()
	_, err := c.S3.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	})
	if err != nil {
		t.Fatalf("PutObject %q/%q: %v", bucket, key, err)
	}
}

// GetObjectBody downloads bucket/key and returns its body, failing on error.
func (c *Client) GetObjectBody(t testing.TB, bucket, key string) []byte {
	t.Helper()
	out, err := c.S3.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("GetObject %q/%q: %v", bucket, key, err)
	}
	defer out.Body.Close()
	data, readErr := io.ReadAll(out.Body)
	if readErr != nil {
		t.Fatalf("reading GetObject body %q/%q: %v", bucket, key, readErr)
	}
	return data
}
