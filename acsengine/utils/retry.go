package utils

import (
	"acsrp/e2e/pkg/reporter"
	"fmt"
	"math/rand"
	"time"

	"github.com/jpillora/backoff"
)

const maxAttempts = 16

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func init() {
	rand.Seed(time.Now().UnixNano())
}

func rando(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// retryableError is used internally to imply that an operation failed but
// may succeed in the future.
type retryableError struct {
	error
}

func canBeRetried(err error) bool {
	_, ok := err.(*retryableError)
	return ok
}

// Retry marks the error as retryable.
func Retry(err error) error {
	if err == nil {
		return nil
	}
	return &retryableError{err}
}

// UntilSuccess calls the provided function and reattempts with exponential
// backoff if the error has been wrapped by Retry.
//
// The provided id string will be included in any events propagated through
// the given reporter to allow correlation of events relating to this operation.
func UntilSuccess(r reporter.Interface, id string, fn func() error) error {
	b := &backoff.Backoff{
		Factor: 1.2,
		Min:    time.Second,
		Max:    time.Second * 20,
	}
	var i int
	for {
		i++
		b.Attempt()
		err := fn()
		if err == nil {
			return nil
		}
		if i >= maxAttempts {
			return fmt.Errorf("Reached maximum attempts (%v): %v", maxAttempts, err)
		}
		if canBeRetried(err) {
			r.Emit(reporter.Event{
				"action":            "retry-operation",
				"operationId":       id,
				"attempt":           i,
				"attemptsRemaining": maxAttempts - i,
				"error":             err.Error(),
			})
			time.Sleep(b.Duration())
			continue
		}
		return err
	}
}

// TX returns a random string to be used as a "transaction ID" for logging
// purposes.
func TX() string {
	return rando(8)
}
