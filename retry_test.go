// Copyright (c) 2024 Tiago Melo. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

package retry

import (
	"errors"
	"testing"
	"time"
)

type mockStrategy struct {
	delays   []time.Duration
	attempts int
}

func (m *mockStrategy) GetDelay(attempt int) time.Duration {
	if attempt < len(m.delays) {
		return m.delays[attempt]
	}
	return 0
}

func (m *mockStrategy) GetAttempts() int {
	return m.attempts
}

func TestDo(t *testing.T) {
	tests := []struct {
		name             string
		strategy         retryStrategy
		f                func() error
		expectedErr      error
		expectedAttempts int
	}{
		{
			name: "Successful execution on first attempt",
			strategy: &mockStrategy{
				delays:   []time.Duration{0},
				attempts: 3,
			},
			f: func() error {
				return nil
			},
			expectedErr:      nil,
			expectedAttempts: 1,
		},
		{
			name: "Retry with success after 2 attempts",
			strategy: &mockStrategy{
				delays:   []time.Duration{time.Millisecond, time.Millisecond},
				attempts: 3,
			},
			f: func() func() error {
				count := 0
				return func() error {
					if count < 1 { // Fail on the first attempt
						count++
						return errors.New("retryable error")
					}
					return nil // Succeed on the second attempt
				}
			}(),
			expectedErr:      nil,
			expectedAttempts: 2,
		},
		{
			name: "Exhaust retries",
			strategy: &mockStrategy{
				delays:   []time.Duration{time.Millisecond, time.Millisecond},
				attempts: 2,
			},
			f: func() error {
				return errors.New("retryable error")
			},
			expectedErr:      errors.New("retryable error"),
			expectedAttempts: 2,
		},
		{
			name: "Fatal error stops retries immediately",
			strategy: &mockStrategy{
				delays:   []time.Duration{time.Millisecond, time.Millisecond},
				attempts: 3,
			},
			f: func() error {
				return EndRetry(errors.New("fatal error"))
			},
			expectedErr:      errors.New("fatal error"),
			expectedAttempts: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts, err := Do(tt.f, tt.strategy)
			if (err != nil && tt.expectedErr == nil) || (err == nil && tt.expectedErr != nil) || (err != nil && err.Error() != tt.expectedErr.Error()) {
				t.Errorf("Expected error %v, got %v", tt.expectedErr, err)
			}
			if attempts != tt.expectedAttempts {
				t.Errorf("Expected attempts %v, got %v", tt.expectedAttempts, attempts)
			}
		})
	}
}

func TestCapDelay(t *testing.T) {
	tests := []struct {
		name     string
		delay    time.Duration
		max      time.Duration
		expected time.Duration
	}{
		{
			name:     "No cap applied",
			delay:    500 * time.Millisecond,
			max:      1 * time.Second,
			expected: 500 * time.Millisecond,
		},
		{
			name:     "Cap applied",
			delay:    2 * time.Second,
			max:      1 * time.Second,
			expected: 1 * time.Second,
		},
		{
			name:     "Max zero, no cap",
			delay:    1 * time.Second,
			max:      0,
			expected: 1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := capDelay(tt.delay, tt.max)
			if actual != tt.expected {
				t.Errorf("Expected delay %v, got %v", tt.expected, actual)
			}
		})
	}
}

func TestBackoffImplementations(t *testing.T) {
	strategies := []struct {
		name     string
		strategy retryStrategy
	}{
		{
			name:     "LinearBackoff",
			strategy: NewLinearBackoff(100*time.Millisecond, 1*time.Second, 5),
		},
		{
			name:     "ExponentialBackoff",
			strategy: NewExponentialBackoff(100*time.Millisecond, 1*time.Second, 5),
		},
		{
			name:     "RandomizedBackoff",
			strategy: NewRandomizedBackoff(100*time.Millisecond, 1*time.Second, 5),
		},
		{
			name:     "ConstantBackoff",
			strategy: NewConstantBackoff(100*time.Millisecond, 5),
		},
	}

	for _, tt := range strategies {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < tt.strategy.GetAttempts(); i++ {
				delay := tt.strategy.GetDelay(i)
				if delay > 1*time.Second {
					t.Errorf("Delay exceeded max delay: %v", delay)
				}
			}
		})
	}
}

func TestFatalError(t *testing.T) {
	expectedMessage := "fatal error occurred"
	fatalErr := &fatal{cause: errors.New(expectedMessage)}

	if fatalErr.Error() != expectedMessage {
		t.Errorf("Expected error message '%s', got '%s'", expectedMessage, fatalErr.Error())
	}
}
