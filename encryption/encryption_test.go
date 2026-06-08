package encryption_test

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
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

func genSSECKey() (keyB64, md5B64 string) {
	var key [32]byte
	_, _ = rand.Read(key[:])
	sum := md5.Sum(key[:])
	return base64.StdEncoding.EncodeToString(key[:]), base64.StdEncoding.EncodeToString(sum[:])
}

func TestSSES3RoundTrip(t *testing.T) {
	skip.Feature(t, "encryption_sse_s3", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("enc")
	client.CreateBucket(t, bucket)

	_, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:               aws.String(bucket),
		Key:                  aws.String("encrypted.txt"),
		Body:                 bytes.NewReader([]byte("sse-s3 data")),
		ServerSideEncryption: types.ServerSideEncryptionAes256,
	})
	require.NoError(t, err)

	headOut, err := client.S3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("encrypted.txt"),
	})
	require.NoError(t, err)
	require.Equal(t, types.ServerSideEncryptionAes256, headOut.ServerSideEncryption)
}

func TestSSECRoundTrip(t *testing.T) {
	skip.Feature(t, "encryption_sse_c", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("enc")
	client.CreateBucket(t, bucket)

	keyB64, md5B64 := genSSECKey()

	_, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:               aws.String(bucket),
		Key:                  aws.String("ssec.txt"),
		Body:                 bytes.NewReader([]byte("secret data")),
		SSECustomerAlgorithm: aws.String("AES256"),
		SSECustomerKey:       aws.String(keyB64),
		SSECustomerKeyMD5:    aws.String(md5B64),
	})
	require.NoError(t, err)

	getOut, err := client.S3.GetObject(ctx, &s3.GetObjectInput{
		Bucket:               aws.String(bucket),
		Key:                  aws.String("ssec.txt"),
		SSECustomerAlgorithm: aws.String("AES256"),
		SSECustomerKey:       aws.String(keyB64),
		SSECustomerKeyMD5:    aws.String(md5B64),
	})
	require.NoError(t, err)
	defer getOut.Body.Close()

	body, err := io.ReadAll(getOut.Body)
	require.NoError(t, err)
	require.Equal(t, "secret data", string(body))
}

func TestSSECWrongKey(t *testing.T) {
	skip.Feature(t, "encryption_sse_c", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("enc")
	client.CreateBucket(t, bucket)

	keyB64A, md5B64A := genSSECKey()
	keyB64B, md5B64B := genSSECKey()

	_, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:               aws.String(bucket),
		Key:                  aws.String("ssec-wrong.txt"),
		Body:                 bytes.NewReader([]byte("secret data")),
		SSECustomerAlgorithm: aws.String("AES256"),
		SSECustomerKey:       aws.String(keyB64A),
		SSECustomerKeyMD5:    aws.String(md5B64A),
	})
	require.NoError(t, err)

	_, err = client.S3.GetObject(ctx, &s3.GetObjectInput{
		Bucket:               aws.String(bucket),
		Key:                  aws.String("ssec-wrong.txt"),
		SSECustomerAlgorithm: aws.String("AES256"),
		SSECustomerKey:       aws.String(keyB64B),
		SSECustomerKeyMD5:    aws.String(md5B64B),
	})
	require.Error(t, err)
}
