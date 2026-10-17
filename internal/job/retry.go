package job

import (
	"errors"
	"math/rand/v2"
	"time"
)

type permanent struct{ error }

// Permanent marks errors that must not consume further attempts.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return permanent{err}
}

// IsPermanent tests a wrapped permanent error.
func IsPermanent(err error) bool { var p permanent; return errors.As(err, &p) }

// RetryDelay returns bounded full-jitter exponential backoff.
func RetryDelay(attempt int) time.Duration {
	d := time.Second
	for n := 1; n < attempt && d < 5*time.Minute; n++ {
		d = min(d*2, 5*time.Minute)
	}
	return time.Duration(rand.Int64N(int64(d)))
}
