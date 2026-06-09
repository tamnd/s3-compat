package buckets_test

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/require"
	s3assert "github.com/tamnd/s3-compat/compat/assert"
	"github.com/tamnd/s3-compat/compat/s3client"
	"github.com/tamnd/s3-compat/compat/skip"
)

var client *s3client.Client

func TestMain(m *testing.M) {
	c, err := s3client.Load("")
	if err != nil {
		panic("s3compat setup: " + err.Error())
	}
	client = c
	os.Exit(m.Run())
}

func TestCreateBucketAndDelete(t *testing.T) {
	bucket := client.RandBucket("buckets")
	ctx := context.Background()
	_, err := client.S3.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	require.NoError(t, err)
	defer client.DeleteBucketForce(ctx, bucket)
	_, err = client.S3.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
	require.NoError(t, err)
	_, err = client.S3.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	require.NoError(t, err)
	_, err = client.S3.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
	require.Error(t, err)
}

func TestCreateBucketIdempotent(t *testing.T) {
	bucket := client.RandBucket("buckets")
	ctx := context.Background()
	_, err := client.S3.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	require.NoError(t, err)
	defer client.DeleteBucketForce(ctx, bucket)
	_, err = client.S3.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	// S3 returns 200 for idempotent create; some targets return BucketAlreadyOwnedByYou
	if err != nil {
		var already *types.BucketAlreadyOwnedByYou
		require.ErrorAs(t, err, &already, "second CreateBucket should be idempotent or BucketAlreadyOwnedByYou")
	}
}

func TestDeleteNonEmptyBucket(t *testing.T) {
	bucket := client.RandBucket("buckets")
	client.CreateBucket(t, bucket)
	client.PutObject(t, bucket, "key.txt", []byte("data"))
	_, err := client.S3.DeleteBucket(context.Background(), &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	s3assert.BucketNotEmpty(t, err)
}

func TestHeadBucketNotFound(t *testing.T) {
	_, err := client.S3.HeadBucket(context.Background(), &s3.HeadBucketInput{
		Bucket: aws.String("s3compat-does-not-exist-" + "xyzzy999"),
	})
	require.Error(t, err)
}

func TestListBucketsContainsCreated(t *testing.T) {
	bucket := client.RandBucket("buckets")
	client.CreateBucket(t, bucket)
	out, err := client.S3.ListBuckets(context.Background(), nil)
	require.NoError(t, err)
	var found bool
	for _, b := range out.Buckets {
		if aws.ToString(b.Name) == bucket {
			found = true
			break
		}
	}
	require.True(t, found, "bucket %s not in ListBuckets response", bucket)
}

func TestGetBucketLocation(t *testing.T) {
	bucket := client.RandBucket("buckets")
	client.CreateBucket(t, bucket)
	out, err := client.S3.GetBucketLocation(context.Background(), &s3.GetBucketLocationInput{Bucket: aws.String(bucket)})
	require.NoError(t, err)
	_ = out.LocationConstraint // may be "" (us-east-1) or a region string
}

func TestBucketTagging(t *testing.T) {
	skip.Feature(t, "tagging", client.Target.Features)
	bucket := client.RandBucket("buckets")
	client.CreateBucket(t, bucket)
	ctx := context.Background()
	_, err := client.S3.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{
		Bucket: aws.String(bucket),
		Tagging: &types.Tagging{TagSet: []types.Tag{
			{Key: aws.String("env"), Value: aws.String("test")},
			{Key: aws.String("owner"), Value: aws.String("compat")},
		}},
	})
	require.NoError(t, err)
	out, err := client.S3.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{Bucket: aws.String(bucket)})
	require.NoError(t, err)
	require.Len(t, out.TagSet, 2)
	_, err = client.S3.DeleteBucketTagging(ctx, &s3.DeleteBucketTaggingInput{Bucket: aws.String(bucket)})
	require.NoError(t, err)
}

func TestVersioningEnableAndGet(t *testing.T) {
	skip.Feature(t, "versioning", client.Target.Features)
	bucket := client.RandBucket("buckets")
	client.CreateBucket(t, bucket)
	ctx := context.Background()
	_, err := client.S3.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket:                  aws.String(bucket),
		VersioningConfiguration: &types.VersioningConfiguration{Status: types.BucketVersioningStatusEnabled},
	})
	require.NoError(t, err)
	out, err := client.S3.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{Bucket: aws.String(bucket)})
	require.NoError(t, err)
	require.Equal(t, types.BucketVersioningStatusEnabled, out.Status)
}

func TestVersioningSuspend(t *testing.T) {
	skip.Feature(t, "versioning", client.Target.Features)
	bucket := client.RandBucket("buckets")
	client.CreateBucket(t, bucket)
	ctx := context.Background()
	_, err := client.S3.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket:                  aws.String(bucket),
		VersioningConfiguration: &types.VersioningConfiguration{Status: types.BucketVersioningStatusEnabled},
	})
	require.NoError(t, err)
	_, err = client.S3.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket:                  aws.String(bucket),
		VersioningConfiguration: &types.VersioningConfiguration{Status: types.BucketVersioningStatusSuspended},
	})
	require.NoError(t, err)
	out, err := client.S3.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{Bucket: aws.String(bucket)})
	require.NoError(t, err)
	require.Equal(t, types.BucketVersioningStatusSuspended, out.Status)
}
