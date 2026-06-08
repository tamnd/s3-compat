package notifications_test

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

func TestNotificationConfigRoundTrip(t *testing.T) {
	skip.Feature(t, "notifications", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("notif")
	client.CreateBucket(t, bucket)

	_, err := client.S3.PutBucketNotificationConfiguration(ctx, &s3.PutBucketNotificationConfigurationInput{
		Bucket: aws.String(bucket),
		NotificationConfiguration: &types.NotificationConfiguration{
			QueueConfigurations: []types.QueueConfiguration{
				{
					Id:       aws.String("test-notif"),
					QueueArn: aws.String("arn:aws:sqs:us-east-1:123456789012:test-queue"),
					Events:   []types.Event{"s3:ObjectCreated:*"},
				},
			},
		},
	})
	require.NoError(t, err)

	out, err := client.S3.GetBucketNotificationConfiguration(ctx, &s3.GetBucketNotificationConfigurationInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)
	require.NotEmpty(t, out.QueueConfigurations)
}

func TestNotificationConfigClear(t *testing.T) {
	skip.Feature(t, "notifications", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("notif")
	client.CreateBucket(t, bucket)

	// Put initial config
	_, err := client.S3.PutBucketNotificationConfiguration(ctx, &s3.PutBucketNotificationConfigurationInput{
		Bucket: aws.String(bucket),
		NotificationConfiguration: &types.NotificationConfiguration{
			QueueConfigurations: []types.QueueConfiguration{
				{
					Id:       aws.String("test-notif"),
					QueueArn: aws.String("arn:aws:sqs:us-east-1:123456789012:test-queue"),
					Events:   []types.Event{"s3:ObjectCreated:*"},
				},
			},
		},
	})
	require.NoError(t, err)

	// Clear config with empty notification configuration
	_, err = client.S3.PutBucketNotificationConfiguration(ctx, &s3.PutBucketNotificationConfigurationInput{
		Bucket:                    aws.String(bucket),
		NotificationConfiguration: &types.NotificationConfiguration{},
	})
	require.NoError(t, err)

	out, err := client.S3.GetBucketNotificationConfiguration(ctx, &s3.GetBucketNotificationConfigurationInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)
	require.Empty(t, out.QueueConfigurations)
}
