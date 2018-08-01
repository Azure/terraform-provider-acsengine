package utils

import (
	"k8s.io/apimachinery/pkg/util/wait"
)

// RetryOnFailedGet is based on k8s.io/client-go RetryOnConflict but for any error
// I would like to figure out what the exact error is so I can use it
func RetryOnFailedGet(backoff wait.Backoff, fn func() error) error {
	var lastConflictErr error
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		err := fn()
		switch {
		case err == nil:
			return true, nil
		// case errors.IsConflict(err):
		default:
			lastConflictErr = err
			return false, nil
			// default:
			// return false, err
		}
	})
	if err == wait.ErrWaitTimeout {
		err = lastConflictErr
	}
	return err
}
