package objectlock_test

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

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

func createLockBucket(t *testing.T, ctx context.Context) string {
	t.Helper()
	bucket := client.RandBucket("lock")
	_, err := client.S3.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket:                     aws.String(bucket),
		ObjectLockEnabledForBucket: aws.Bool(true),
	})
	require.NoError(t, err)
	t.Cleanup(func() { client.DeleteBucketForce(ctx, bucket) })
	return bucket
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

func TestObjectLockConfigRoundTrip(t *testing.T) {
	skip.Feature(t, "object_lock", client.Target.Features)

	ctx := context.Background()
	bucket := createLockBucket(t, ctx)
	enableVersioning(t, ctx, bucket)

	days := int32(1)
	_, err := client.S3.PutObjectLockConfiguration(ctx, &s3.PutObjectLockConfigurationInput{
		Bucket: aws.String(bucket),
		ObjectLockConfiguration: &types.ObjectLockConfiguration{
			ObjectLockEnabled: types.ObjectLockEnabledEnabled,
			Rule: &types.ObjectLockRule{
				DefaultRetention: &types.DefaultRetention{
					Mode: types.ObjectLockRetentionModeGovernance,
					Days: &days,
				},
			},
		},
	})
	require.NoError(t, err)

	out, err := client.S3.GetObjectLockConfiguration(ctx, &s3.GetObjectLockConfigurationInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)
	require.NotNil(t, out.ObjectLockConfiguration)
	require.NotNil(t, out.ObjectLockConfiguration.Rule)
	require.Equal(t, types.ObjectLockRetentionModeGovernance, out.ObjectLockConfiguration.Rule.DefaultRetention.Mode)
	require.Equal(t, days, aws.ToInt32(out.ObjectLockConfiguration.Rule.DefaultRetention.Days))
}

func TestObjectRetentionGovernanceBlocks(t *testing.T) {
	skip.Feature(t, "object_lock", client.Target.Features)

	ctx := context.Background()
	bucket := createLockBucket(t, ctx)
	enableVersioning(t, ctx, bucket)

	retainUntil := time.Now().Add(24 * time.Hour)
	putOut, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:                    aws.String(bucket),
		Key:                       aws.String("locked.txt"),
		Body:                      bytes.NewReader([]byte("locked data")),
		ObjectLockMode:            types.ObjectLockModeGovernance,
		ObjectLockRetainUntilDate: &retainUntil,
	})
	require.NoError(t, err)

	_, err = client.S3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String("locked.txt"),
		VersionId: putOut.VersionId,
	})
	require.Error(t, err)
}

func TestLegalHoldBlocksDelete(t *testing.T) {
	skip.Feature(t, "object_lock", client.Target.Features)

	ctx := context.Background()
	bucket := createLockBucket(t, ctx)
	enableVersioning(t, ctx, bucket)

	putOut, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("held.txt"),
		Body:   bytes.NewReader([]byte("held data")),
	})
	require.NoError(t, err)

	_, err = client.S3.PutObjectLegalHold(ctx, &s3.PutObjectLegalHoldInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String("held.txt"),
		VersionId: putOut.VersionId,
		LegalHold: &types.ObjectLockLegalHold{
			Status: types.ObjectLockLegalHoldStatusOn,
		},
	})
	require.NoError(t, err)

	_, err = client.S3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String("held.txt"),
		VersionId: putOut.VersionId,
	})
	require.Error(t, err)
}

func TestLegalHoldCanBeReleased(t *testing.T) {
	skip.Feature(t, "object_lock", client.Target.Features)

	ctx := context.Background()
	bucket := createLockBucket(t, ctx)
	enableVersioning(t, ctx, bucket)

	putOut, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("held.txt"),
		Body:   bytes.NewReader([]byte("held data")),
	})
	require.NoError(t, err)

	// Put legal hold ON
	_, err = client.S3.PutObjectLegalHold(ctx, &s3.PutObjectLegalHoldInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String("held.txt"),
		VersionId: putOut.VersionId,
		LegalHold: &types.ObjectLockLegalHold{
			Status: types.ObjectLockLegalHoldStatusOn,
		},
	})
	require.NoError(t, err)

	// Release legal hold
	_, err = client.S3.PutObjectLegalHold(ctx, &s3.PutObjectLegalHoldInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String("held.txt"),
		VersionId: putOut.VersionId,
		LegalHold: &types.ObjectLockLegalHold{
			Status: types.ObjectLockLegalHoldStatusOff,
		},
	})
	require.NoError(t, err)

	_, err = client.S3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String("held.txt"),
		VersionId: putOut.VersionId,
	})
	require.NoError(t, err)
}

func TestGetObjectRetention(t *testing.T) {
	skip.Feature(t, "object_lock", client.Target.Features)

	ctx := context.Background()
	bucket := createLockBucket(t, ctx)
	enableVersioning(t, ctx, bucket)

	retainUntil := time.Now().Add(24 * time.Hour)
	putOut, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:                    aws.String(bucket),
		Key:                       aws.String("retained.txt"),
		Body:                      bytes.NewReader([]byte("retained data")),
		ObjectLockMode:            types.ObjectLockModeGovernance,
		ObjectLockRetainUntilDate: &retainUntil,
	})
	require.NoError(t, err)

	retOut, err := client.S3.GetObjectRetention(ctx, &s3.GetObjectRetentionInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String("retained.txt"),
		VersionId: putOut.VersionId,
	})
	require.NoError(t, err)
	require.NotNil(t, retOut.Retention)
	require.NotNil(t, retOut.Retention.RetainUntilDate)
}
