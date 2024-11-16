// Copyright (c) 2024 Tiago Melo. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

package retry

import (
	"errors"
	"math/rand"
	"time"
)

// retryStrategy defines the strategy pattern for retrying a function.
type retryStrategy interface {
	// GetDelay returns the delay for the given attempt.
	GetDelay(attempt int) time.Duration

	// GetAttempts returns the number of attempts.
	GetAttempts() int
}

// LinearBackoff is a retry strategy that waits a fixed amount of time between each retry.
type LinearBackoff struct {
	retryDelay time.Duration
	maxDelay   time.Duration
	attempts   int
}

// NewLinearBackoff creates a new LinearBackoff strategy.
func NewLinearBackoff(retryDelay, maxDelay time.Duration, attempts int) LinearBackoff {
	return LinearBackoff{
		retryDelay: retryDelay,
		maxDelay:   maxDelay,
		attempts:   attempts,
	}
}

func (b LinearBackoff) GetDelay(attempt int) time.Duration {
	return capDelay(time.Duration(attempt)*b.retryDelay, b.maxDelay)
}

func (b LinearBackoff) GetAttempts() int {
	return b.attempts
}

// ExponentialBackoff is a retry strategy that waits an
// exponentially increasing amount of time between retries.
type ExponentialBackoff struct {
	retryDelay time.Duration
	maxDelay   time.Duration
	attempts   int
}

// NewExponentialBackoff creates a new ExponentialBackoff strategy.
func NewExponentialBackoff(retryDelay, maxDelay time.Duration, attempts int) ExponentialBackoff {
	return ExponentialBackoff{
		retryDelay: retryDelay,
		maxDelay:   maxDelay,
		attempts:   attempts,
	}
}

func (b ExponentialBackoff) GetDelay(attempt int) time.Duration {
	return capDelay(time.Duration(1<<attempt)*b.retryDelay, b.maxDelay)
}

func (b ExponentialBackoff) GetAttempts() int {
	return b.attempts
}

// RandomizedBackoff is a retry strategy that waits a
// random amount of time between retries.
type RandomizedBackoff struct {
	retryDelay time.Duration
	maxDelay   time.Duration
	attempts   int
	rand       *rand.Rand
}

// NewRandomizedBackoff creates a new RandomizedBackoff strategy.
func NewRandomizedBackoff(retryDelay, maxDelay time.Duration, attempts int) RandomizedBackoff {
	return RandomizedBackoff{
		retryDelay: retryDelay,
		maxDelay:   maxDelay,
		attempts:   attempts,
		rand:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (b RandomizedBackoff) GetDelay(attempt int) time.Duration {
	return capDelay(time.Duration(b.rand.Intn(attempt+1))*b.retryDelay, b.maxDelay)
}

func (b RandomizedBackoff) GetAttempts() int {
	return b.attempts
}

// ConstantBackoff is a retry strategy that waits a
// fixed amount of time between retries.
type ConstantBackoff struct {
	retryDelay time.Duration
	attempts   int
}

// NewConstantBackoff creates a new ConstantBackoff strategy.
func NewConstantBackoff(retryDelay time.Duration, attempts int) ConstantBackoff {
	return ConstantBackoff{
		retryDelay: retryDelay,
		attempts:   attempts,
	}
}

func (b ConstantBackoff) GetDelay(_ int) time.Duration {
	return b.retryDelay
}

func (b ConstantBackoff) GetAttempts() int {
	return b.attempts
}

// Do retries a function until it returns nil or a fatal error.
// The function will be retried according to the retry strategy.
func Do(f func() error, rs retryStrategy) (attempts int, err error) {
	maxAttempts := rs.GetAttempts()
	for {
		if err = attemptRetry(f, rs, attempts); err == nil {
			return attempts + 1, nil
		}
		if fatalErr := checkFatal(err); fatalErr != nil {
			return attempts + 1, fatalErr
		}
		attempts++
		if shouldStopRetry(attempts, maxAttempts) {
			break
		}
	}
	return attempts, err
}

// capDelay ensures the delay does not exceed the max limit.
func capDelay(delay, max time.Duration) time.Duration {
	if max > 0 && delay > max {
		return max
	}
	return delay
}

// EndRetry signals to stop retrying by wrapping an error as fatal.
func EndRetry(err error) error {
	return &fatal{cause: err}
}

// fatal is a non-recoverable error type.
type fatal struct {
	cause error
}

// Error returns the error message.
func (f *fatal) Error() string {
	return f.cause.Error()
}

// Unwrap returns the wrapped error.
func (f *fatal) Unwrap() error {
	return f.cause
}

// shouldStopRetry checks if the retry should stop.
func shouldStopRetry(attempts, maxAttempts int) bool {
	// maxAttempts == -1 means infinite retries.
	return maxAttempts != -1 && attempts >= maxAttempts
}

// attemptRetry retries a function according to the retry strategy.
func attemptRetry(f func() error, rs retryStrategy, attempts int) error {
	if attempts > 0 {
		time.Sleep(rs.GetDelay(attempts))
	}
	return f()
}

// checkFatal checks if the error is a fatal error.
func checkFatal(err error) error {
	var fatalErr *fatal
	if errors.As(err, &fatalErr) {
		return fatalErr.Unwrap()
	}
	return nil
}
