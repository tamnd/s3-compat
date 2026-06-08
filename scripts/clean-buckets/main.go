// clean-buckets removes all test buckets matching a given prefix from the target.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/tamnd/s3-compat/compat/s3client"
)

func main() {
	prefix := flag.String("prefix", "s3compat-", "bucket name prefix to match")
	flag.Parse()

	client, err := s3client.Load("")
	if err != nil {
		log.Fatalf("loading s3 client: %v", err)
	}

	ctx := context.Background()
	out, err := client.S3.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		log.Fatalf("listing buckets: %v", err)
	}

	deleted := 0
	for _, b := range out.Buckets {
		name := ""
		if b.Name != nil {
			name = *b.Name
		}
		if !strings.HasPrefix(name, *prefix) {
			continue
		}
		fmt.Printf("deleting %s...\n", name)
		client.DeleteBucketForce(ctx, name)
		deleted++
	}
	fmt.Printf("deleted %d bucket(s) with prefix %q\n", deleted, *prefix)
}
