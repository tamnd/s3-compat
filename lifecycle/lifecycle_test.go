package lifecycle_test

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

func TestLifecyclePutAndGet(t *testing.T) {
	skip.Feature(t, "lifecycle", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("lc")
	client.CreateBucket(t, bucket)

	days := int32(30)
	_, err := client.S3.PutBucketLifecycleConfiguration(ctx, &s3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(bucket),
		LifecycleConfiguration: &types.BucketLifecycleConfiguration{
			Rules: []types.LifecycleRule{
				{
					ID:     aws.String("expire-logs"),
					Status: types.ExpirationStatusEnabled,
					Filter: &types.LifecycleRuleFilter{Prefix: aws.String("logs/")},
					Expiration: &types.LifecycleExpiration{
						Days: &days,
					},
				},
			},
		},
	})
	require.NoError(t, err)

	out, err := client.S3.GetBucketLifecycleConfiguration(ctx, &s3.GetBucketLifecycleConfigurationInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)
	require.Len(t, out.Rules, 1)
	require.Equal(t, "expire-logs", aws.ToString(out.Rules[0].ID))
	require.Equal(t, days, aws.ToInt32(out.Rules[0].Expiration.Days))
}

func TestLifecycleDelete(t *testing.T) {
	skip.Feature(t, "lifecycle", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("lc")
	client.CreateBucket(t, bucket)

	days := int32(30)
	_, err := client.S3.PutBucketLifecycleConfiguration(ctx, &s3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(bucket),
		LifecycleConfiguration: &types.BucketLifecycleConfiguration{
			Rules: []types.LifecycleRule{
				{
					ID:     aws.String("expire-logs"),
					Status: types.ExpirationStatusEnabled,
					Filter: &types.LifecycleRuleFilter{Prefix: aws.String("logs/")},
					Expiration: &types.LifecycleExpiration{
						Days: &days,
					},
				},
			},
		},
	})
	require.NoError(t, err)

	_, err = client.S3.DeleteBucketLifecycle(ctx, &s3.DeleteBucketLifecycleInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)

	_, err = client.S3.GetBucketLifecycleConfiguration(ctx, &s3.GetBucketLifecycleConfigurationInput{
		Bucket: aws.String(bucket),
	})
	require.Error(t, err)
}

func TestLifecycleMultipleRules(t *testing.T) {
	skip.Feature(t, "lifecycle", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("lc")
	client.CreateBucket(t, bucket)

	days1 := int32(30)
	days2 := int32(90)
	_, err := client.S3.PutBucketLifecycleConfiguration(ctx, &s3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(bucket),
		LifecycleConfiguration: &types.BucketLifecycleConfiguration{
			Rules: []types.LifecycleRule{
				{
					ID:     aws.String("rule-one"),
					Status: types.ExpirationStatusEnabled,
					Filter: &types.LifecycleRuleFilter{Prefix: aws.String("logs/")},
					Expiration: &types.LifecycleExpiration{
						Days: &days1,
					},
				},
				{
					ID:     aws.String("rule-two"),
					Status: types.ExpirationStatusEnabled,
					Filter: &types.LifecycleRuleFilter{Prefix: aws.String("tmp/")},
					Expiration: &types.LifecycleExpiration{
						Days: &days2,
					},
				},
			},
		},
	})
	require.NoError(t, err)

	out, err := client.S3.GetBucketLifecycleConfiguration(ctx, &s3.GetBucketLifecycleConfigurationInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)
	require.Len(t, out.Rules, 2)

	ids := map[string]bool{}
	for _, r := range out.Rules {
		ids[aws.ToString(r.ID)] = true
	}
	require.True(t, ids["rule-one"])
	require.True(t, ids["rule-two"])
}
