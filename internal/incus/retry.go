package incus

import "time"

type retryPolicy struct {
	// number of retries
	attempts int
	// between attempts
	delay time.Duration
}
