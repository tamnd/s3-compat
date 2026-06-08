package policy_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

func buildPolicy(bucket string) string {
	return fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"arn:aws:s3:::%s/*"}]}`, bucket)
}

// objectURL builds the URL for bucket/key using the target endpoint.
func objectURL(bucket, key string) string {
	endpoint := client.Target.Endpoint
	if endpoint == "" {
		return "https://" + bucket + ".s3.amazonaws.com/" + key
	}
	endpoint = strings.TrimRight(endpoint, "/")
	return endpoint + "/" + bucket + "/" + key
}

func TestBucketPolicyPutAndGet(t *testing.T) {
	skip.Feature(t, "policy", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("pol")
	client.CreateBucket(t, bucket)

	policy := buildPolicy(bucket)
	_, err := client.S3.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
		Bucket: aws.String(bucket),
		Policy: aws.String(policy),
	})
	require.NoError(t, err)

	out, err := client.S3.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)
	require.Contains(t, aws.ToString(out.Policy), bucket)
}

func TestBucketPolicyDelete(t *testing.T) {
	skip.Feature(t, "policy", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("pol")
	client.CreateBucket(t, bucket)

	policy := buildPolicy(bucket)
	_, err := client.S3.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
		Bucket: aws.String(bucket),
		Policy: aws.String(policy),
	})
	require.NoError(t, err)

	_, err = client.S3.DeleteBucketPolicy(ctx, &s3.DeleteBucketPolicyInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)

	_, err = client.S3.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{
		Bucket: aws.String(bucket),
	})
	require.Error(t, err)
}

func TestBucketPolicyPublicRead(t *testing.T) {
	skip.Feature(t, "policy", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("pol")
	client.CreateBucket(t, bucket)

	policy := buildPolicy(bucket)
	_, err := client.S3.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
		Bucket: aws.String(bucket),
		Policy: aws.String(policy),
	})
	require.NoError(t, err)

	_, err = client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("public.txt"),
		Body:   bytes.NewReader([]byte("public content")),
	})
	require.NoError(t, err)

	url := objectURL(bucket, "public.txt")
	resp, err := http.Get(url) //nolint:noctx
	require.NoError(t, err)
	defer resp.Body.Close()
	// Drain the body.
	_, _ = io.Copy(io.Discard, resp.Body)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestBucketPolicyMalformedJSON(t *testing.T) {
	skip.Feature(t, "policy", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("pol")
	client.CreateBucket(t, bucket)

	_, err := client.S3.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
		Bucket: aws.String(bucket),
		Policy: aws.String("not-json"),
	})
	require.Error(t, err)
}
