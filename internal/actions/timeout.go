package actions

import (
	"context"
	"fmt"
	"time"
)

var DefaultActionTimeout = 30 * time.Second

func actionTimeout() time.Duration {
	if ActiveConfig != nil && ActiveConfig.ActionTimeoutMs > 0 {
		return time.Duration(ActiveConfig.ActionTimeoutMs) * time.Millisecond
	}
	return DefaultActionTimeout
}

func WithTimeout(fn func() error) error {
	ctx, cancel := context.WithTimeout(context.Background(), actionTimeout())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("action timed out after %v", actionTimeout())
	}
}
