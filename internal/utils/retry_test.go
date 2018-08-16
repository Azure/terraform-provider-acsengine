package utils

import (
	"fmt"
	"testing"

	"k8s.io/client-go/util/retry"
)

func TestRetryOnFailure(t *testing.T) {
	i := 0
	retryErr := RetryOnFailure(retry.DefaultRetry, func() error {
		i++
		if i == 3 {
			return nil
		}
		return fmt.Errorf("failed on this try")
	})
	if retryErr != nil {
		t.Fatalf("RetryOnFailure should have succeeded but got error: %+v", retryErr)
	}
	if i != 3 {
		t.Fatalf("i should be equal to 3")
	}
}
