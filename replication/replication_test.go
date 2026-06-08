package replication_test

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

func TestReplicationConfigRoundTrip(t *testing.T) {
	skip.Feature(t, "replication", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("repl")
	client.CreateBucket(t, bucket)
	enableVersioning(t, ctx, bucket)

	_, err := client.S3.PutBucketReplication(ctx, &s3.PutBucketReplicationInput{
		Bucket: aws.String(bucket),
		ReplicationConfiguration: &types.ReplicationConfiguration{
			Role: aws.String("arn:aws:iam::123456789012:role/replication-role"),
			Rules: []types.ReplicationRule{
				{
					ID:     aws.String("rule1"),
					Status: types.ReplicationRuleStatusEnabled,
					Destination: &types.Destination{
						Bucket: aws.String("arn:aws:s3:::dest-bucket"),
					},
					Filter: &types.ReplicationRuleFilter{Prefix: aws.String("")},
				},
			},
		},
	})
	require.NoError(t, err)

	out, err := client.S3.GetBucketReplication(ctx, &s3.GetBucketReplicationInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)
	require.NotNil(t, out.ReplicationConfiguration)
	require.Len(t, out.ReplicationConfiguration.Rules, 1)
	require.Equal(t, "rule1", aws.ToString(out.ReplicationConfiguration.Rules[0].ID))
}

func TestReplicationConfigDelete(t *testing.T) {
	skip.Feature(t, "replication", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("repl")
	client.CreateBucket(t, bucket)
	enableVersioning(t, ctx, bucket)

	_, err := client.S3.PutBucketReplication(ctx, &s3.PutBucketReplicationInput{
		Bucket: aws.String(bucket),
		ReplicationConfiguration: &types.ReplicationConfiguration{
			Role: aws.String("arn:aws:iam::123456789012:role/replication-role"),
			Rules: []types.ReplicationRule{
				{
					ID:     aws.String("rule1"),
					Status: types.ReplicationRuleStatusEnabled,
					Destination: &types.Destination{
						Bucket: aws.String("arn:aws:s3:::dest-bucket"),
					},
					Filter: &types.ReplicationRuleFilter{Prefix: aws.String("")},
				},
			},
		},
	})
	require.NoError(t, err)

	_, err = client.S3.DeleteBucketReplication(ctx, &s3.DeleteBucketReplicationInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)

	_, err = client.S3.GetBucketReplication(ctx, &s3.GetBucketReplicationInput{
		Bucket: aws.String(bucket),
	})
	require.Error(t, err)
}
