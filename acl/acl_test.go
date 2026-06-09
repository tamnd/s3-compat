package acl_test

import (
	"bytes"
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

// objectURL builds the URL for bucket/key using the target endpoint.
func objectURL(bucket, key string) string {
	endpoint := client.Target.Endpoint
	if endpoint == "" {
		return "https://" + bucket + ".s3.amazonaws.com/" + key
	}
	endpoint = strings.TrimRight(endpoint, "/")
	return endpoint + "/" + bucket + "/" + key
}

func TestGetBucketACL(t *testing.T) {
	skip.Feature(t, "acl", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("acl")
	client.CreateBucket(t, bucket)

	out, err := client.S3.GetBucketAcl(ctx, &s3.GetBucketAclInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)
	require.NotNil(t, out.Owner)

	ownerID := aws.ToString(out.Owner.ID)
	displayName := aws.ToString(out.Owner.DisplayName)
	require.True(t, ownerID != "" || displayName != "", "expected Owner.ID or Owner.DisplayName to be non-empty")
}

func TestPutBucketACLPublicRead(t *testing.T) {
	skip.Feature(t, "acl", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("acl")
	client.CreateBucket(t, bucket)

	_, err := client.S3.PutBucketAcl(ctx, &s3.PutBucketAclInput{
		Bucket: aws.String(bucket),
		ACL:    types.BucketCannedACLPublicRead,
	})
	require.NoError(t, err)

	_, err = client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("test-acl.txt"),
		Body:   bytes.NewReader([]byte("public content")),
	})
	require.NoError(t, err)

	url := objectURL(bucket, "test-acl.txt")
	hreq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err)
	resp, err := client.HTTPClient.Do(hreq)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetObjectACL(t *testing.T) {
	skip.Feature(t, "acl", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("acl")
	client.CreateBucket(t, bucket)

	_, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("obj.txt"),
		Body:   bytes.NewReader([]byte("data")),
	})
	require.NoError(t, err)

	out, err := client.S3.GetObjectAcl(ctx, &s3.GetObjectAclInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("obj.txt"),
	})
	require.NoError(t, err)
	require.NotEmpty(t, out.Grants)
}

func TestPutObjectACLPublicRead(t *testing.T) {
	skip.Feature(t, "acl", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("acl")
	client.CreateBucket(t, bucket)

	_, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("pub.txt"),
		Body:   bytes.NewReader([]byte("public object")),
	})
	require.NoError(t, err)

	_, err = client.S3.PutObjectAcl(ctx, &s3.PutObjectAclInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("pub.txt"),
		ACL:    types.ObjectCannedACLPublicRead,
	})
	require.NoError(t, err)

	url := objectURL(bucket, "pub.txt")
	hreq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err)
	resp, err := client.HTTPClient.Do(hreq)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
