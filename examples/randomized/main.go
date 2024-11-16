// Copyright (c) 2024 Tiago Melo. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/tiagomelo/go-retry"
)

func main() {
	strategy := retry.NewRandomizedBackoff(100*time.Millisecond, 1*time.Second, 5)

	fmt.Println("=== case 1: retryable error, eventually succeeds ===")
	attempts, err := retry.Do(func() error {
		fmt.Println("attempting operation...")
		return errors.New("temporary error")
	}, strategy)

	fmt.Printf("completed after %d attempts. Error: %v\n\n", attempts, err)

	fmt.Println("=== case 2: fatal error stops retries ===")
	var count int
	attempts, err = retry.Do(func() error {
		count++
		fmt.Println("attempting operation...")
		if count == 3 {
			return retry.EndRetry(errors.New("fatal error"))
		}
		return errors.New("temporary error")
	}, strategy)

	fmt.Printf("completed after %d attempts. Error: %v\n", attempts, err)
}
