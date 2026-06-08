package auth_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
	"github.com/tamnd/s3-compat/compat/s3client"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
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

func buildClient(t *testing.T, accessKey, secretKey string) *s3.Client {
	t.Helper()
	httpC := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}} //nolint:gosec
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(client.Target.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		awsconfig.WithHTTPClient(httpC),
	)
	require.NoError(t, err)
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = client.Target.UsePathStyle
		if client.Target.Endpoint != "" {
			o.BaseEndpoint = aws.String(client.Target.Endpoint)
		}
	})
}

func TestValidCredentials(t *testing.T) {
	ctx := context.Background()
	_, err := client.S3.ListBuckets(ctx, nil)
	require.NoError(t, err)
}

func TestInvalidSecretKey(t *testing.T) {
	bad := buildClient(t, client.Target.AccessKey, "wrong-secret-key-xyz")
	_, err := bad.ListBuckets(context.Background(), nil)
	require.Error(t, err)
}

func TestInvalidAccessKeyID(t *testing.T) {
	bad := buildClient(t, "FAKEID12345", "fakesecret")
	_, err := bad.ListBuckets(context.Background(), nil)
	require.Error(t, err)
}

func TestPresignedGetURL(t *testing.T) {
	bucket := client.RandBucket("auth")
	client.CreateBucket(t, bucket)
	client.PutObject(t, bucket, "hello.txt", []byte("hello presign"))

	presignClient := s3.NewPresignClient(client.S3)
	req, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("hello.txt"),
	}, s3.WithPresignExpires(10*time.Minute))
	require.NoError(t, err)

	resp, err := http.Get(req.URL) //nolint:noctx
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	require.Equal(t, []byte("hello presign"), body)
}

func TestPresignedPutURL(t *testing.T) {
	bucket := client.RandBucket("auth")
	client.CreateBucket(t, bucket)

	presignClient := s3.NewPresignClient(client.S3)
	req, err := presignClient.PresignPutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("presigned.txt"),
	}, s3.WithPresignExpires(10*time.Minute))
	require.NoError(t, err)

	putBody := strings.NewReader("presign-put-body")
	hr, err := http.NewRequest(http.MethodPut, req.URL, putBody)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(hr)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.True(t, resp.StatusCode < 300, "presigned PUT status: "+fmt.Sprint(resp.StatusCode))

	body := client.GetObjectBody(t, bucket, "presigned.txt")
	require.Equal(t, []byte("presign-put-body"), body)
}

func TestPresignedURLExpired(t *testing.T) {
	bucket := client.RandBucket("auth")
	client.CreateBucket(t, bucket)
	client.PutObject(t, bucket, "exp.txt", []byte("expiry test"))

	presignClient := s3.NewPresignClient(client.S3)
	req, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("exp.txt"),
	}, s3.WithPresignExpires(1*time.Second))
	require.NoError(t, err)

	time.Sleep(3 * time.Second)

	resp, err := http.Get(req.URL) //nolint:noctx
	if err == nil {
		defer resp.Body.Close()
		require.NotEqual(t, 200, resp.StatusCode, "expired presigned URL should not return 200")
	}
}
