package tagging_test

import (
	"context"
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

func TestObjectTaggingRoundTrip(t *testing.T) {
	skip.Feature(t, "tagging", client.Target.Features)

	bucket := client.RandBucket("tag")
	client.CreateBucket(t, bucket)
	client.PutObject(t, bucket, "obj.txt", []byte("data"))
	ctx := context.Background()

	_, err := client.S3.PutObjectTagging(ctx, &s3.PutObjectTaggingInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("obj.txt"),
		Tagging: &types.Tagging{
			TagSet: []types.Tag{
				{Key: aws.String("k1"), Value: aws.String("v1")},
				{Key: aws.String("k2"), Value: aws.String("v2")},
				{Key: aws.String("k3"), Value: aws.String("v3")},
			},
		},
	})
	require.NoError(t, err)

	out, err := client.S3.GetObjectTagging(ctx, &s3.GetObjectTaggingInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("obj.txt"),
	})
	require.NoError(t, err)
	require.Len(t, out.TagSet, 3)
}

func TestObjectTaggingDelete(t *testing.T) {
	skip.Feature(t, "tagging", client.Target.Features)

	bucket := client.RandBucket("tag")
	client.CreateBucket(t, bucket)
	client.PutObject(t, bucket, "del.txt", []byte("data"))
	ctx := context.Background()

	_, err := client.S3.PutObjectTagging(ctx, &s3.PutObjectTaggingInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("del.txt"),
		Tagging: &types.Tagging{
			TagSet: []types.Tag{
				{Key: aws.String("a"), Value: aws.String("1")},
				{Key: aws.String("b"), Value: aws.String("2")},
			},
		},
	})
	require.NoError(t, err)

	_, err = client.S3.DeleteObjectTagging(ctx, &s3.DeleteObjectTaggingInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("del.txt"),
	})
	require.NoError(t, err)

	out, err := client.S3.GetObjectTagging(ctx, &s3.GetObjectTaggingInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("del.txt"),
	})
	require.NoError(t, err)
	require.Len(t, out.TagSet, 0)
}

func TestBucketTaggingRoundTrip(t *testing.T) {
	skip.Feature(t, "tagging", client.Target.Features)

	bucket := client.RandBucket("tag")
	client.CreateBucket(t, bucket)
	ctx := context.Background()

	_, err := client.S3.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{
		Bucket: aws.String(bucket),
		Tagging: &types.Tagging{
			TagSet: []types.Tag{
				{Key: aws.String("env"), Value: aws.String("test")},
				{Key: aws.String("team"), Value: aws.String("infra")},
			},
		},
	})
	require.NoError(t, err)

	out, err := client.S3.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)
	require.Len(t, out.TagSet, 2)

	keys := make(map[string]bool)
	for _, tag := range out.TagSet {
		keys[aws.ToString(tag.Key)] = true
	}
	require.True(t, keys["env"], "tag key 'env' not found")
	require.True(t, keys["team"], "tag key 'team' not found")
}

func TestBucketTaggingDelete(t *testing.T) {
	skip.Feature(t, "tagging", client.Target.Features)

	bucket := client.RandBucket("tag")
	client.CreateBucket(t, bucket)
	ctx := context.Background()

	_, err := client.S3.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{
		Bucket: aws.String(bucket),
		Tagging: &types.Tagging{
			TagSet: []types.Tag{
				{Key: aws.String("x"), Value: aws.String("y")},
			},
		},
	})
	require.NoError(t, err)

	_, err = client.S3.DeleteBucketTagging(ctx, &s3.DeleteBucketTaggingInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)

	out, err := client.S3.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucket),
	})
	if err == nil {
		require.Len(t, out.TagSet, 0)
	}
}
