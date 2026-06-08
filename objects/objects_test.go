package objects_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
	s3assert "github.com/tamnd/s3-compat/compat/assert"
	"github.com/tamnd/s3-compat/compat/s3client"
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

func TestPutGetRoundTrip(t *testing.T) {
	bucket := client.RandBucket("objects")
	client.CreateBucket(t, bucket)
	ctx := context.Background()
	body := []byte("hello world")
	_, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("roundtrip.txt"),
		Body:   bytes.NewReader(body),
	})
	require.NoError(t, err)
	out, err := client.S3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("roundtrip.txt"),
	})
	require.NoError(t, err)
	defer out.Body.Close()
	got, err := io.ReadAll(out.Body)
	require.NoError(t, err)
	require.Equal(t, body, got)
	require.NotEmpty(t, aws.ToString(out.ETag))
}

func TestPutGetEmptyObject(t *testing.T) {
	bucket := client.RandBucket("objects")
	client.CreateBucket(t, bucket)
	ctx := context.Background()
	_, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("empty.txt"),
		Body:   bytes.NewReader([]byte{}),
	})
	require.NoError(t, err)
	out, err := client.S3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("empty.txt"),
	})
	require.NoError(t, err)
	defer out.Body.Close()
	got, err := io.ReadAll(out.Body)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestHeadObject(t *testing.T) {
	bucket := client.RandBucket("objects")
	client.CreateBucket(t, bucket)
	ctx := context.Background()
	body := []byte("hello world")
	_, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("head.txt"),
		Body:   bytes.NewReader(body),
	})
	require.NoError(t, err)
	out, err := client.S3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("head.txt"),
	})
	require.NoError(t, err)
	require.EqualValues(t, int64(len(body)), aws.ToInt64(out.ContentLength))
	require.NotEmpty(t, aws.ToString(out.ETag))
}

func TestObjectMetadata(t *testing.T) {
	bucket := client.RandBucket("objects")
	client.CreateBucket(t, bucket)
	ctx := context.Background()
	_, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String("meta.txt"),
		Body:     bytes.NewReader([]byte("data")),
		Metadata: map[string]string{"purpose": "testval"},
	})
	require.NoError(t, err)
	out, err := client.S3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("meta.txt"),
	})
	require.NoError(t, err)
	require.Equal(t, "testval", out.Metadata["purpose"])
}

func TestObjectContentType(t *testing.T) {
	bucket := client.RandBucket("objects")
	client.CreateBucket(t, bucket)
	ctx := context.Background()
	_, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String("ct.json"),
		Body:        bytes.NewReader([]byte("{}")),
		ContentType: aws.String("application/json"),
	})
	require.NoError(t, err)
	out, err := client.S3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("ct.json"),
	})
	require.NoError(t, err)
	require.True(t, strings.Contains(aws.ToString(out.ContentType), "application/json"),
		"expected ContentType to contain application/json, got %q", aws.ToString(out.ContentType))
}

func TestGetObjectNotFound(t *testing.T) {
	bucket := client.RandBucket("objects")
	client.CreateBucket(t, bucket)
	_, err := client.S3.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("does-not-exist.txt"),
	})
	s3assert.NoSuchKey(t, err)
}

func TestDeleteObject(t *testing.T) {
	bucket := client.RandBucket("objects")
	client.CreateBucket(t, bucket)
	ctx := context.Background()
	client.PutObject(t, bucket, "todelete.txt", []byte("bye"))
	_, err := client.S3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("todelete.txt"),
	})
	require.NoError(t, err)
	_, err = client.S3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("todelete.txt"),
	})
	s3assert.NoSuchKey(t, err)
}

func TestDeleteNonExistentObject(t *testing.T) {
	bucket := client.RandBucket("objects")
	client.CreateBucket(t, bucket)
	_, err := client.S3.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("no-such-key.txt"),
	})
	require.NoError(t, err)
}

func TestCopyObject(t *testing.T) {
	bucket := client.RandBucket("objects")
	client.CreateBucket(t, bucket)
	ctx := context.Background()
	body := []byte("copy me")
	client.PutObject(t, bucket, "srckey", body)
	_, err := client.S3.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String("dstkey"),
		CopySource: aws.String(bucket + "/srckey"),
	})
	require.NoError(t, err)
	got := client.GetObjectBody(t, bucket, "dstkey")
	require.Equal(t, body, got)
}

func TestListObjectsV2Basic(t *testing.T) {
	bucket := client.RandBucket("objects")
	client.CreateBucket(t, bucket)
	keys := []string{"obj1.txt", "obj2.txt", "obj3.txt"}
	for _, k := range keys {
		client.PutObject(t, bucket, k, []byte("data"))
	}
	out, err := client.S3.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)
	found := make(map[string]bool)
	for _, obj := range out.Contents {
		found[aws.ToString(obj.Key)] = true
	}
	for _, k := range keys {
		require.True(t, found[k], "key %q not found in listing", k)
	}
}

func TestListObjectsV2Prefix(t *testing.T) {
	bucket := client.RandBucket("objects")
	client.CreateBucket(t, bucket)
	client.PutObject(t, bucket, "pfx/a", []byte("a"))
	client.PutObject(t, bucket, "pfx/b", []byte("b"))
	client.PutObject(t, bucket, "other", []byte("other"))
	out, err := client.S3.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String("pfx/"),
	})
	require.NoError(t, err)
	require.Len(t, out.Contents, 2)
}

func TestListObjectsV2Delimiter(t *testing.T) {
	bucket := client.RandBucket("objects")
	client.CreateBucket(t, bucket)
	client.PutObject(t, bucket, "a/b", []byte("ab"))
	client.PutObject(t, bucket, "a/c", []byte("ac"))
	client.PutObject(t, bucket, "d", []byte("d"))
	out, err := client.S3.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Delimiter: aws.String("/"),
	})
	require.NoError(t, err)
	var foundD bool
	for _, obj := range out.Contents {
		if aws.ToString(obj.Key) == "d" {
			foundD = true
		}
	}
	require.True(t, foundD, "key 'd' not found in Contents")
	var foundPrefix bool
	for _, cp := range out.CommonPrefixes {
		if aws.ToString(cp.Prefix) == "a/" {
			foundPrefix = true
		}
	}
	require.True(t, foundPrefix, "common prefix 'a/' not found")
}

func TestListObjectsV2Pagination(t *testing.T) {
	bucket := client.RandBucket("objects")
	client.CreateBucket(t, bucket)
	keys := []string{"p1", "p2", "p3", "p4", "p5"}
	for _, k := range keys {
		client.PutObject(t, bucket, k, []byte("x"))
	}
	found := make(map[string]bool)
	var token *string
	for {
		out, err := client.S3.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
			Bucket:            aws.String(bucket),
			MaxKeys:           aws.Int32(2),
			ContinuationToken: token,
		})
		require.NoError(t, err)
		for _, obj := range out.Contents {
			found[aws.ToString(obj.Key)] = true
		}
		if !aws.ToBool(out.IsTruncated) {
			break
		}
		token = out.NextContinuationToken
	}
	for _, k := range keys {
		require.True(t, found[k], "key %q not found after paginated listing", k)
	}
}

func TestGetObjectRangeHeader(t *testing.T) {
	bucket := client.RandBucket("objects")
	client.CreateBucket(t, bucket)
	body := bytes.Repeat([]byte("A"), 100)
	client.PutObject(t, bucket, "ranged.bin", body)
	out, err := client.S3.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("ranged.bin"),
		Range:  aws.String("bytes=0-9"),
	})
	require.NoError(t, err)
	defer out.Body.Close()
	got, err := io.ReadAll(out.Body)
	require.NoError(t, err)
	require.Len(t, got, 10)
}

func TestPutLargeObject(t *testing.T) {
	bucket := client.RandBucket("objects")
	client.CreateBucket(t, bucket)
	const size = 8 * 1024 * 1024
	body := bytes.Repeat([]byte("X"), size)
	ctx := context.Background()
	_, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("large.bin"),
		Body:   bytes.NewReader(body),
	})
	require.NoError(t, err)
	out, err := client.S3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("large.bin"),
	})
	require.NoError(t, err)
	require.EqualValues(t, int64(size), aws.ToInt64(out.ContentLength))
}
