package main

import (
	"fmt"
	"time"
)

func withRetry(numAttempts int, interval time.Duration, f func() error) error {
	var err error
	for i := 0; i < numAttempts; i++ {

		err = f()
		if err == nil {
			return nil
		}

		time.Sleep(interval)
	}

	return fmt.Errorf("failed after %d retries: %w", numAttempts, err)
}
