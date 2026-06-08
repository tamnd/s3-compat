package versioning_test

import (
	"bytes"
	"context"
	"io"
	"os"
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

func enableVersioning(t *testing.T, ctx context.Context, bucket string) {
	t.Helper()
	_, err := client.S3.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(bucket),
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: types.BucketVersioningStatusEnabled,
		},
	})
	require.NoError(t, err)
}

func TestVersioningEnable(t *testing.T) {
	skip.Feature(t, "versioning", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("ver")
	client.CreateBucket(t, bucket)

	enableVersioning(t, ctx, bucket)

	out, err := client.S3.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)
	require.Equal(t, types.BucketVersioningStatusEnabled, out.Status)
}

func TestVersioningPutReturnsVersionID(t *testing.T) {
	skip.Feature(t, "versioning", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("ver")
	client.CreateBucket(t, bucket)
	enableVersioning(t, ctx, bucket)

	out, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("file.txt"),
		Body:   bytes.NewReader([]byte("hello")),
	})
	require.NoError(t, err)
	require.NotNil(t, out.VersionId)
}

func TestVersioningMultipleVersions(t *testing.T) {
	skip.Feature(t, "versioning", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("ver")
	client.CreateBucket(t, bucket)
	enableVersioning(t, ctx, bucket)

	_, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("file.txt"),
		Body:   bytes.NewReader([]byte("version1")),
	})
	require.NoError(t, err)

	_, err = client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("file.txt"),
		Body:   bytes.NewReader([]byte("version2")),
	})
	require.NoError(t, err)

	listOut, err := client.S3.ListObjectVersions(ctx, &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(listOut.Versions), 2)
}

func TestVersioningGetByVersionID(t *testing.T) {
	skip.Feature(t, "versioning", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("ver")
	client.CreateBucket(t, bucket)
	enableVersioning(t, ctx, bucket)

	put1, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("file.txt"),
		Body:   bytes.NewReader([]byte("body1")),
	})
	require.NoError(t, err)
	v1 := put1.VersionId

	_, err = client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("file.txt"),
		Body:   bytes.NewReader([]byte("body2")),
	})
	require.NoError(t, err)

	getOut, err := client.S3.GetObject(ctx, &s3.GetObjectInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String("file.txt"),
		VersionId: v1,
	})
	require.NoError(t, err)
	defer getOut.Body.Close()

	data, err := io.ReadAll(getOut.Body)
	require.NoError(t, err)
	require.Equal(t, "body1", string(data))
}

func TestVersioningDeleteCreatesMarker(t *testing.T) {
	skip.Feature(t, "versioning", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("ver")
	client.CreateBucket(t, bucket)
	enableVersioning(t, ctx, bucket)

	_, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("file.txt"),
		Body:   bytes.NewReader([]byte("data")),
	})
	require.NoError(t, err)

	_, err = client.S3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("file.txt"),
	})
	require.NoError(t, err)

	listOut, err := client.S3.ListObjectVersions(ctx, &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)

	var found bool
	for _, dm := range listOut.DeleteMarkers {
		if aws.ToString(dm.Key) == "file.txt" {
			found = true
			break
		}
	}
	require.True(t, found, "expected delete marker for file.txt")

	_, err = client.S3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("file.txt"),
	})
	require.Error(t, err)
}

func TestVersioningDeleteByVersionID(t *testing.T) {
	skip.Feature(t, "versioning", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("ver")
	client.CreateBucket(t, bucket)
	enableVersioning(t, ctx, bucket)

	putOut, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("file.txt"),
		Body:   bytes.NewReader([]byte("data")),
	})
	require.NoError(t, err)
	versionID := putOut.VersionId

	_, err = client.S3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String("file.txt"),
		VersionId: versionID,
	})
	require.NoError(t, err)

	_, err = client.S3.GetObject(ctx, &s3.GetObjectInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String("file.txt"),
		VersionId: versionID,
	})
	require.Error(t, err)
}

func TestVersioningSuspend(t *testing.T) {
	skip.Feature(t, "versioning", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("ver")
	client.CreateBucket(t, bucket)
	enableVersioning(t, ctx, bucket)

	_, err := client.S3.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(bucket),
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: types.BucketVersioningStatusSuspended,
		},
	})
	require.NoError(t, err)

	out, err := client.S3.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)
	require.Equal(t, types.BucketVersioningStatusSuspended, out.Status)
}
