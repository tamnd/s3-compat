package multipart_test

import (
	"bytes"
	"context"
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

func TestMultipartComplete(t *testing.T) {
	skip.Feature(t, "multipart", client.Target.Features)

	bucket := client.RandBucket("mp")
	client.CreateBucket(t, bucket)
	ctx := context.Background()

	init, err := client.S3.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("big.bin"),
	})
	require.NoError(t, err)
	uploadID := init.UploadId

	partSize := 5*1024*1024 + 1
	var completedParts []types.CompletedPart
	for i := 1; i <= 3; i++ {
		body := bytes.NewReader(bytes.Repeat([]byte("A"), partSize))
		up, err := client.S3.UploadPart(ctx, &s3.UploadPartInput{
			Bucket:     aws.String(bucket),
			Key:        aws.String("big.bin"),
			PartNumber: aws.Int32(int32(i)),
			UploadId:   uploadID,
			Body:       body,
		})
		require.NoError(t, err)
		completedParts = append(completedParts, types.CompletedPart{
			ETag:       up.ETag,
			PartNumber: aws.Int32(int32(i)),
		})
	}

	_, err = client.S3.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String("big.bin"),
		UploadId: uploadID,
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	require.NoError(t, err)

	head, err := client.S3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("big.bin"),
	})
	require.NoError(t, err)
	require.Equal(t, int64(3*partSize), aws.ToInt64(head.ContentLength))
}

func TestMultipartAbort(t *testing.T) {
	skip.Feature(t, "multipart", client.Target.Features)

	bucket := client.RandBucket("mp")
	client.CreateBucket(t, bucket)
	ctx := context.Background()

	init, err := client.S3.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("abort.bin"),
	})
	require.NoError(t, err)
	uploadID := init.UploadId

	_, err = client.S3.UploadPart(ctx, &s3.UploadPartInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String("abort.bin"),
		PartNumber: aws.Int32(1),
		UploadId:   uploadID,
		Body:       bytes.NewReader([]byte("small part body")),
	})
	require.NoError(t, err)

	_, err = client.S3.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String("abort.bin"),
		UploadId: uploadID,
	})
	require.NoError(t, err)

	list, err := client.S3.ListMultipartUploads(ctx, &s3.ListMultipartUploadsInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)
	for _, u := range list.Uploads {
		require.NotEqual(t, aws.ToString(uploadID), aws.ToString(u.UploadId),
			"aborted upload ID should not appear in ListMultipartUploads")
	}
}

func TestListMultipartUploads(t *testing.T) {
	skip.Feature(t, "multipart", client.Target.Features)
	skip.Feature(t, "list_multipart", client.Target.Features)

	bucket := client.RandBucket("mp")
	client.CreateBucket(t, bucket)
	ctx := context.Background()

	init1, err := client.S3.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("file1.bin"),
	})
	require.NoError(t, err)

	init2, err := client.S3.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("file2.bin"),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = client.S3.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
			Bucket:   aws.String(bucket),
			Key:      aws.String("file1.bin"),
			UploadId: init1.UploadId,
		})
		_, _ = client.S3.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
			Bucket:   aws.String(bucket),
			Key:      aws.String("file2.bin"),
			UploadId: init2.UploadId,
		})
	})

	list, err := client.S3.ListMultipartUploads(ctx, &s3.ListMultipartUploadsInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)

	found := make(map[string]bool)
	for _, u := range list.Uploads {
		found[aws.ToString(u.UploadId)] = true
	}
	require.True(t, found[aws.ToString(init1.UploadId)], "upload ID 1 not found in list")
	require.True(t, found[aws.ToString(init2.UploadId)], "upload ID 2 not found in list")
}

func TestListParts(t *testing.T) {
	skip.Feature(t, "multipart", client.Target.Features)

	bucket := client.RandBucket("mp")
	client.CreateBucket(t, bucket)
	ctx := context.Background()

	init, err := client.S3.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("parts.bin"),
	})
	require.NoError(t, err)
	uploadID := init.UploadId

	t.Cleanup(func() {
		_, _ = client.S3.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
			Bucket:   aws.String(bucket),
			Key:      aws.String("parts.bin"),
			UploadId: uploadID,
		})
	})

	partSize := 5*1024*1024 + 1
	for _, pn := range []int32{1, 2} {
		_, err = client.S3.UploadPart(ctx, &s3.UploadPartInput{
			Bucket:     aws.String(bucket),
			Key:        aws.String("parts.bin"),
			PartNumber: aws.Int32(pn),
			UploadId:   uploadID,
			Body:       bytes.NewReader(bytes.Repeat([]byte("B"), partSize)),
		})
		require.NoError(t, err)
	}

	lp, err := client.S3.ListParts(ctx, &s3.ListPartsInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String("parts.bin"),
		UploadId: uploadID,
	})
	require.NoError(t, err)
	require.Len(t, lp.Parts, 2)

	partNums := make(map[int32]bool)
	for _, p := range lp.Parts {
		partNums[aws.ToInt32(p.PartNumber)] = true
	}
	require.True(t, partNums[1], "part 1 not found")
	require.True(t, partNums[2], "part 2 not found")
}

func TestMultipartNoSuchUpload(t *testing.T) {
	skip.Feature(t, "multipart", client.Target.Features)

	bucket := client.RandBucket("mp")
	client.CreateBucket(t, bucket)
	ctx := context.Background()

	_, err := client.S3.UploadPart(ctx, &s3.UploadPartInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String("fake.bin"),
		PartNumber: aws.Int32(1),
		UploadId:   aws.String("fakeid-that-does-not-exist-xyz"),
		Body:       bytes.NewReader([]byte("data")),
	})
	require.Error(t, err)
}

func TestMultipartETagFormat(t *testing.T) {
	skip.Feature(t, "multipart", client.Target.Features)

	bucket := client.RandBucket("mp")
	client.CreateBucket(t, bucket)
	ctx := context.Background()

	init, err := client.S3.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("etag.bin"),
	})
	require.NoError(t, err)
	uploadID := init.UploadId

	partSize := 5*1024*1024 + 1
	var completedParts []types.CompletedPart
	for i := 1; i <= 2; i++ {
		body := bytes.NewReader(bytes.Repeat([]byte("C"), partSize))
		up, err := client.S3.UploadPart(ctx, &s3.UploadPartInput{
			Bucket:     aws.String(bucket),
			Key:        aws.String("etag.bin"),
			PartNumber: aws.Int32(int32(i)),
			UploadId:   uploadID,
			Body:       body,
		})
		require.NoError(t, err)
		completedParts = append(completedParts, types.CompletedPart{
			ETag:       up.ETag,
			PartNumber: aws.Int32(int32(i)),
		})
	}

	_, err = client.S3.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String("etag.bin"),
		UploadId: uploadID,
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	require.NoError(t, err)

	head, err := client.S3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("etag.bin"),
	})
	require.NoError(t, err)
	etag := strings.Trim(aws.ToString(head.ETag), `"`)
	require.True(t, strings.HasSuffix(etag, "-2"),
		"expected ETag to end in -2 for 2-part upload, got %q", etag)
}
