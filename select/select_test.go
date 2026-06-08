package select_test

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

func TestSelectCSV(t *testing.T) {
	skip.Feature(t, "select", client.Target.Features)

	ctx := context.Background()
	bucket := client.RandBucket("sel")
	client.CreateBucket(t, bucket)

	csvData := "id,name\n1,alice\n2,bob\n"
	_, err := client.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("data.csv"),
		Body:   bytes.NewReader([]byte(csvData)),
	})
	require.NoError(t, err)

	out, err := client.S3.SelectObjectContent(ctx, &s3.SelectObjectContentInput{
		Bucket:         aws.String(bucket),
		Key:            aws.String("data.csv"),
		Expression:     aws.String("SELECT * FROM S3Object"),
		ExpressionType: types.ExpressionTypeSql,
		InputSerialization: &types.InputSerialization{
			CSV: &types.CSVInput{
				FileHeaderInfo: types.FileHeaderInfoUse,
			},
		},
		OutputSerialization: &types.OutputSerialization{
			CSV: &types.CSVOutput{},
		},
	})
	require.NoError(t, err)

	stream := out.GetStream()
	defer stream.Close()

	var buf strings.Builder
	for event := range stream.Events() {
		if rec, ok := event.(*types.SelectObjectContentEventStreamMemberRecords); ok {
			buf.Write(rec.Value.Payload)
		}
	}
	require.NoError(t, stream.Err())
	require.Contains(t, buf.String(), "alice")
	require.Contains(t, buf.String(), "bob")
}
