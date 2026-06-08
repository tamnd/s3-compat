// Package assert provides S3-specific test assertion helpers.
package assert

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/smithy-go"
)

// NoError fails t if err is non-nil.
func NoError(t testing.TB, err error, msgAndArgs ...any) {
	t.Helper()
	if err != nil {
		if len(msgAndArgs) > 0 {
			t.Fatalf("%v: unexpected error: %v", fmt.Sprint(msgAndArgs...), err)
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

// S3ErrorCode fails t if err is not an API error with the expected code.
func S3ErrorCode(t testing.TB, err error, wantCode string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected S3 error %q, got nil", wantCode)
		return
	}
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected smithy.APIError with code %q, got %T: %v", wantCode, err, err)
		return
	}
	if apiErr.ErrorCode() != wantCode {
		t.Fatalf("expected error code %q, got %q (%v)", wantCode, apiErr.ErrorCode(), err)
	}
}

// httpStatusErr is the interface exposed by AWS SDK HTTP response errors.
type httpStatusErr interface {
	HTTPStatusCode() int
}

// S3HTTPStatus fails t if err does not carry the expected HTTP status code.
func S3HTTPStatus(t testing.TB, err error, wantStatus int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected HTTP %d error, got nil", wantStatus)
		return
	}
	var httpErr httpStatusErr
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected error with HTTPStatusCode() %d, got %T: %v", wantStatus, err, err)
		return
	}
	if httpErr.HTTPStatusCode() != wantStatus {
		t.Fatalf("expected HTTP status %d, got %d (%v)", wantStatus, httpErr.HTTPStatusCode(), err)
	}
}

// NoSuchBucket fails t if err is not a NoSuchBucket S3 error.
func NoSuchBucket(t testing.TB, err error) {
	t.Helper()
	S3ErrorCode(t, err, "NoSuchBucket")
}

// NoSuchKey fails t if err is not a NoSuchKey S3 error.
func NoSuchKey(t testing.TB, err error) {
	t.Helper()
	S3ErrorCode(t, err, "NoSuchKey")
}

// AccessDenied fails t if err is not an AccessDenied S3 error.
func AccessDenied(t testing.TB, err error) {
	t.Helper()
	S3ErrorCode(t, err, "AccessDenied")
}

// BucketNotEmpty fails t if err is not a bucket-not-empty or conflict error.
// Accepted codes: BucketNotEmpty, Conflict, BucketAlreadyExists.
func BucketNotEmpty(t testing.TB, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected bucket-not-empty error, got nil")
		return
	}
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected smithy.APIError, got %T: %v", err, err)
		return
	}
	code := apiErr.ErrorCode()
	switch code {
	case "BucketNotEmpty", "Conflict", "BucketAlreadyExists":
		// accepted
	default:
		t.Fatalf("expected BucketNotEmpty/Conflict/BucketAlreadyExists, got %q (%v)", code, err)
	}
}
