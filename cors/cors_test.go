package cors_test

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/require"
	"github.com/tamnd/s3-compat/compat/s3client"
	"github.com/tamnd/s3-compat/compat/skip"
)

var client *s3client.Client

func TestMain(m *testing.M) {
	c, err := s3client.Load("")
	if err != nil {
		panic(err.Error())
	}
	client = c
	os.Exit(m.Run())
}

// bucketURL builds the URL for an S3 bucket using the target endpoint.
// For path-style endpoints it returns endpoint/bucket; otherwise bucket.host.
func bucketURL(bucket string) string {
	endpoint := client.Target.Endpoint
	if endpoint == "" {
		// Virtual-hosted style with no explicit endpoint; not directly usable.
		return "https://" + bucket + ".s3.amazonaws.com"
	}
	// Strip trailing slash if present.
	endpoint = strings.TrimRight(endpoint, "/")
	return endpoint + "/" + bucket
}

func TestCORSPutAndGet(t *testing.T) {
	skip.Feature(t, "cors", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("cors")
	client.CreateBucket(t, bucket)

	_, err := client.S3.PutBucketCors(ctx, &s3.PutBucketCorsInput{
		Bucket: aws.String(bucket),
		CORSConfiguration: &types.CORSConfiguration{
			CORSRules: []types.CORSRule{
				{
					AllowedOrigins: []string{"http://example.com"},
					AllowedMethods: []string{"GET"},
				},
			},
		},
	})
	require.NoError(t, err)

	out, err := client.S3.GetBucketCors(ctx, &s3.GetBucketCorsInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)
	require.Len(t, out.CORSRules, 1)
	require.Equal(t, "http://example.com", out.CORSRules[0].AllowedOrigins[0])
}

func TestCORSDelete(t *testing.T) {
	skip.Feature(t, "cors", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("cors")
	client.CreateBucket(t, bucket)

	_, err := client.S3.PutBucketCors(ctx, &s3.PutBucketCorsInput{
		Bucket: aws.String(bucket),
		CORSConfiguration: &types.CORSConfiguration{
			CORSRules: []types.CORSRule{
				{
					AllowedOrigins: []string{"http://example.com"},
					AllowedMethods: []string{"GET"},
				},
			},
		},
	})
	require.NoError(t, err)

	_, err = client.S3.DeleteBucketCors(ctx, &s3.DeleteBucketCorsInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)

	_, err = client.S3.GetBucketCors(ctx, &s3.GetBucketCorsInput{
		Bucket: aws.String(bucket),
	})
	require.Error(t, err)
}

func TestCORSPreflight(t *testing.T) {
	skip.Feature(t, "cors", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("cors")
	client.CreateBucket(t, bucket)

	maxAge := int32(300)
	_, err := client.S3.PutBucketCors(ctx, &s3.PutBucketCorsInput{
		Bucket: aws.String(bucket),
		CORSConfiguration: &types.CORSConfiguration{
			CORSRules: []types.CORSRule{
				{
					AllowedOrigins: []string{"http://cors-test.example"},
					AllowedMethods: []string{"GET"},
					MaxAgeSeconds:  &maxAge,
				},
			},
		},
	})
	require.NoError(t, err)

	url := bucketURL(bucket)
	req, err := http.NewRequest("OPTIONS", url, nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://cors-test.example")
	req.Header.Set("Access-Control-Request-Method", "GET")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Origin"))
}

func TestCORSPreflightNoMatch(t *testing.T) {
	skip.Feature(t, "cors", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("cors")
	client.CreateBucket(t, bucket)

	maxAge := int32(300)
	_, err := client.S3.PutBucketCors(ctx, &s3.PutBucketCorsInput{
		Bucket: aws.String(bucket),
		CORSConfiguration: &types.CORSConfiguration{
			CORSRules: []types.CORSRule{
				{
					AllowedOrigins: []string{"http://cors-test.example"},
					AllowedMethods: []string{"GET"},
					MaxAgeSeconds:  &maxAge,
				},
			},
		},
	})
	require.NoError(t, err)

	url := bucketURL(bucket)
	req, err := http.NewRequest("OPTIONS", url, nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://not-allowed.example")
	req.Header.Set("Access-Control-Request-Method", "GET")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
}

func TestCORSWildcardOrigin(t *testing.T) {
	skip.Feature(t, "cors", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("cors")
	client.CreateBucket(t, bucket)

	_, err := client.S3.PutBucketCors(ctx, &s3.PutBucketCorsInput{
		Bucket: aws.String(bucket),
		CORSConfiguration: &types.CORSConfiguration{
			CORSRules: []types.CORSRule{
				{
					AllowedOrigins: []string{"*"},
					AllowedMethods: []string{"GET"},
				},
			},
		},
	})
	require.NoError(t, err)

	url := bucketURL(bucket)
	req, err := http.NewRequest("OPTIONS", url, nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://anything.example")
	req.Header.Set("Access-Control-Request-Method", "GET")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Origin"))
}
