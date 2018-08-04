package utils

import (
	"k8s.io/apimachinery/pkg/util/wait"
)

// RetryOnFailure is based on k8s.io/client-go RetryOnConflict but for any error
// If I figure out common error(s) I can use it as case instead of having default retry
func RetryOnFailure(backoff wait.Backoff, fn func() error) error {
	var lastConflictErr error
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		err := fn()
		switch {
		case err == nil:
			return true, nil
		default:
			lastConflictErr = err
			return false, nil
		}
	})
	if err == wait.ErrWaitTimeout {
		err = lastConflictErr
	}
	return err
}
