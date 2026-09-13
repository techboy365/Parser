package retry

import (
	"context"
	"fmt"
	"time"

	"github.com/techboy365/Parser/internal/config"
)

type PermanentError struct {
	Err error
}

func (e *PermanentError) Error() string {
	return e.Err.Error()
}

func (e *PermanentError) Unwrap() error {
	return e.Err
}

func Permanent(err error) error {
	return &PermanentError{Err: err}
}

func IsPermanent(err error) bool {
	_, ok := err.(*PermanentError)
	return ok
}

func Do(ctx context.Context, cfg config.RetryConfig, op func(attempt int) error) error {
	backoff := time.Duration(cfg.InitialBackoffMS) * time.Millisecond
	var lastErr error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		lastErr = op(attempt)
		if lastErr == nil {
			return nil
		}
		if IsPermanent(lastErr) {
			return lastErr
		}
		if attempt == cfg.MaxAttempts {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		backoff = nextBackoff(backoff, cfg)
	}
	return fmt.Errorf("after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

func nextBackoff(current time.Duration, cfg config.RetryConfig) time.Duration {
	next := time.Duration(float64(current) * cfg.Multiplier)
	max := time.Duration(cfg.MaxBackoffMS) * time.Millisecond
	if next > max {
		return max
	}
	return next
}
