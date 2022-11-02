package asyncretry

import (
	"context"
	"fmt"
	"sync"

	"github.com/avast/retry-go/v4"
)

type AsyncRetryFunc func(ctx context.Context) error

type AsyncRetry interface {
	Do(f AsyncRetryFunc, ctx context.Context, opts ...Option) error
	Shutdown(ctx context.Context) error
}

type asyncRetry struct {
	mu           sync.Mutex // guards wg and shutdownChan
	wg           sync.WaitGroup
	shutdownChan chan struct{}
}

func NewAsyncRetry() AsyncRetry {
	return &asyncRetry{
		wg:           sync.WaitGroup{},
		shutdownChan: make(chan struct{}),
	}
}

var InShutdownErr = fmt.Errorf("AsyncRetry is in shutdown")

func (a *asyncRetry) Do(f AsyncRetryFunc, ctx context.Context, opts ...Option) (retErr error) {
	a.mu.Lock()
	select {
	case <-a.shutdownChan:
		return InShutdownErr
	default:
	}
	a.wg.Add(1)
	a.mu.Unlock()
	defer a.wg.Done()

	config := DefaultConfig
	for _, opt := range opts {
		opt(&config)
	}
	defer func() {
		if err := recover(); err != nil {
			retErr = fmt.Errorf("panicking while AsyncRetry err: %v", err)
		}
	}()

	baseCtx := WithoutCancel(ctx)
	noMoreRetryCtx, noMoreRetry := context.WithCancel(config.context)
	defer noMoreRetry()
	return retry.Do(
		func() error {
			done := make(chan struct{})
			var ctx context.Context
			var cancel context.CancelFunc
			if config.timeout > 0 {
				ctx, cancel = context.WithTimeout(baseCtx, config.timeout)
			} else {
				ctx, cancel = context.WithCancel(baseCtx)
			}
			defer func() {
				close(done)
				cancel()
			}()
			go func() {
				select {
				case <-a.shutdownChan:
					noMoreRetry()
					if config.cancelWhenShutdown {
						cancel()
					}
				case <-config.context.Done():
					if config.cancelWhenConfigContextCanceled {
						cancel()
					}
				case <-done:
				}
			}()
			return f(ctx)
		},
		retry.Attempts(config.attempts),
		retry.OnRetry(retry.OnRetryFunc(config.onRetry)),
		retry.RetryIf(retry.RetryIfFunc(config.retryIf)),
		retry.Context(noMoreRetryCtx),
		retry.Delay(config.delay),
		retry.MaxJitter(config.maxJitter),
	)
}

func (a *asyncRetry) Shutdown(ctx context.Context) error {
	a.mu.Lock()
	select {
	case <-a.shutdownChan:
		// Already closed.
	default:
		// Guarded by a.mu
		close(a.shutdownChan)
	}
	a.mu.Unlock()

	ch := make(chan struct{})
	go func() {
		a.wg.Wait()
		<-ch
	}()

	var err error
	select {
	case <-ch:
	case <-ctx.Done():
		err = ctx.Err()
	}
	return err
}

// Unrecoverable wraps error.
func Unrecoverable(err error) error {
	return retry.Unrecoverable(err)
}

// IsRecoverable checks if error is recoverable
func IsRecoverable(err error) bool {
	return retry.IsRecoverable(err)
}
