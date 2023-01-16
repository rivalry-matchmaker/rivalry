package backoff

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// Config defines a function that configures a backoff
type Config func(b backoff.BackOff) backoff.BackOff

// Permanent is a thin wrapper for backoff.Permanent
var Permanent = backoff.Permanent

// Exponential returns a config that outputs backoff.NewExponentialBackOff
func Exponential() Config {
	return func(_ backoff.BackOff) backoff.BackOff {
		return backoff.NewExponentialBackOff()
	}
}

// Constant returns a config that outputs backoff.NewConstantBackOff
func Constant(d time.Duration) Config {
	return func(_ backoff.BackOff) backoff.BackOff {
		return backoff.NewConstantBackOff(d)
	}
}

// MaxRetry returns a config that sets the Max Retries of the backoff
func MaxRetry(n uint64) Config {
	return func(b backoff.BackOff) backoff.BackOff {
		return backoff.WithMaxRetries(b, n)
	}
}

// Retry sets up the backoff, adds the context and calls backoff.Retry
func Retry(ctx context.Context, f func() error, config ...Config) error {
	var b backoff.BackOff
	for _, conf := range config {
		b = conf(b)
	}
	if len(config) == 0 {
		b = Exponential()(b)
	}
	b = backoff.WithContext(b, ctx)
	return backoff.Retry(f, b)
}
