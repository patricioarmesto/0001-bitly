package main

import "time"

const (
	retryAttempts  = 3
	retryBaseDelay = 100 * time.Millisecond
)

// retry ejecuta op hasta retryAttempts veces, con backoff exponencial entre intentos.
// wait recibe el delay y puede abortar la espera devolviendo un error.
func retry(wait func(time.Duration) error, op func() error) error {
	var lastErr error
	for attempt := 0; attempt < retryAttempts; attempt++ {
		if attempt > 0 {
			if err := wait(time.Duration(1<<attempt) * retryBaseDelay); err != nil {
				return err
			}
		}
		if err := op(); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return lastErr
}
