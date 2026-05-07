package incus

import (
	"context"
	"time"
)

type retryPolicy struct {
	// number of retries
	attempts int
	// between attempts
	delay time.Duration
}

func retry[T any](
	ctx context.Context,
	p retryPolicy,
	op func() (result T, err error),
	shouldRetry func(error) bool,
) (T, error) {
	var result T
	var err error
	for range p.attempts {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
			result, err = op()
			if err != nil && shouldRetry(err) {
				select {
				case <-ctx.Done():
					return result, ctx.Err()
				case <-time.After(p.delay):
				}
			} else {
				return result, err
			}
		}
	}

	return result, err
}
