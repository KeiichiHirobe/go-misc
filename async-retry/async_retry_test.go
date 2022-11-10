package asyncretry

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type contextValueKeyT int

const contextValueKey contextValueKeyT = 1

var counter = 0

func Test_asyncRetry_Do(t *testing.T) {
	type args struct {
		f    AsyncRetryFunc
		ctx  func() context.Context
		opts []Option
	}
	tests := []struct {
		name            string
		args            args
		wantErr         bool
		expectedErr     error
		expectedCounter int
	}{
		{
			name: "Retry until success",
			args: args{
				f: func(ctx context.Context) error {
					counter++
					if counter < 5 {
						return fmt.Errorf("%vth try", counter)
					}
					return nil
				},
				ctx: func() context.Context {
					return context.Background()
				},
				opts: []Option{
					Attempts(10),
					Delay(1 * time.Millisecond),
				},
			},
			wantErr:         false,
			expectedErr:     nil,
			expectedCounter: 5,
		},
		{
			name: "Retry but fail",
			args: args{
				f: func(ctx context.Context) error {
					counter++
					if counter < 5 {
						return fmt.Errorf("%vth try", counter)
					}
					return nil
				},
				ctx: func() context.Context {
					return context.Background()
				},
				opts: []Option{
					Attempts(3),
					Delay(1 * time.Millisecond),
				},
			},
			wantErr: true,
			expectedErr: fmt.Errorf(`All attempts fail:
#1: 1th try
#2: 2th try
#3: 3th try`),
			expectedCounter: 3,
		},
		{
			name: "Cancellation of context, argument of Do is not propagated to AsyncRetryFunc",
			args: args{
				f: func(ctx context.Context) error {
					select {
					case <-ctx.Done():
						return fmt.Errorf("ctx canceled")
					default:
					}
					if ctx.Err() != nil {
						return fmt.Errorf("ctx.Err() must be nil")
					}
					return nil
				},
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				},
				opts: []Option{
					Attempts(1),
				},
			},
			wantErr:     false,
			expectedErr: nil,
		},
		{
			name: "Context, argument of AsyncRetryFunc keeps value",
			args: args{
				f: func(ctx context.Context) error {
					if ctx.Value(contextValueKey) != 1 {
						return fmt.Errorf("ctx.Value mismatch")
					}
					return nil
				},
				ctx: func() context.Context {
					return context.WithValue(context.Background(), contextValueKey, 1)
				},
				opts: []Option{
					Attempts(1),
				},
			},
			wantErr:     false,
			expectedErr: nil,
		},
		{
			name: "Timeout set correctly for each try",
			args: args{
				f: func(ctx context.Context) error {
					counter++
					select {
					case <-ctx.Done():
						if counter < 3 {
							return fmt.Errorf("timeout")
						}
						return nil
					case <-time.After(time.Minute):
						return Unrecoverable(fmt.Errorf("timeout not working"))
					}
				},
				ctx: func() context.Context {
					return context.Background()
				},
				opts: []Option{
					Delay(1 * time.Millisecond),
					Timeout(10 * time.Millisecond),
					Attempts(5),
				},
			},
			wantErr:         false,
			expectedErr:     nil,
			expectedCounter: 3,
		},
		{
			name: "Recover from panic",
			args: args{
				f: func(ctx context.Context) error {
					panic("call panic for test")
				},
				ctx: func() context.Context {
					return context.Background()
				},
				opts: nil,
			},
			wantErr:     true,
			expectedErr: fmt.Errorf("panicking while AsyncRetry err: call panic for test"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter = 0
			a := NewAsyncRetry()
			var err error
			// Be careful not call Do synchronously when actually using
			if err = a.Do(tt.args.f, tt.args.ctx(), tt.args.opts...); (err != nil) != tt.wantErr {
				t.Errorf("Do() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				if tt.expectedErr.Error() != err.Error() {
					t.Errorf("Do() error = %v, expectedErr %v", err, tt.expectedErr)
				}
			}
			if tt.expectedCounter != 0 {
				if counter != tt.expectedCounter {
					t.Errorf("Do() mismatch called count actutal: %v, expected: %v", counter, tt.expectedCounter)
				}
			}
		})
	}
}

func Test_asyncRetry_DoWithConfigContext(t *testing.T) {
	t.Run("Stop Retry when CancelWhenConfigContextCanceled is true", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		a := NewAsyncRetry()
		var counter = 0
		// fixme: this error message is wrong due to retry-go bug
		var expectedErrorMsg = `All attempts fail:
#1: context canceled`
		if err := a.Do(
			func(ctx context.Context) error {
				counter++
				if counter == 1 {
					cancel()
				}
				return fmt.Errorf("always error")
			},
			context.Background(),
			Context(ctx),
			Delay(time.Minute),
			CancelWhenConfigContextCanceled(true),
		); err == nil || err.Error() != expectedErrorMsg {
			t.Errorf("Do() error = %v, wantErr %v", err, expectedErrorMsg)
		}
		if counter != 1 {
			t.Errorf("Do() mismatch called count actutal: %v, expected: %v", counter, 1)
		}
	})
	t.Run("Stop Retry when CancelWhenConfigContextCanceled is false", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		a := NewAsyncRetry()
		var counter = 0
		// fixme: this error message is wrong due to retry-go bug
		var expectedErrorMsg = `All attempts fail:
#1: context canceled`
		if err := a.Do(
			func(ctx context.Context) error {
				counter++
				if counter == 1 {
					cancel()
				}
				return fmt.Errorf("always error")
			},
			context.Background(),
			Context(ctx),
			Delay(time.Minute),
			CancelWhenConfigContextCanceled(false),
		); err == nil || err.Error() != expectedErrorMsg {
			t.Errorf("Do() error = %v, wantErr %v", err, expectedErrorMsg)
		}
		if counter != 1 {
			t.Errorf("Do() mismatch called count actutal: %v, expected: %v", counter, 1)
		}
	})
	t.Run("Context, argument of AsyncRetryFunc is canceled when CancelWhenConfigContextCanceled is true", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		a := NewAsyncRetry()
		var counter = 0
		// fixme: this error message is wrong due to retry-go bug
		var expectedErrorMsg = `All attempts fail:
#1: context canceled`
		if err := a.Do(
			func(ctx context.Context) error {
				counter++
				if counter == 1 {
					cancel()
				}
				<-ctx.Done()
				return fmt.Errorf("argument context canceled")
			},
			context.Background(),
			Context(ctx),
			Timeout(0),
			Delay(time.Minute),
			CancelWhenConfigContextCanceled(true),
		); err == nil || err.Error() != expectedErrorMsg {
			t.Errorf("Do() error = %v, wantErr %v", err, expectedErrorMsg)
		}
		if counter != 1 {
			t.Errorf("Do() mismatch called count actutal: %v, expected: %v", counter, 1)
		}
	})
	t.Run("Context, argument of AsyncRetryFunc is NOT canceled when CancelWhenConfigContextCanceled is true", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		a := NewAsyncRetry()
		var counter = 0
		// fixme: this error message is wrong due to retry-go bug
		var expectedErrorMsg = `All attempts fail:
#1: context canceled`
		if err := a.Do(
			func(ctx context.Context) error {
				counter++
				if counter == 1 {
					cancel()
				}
				select {
				case <-ctx.Done():
					return fmt.Errorf("argument context canceled")
				case <-time.After(time.Second):
					return fmt.Errorf("too long action")
				}
			},
			context.Background(),
			Context(ctx),
			Timeout(0),
			Delay(time.Minute),
			CancelWhenConfigContextCanceled(false),
		); err == nil || err.Error() != expectedErrorMsg {
			t.Errorf("Do() error = %v, wantErr %v", err, expectedErrorMsg)
		}
		if counter != 1 {
			t.Errorf("Do() mismatch called count actutal: %v, expected: %v", counter, 1)
		}
	})
}

func Test_asyncRetry_DoAndShutdown(t *testing.T) {
	t.Run("Stop Retry at shutdown when CancelWhenShutdown is true", func(t *testing.T) {
		var ch = make(chan struct{})
		a := NewAsyncRetry()
		var counter = 0
		var doErr = make(chan error)
		var shutdownErr = make(chan error)
		// fixme: this error message is wrong due to retry-go bug
		var expectedErrorMsg = `All attempts fail:
#1: context canceled`
		go func() {
			doErr <- a.Do(
				func(ctx context.Context) error {
					counter++
					if counter == 1 {
						close(ch)
					}
					return fmt.Errorf("always error")
				},
				context.Background(),
				Context(context.Background()),
				Delay(time.Minute),
				CancelWhenShutdown(true),
			)
		}()
		go func() {
			// wait until Do is called
			<-ch
			shutdownErr <- a.Shutdown(context.Background())
		}()

		var err error
		select {
		case err = <-shutdownErr:
		case <-time.After(time.Second * 10):
			t.Errorf("Do must stop waiting")
		}
		if err != nil {
			t.Errorf("Shutdown() error = %v, wantErr %v", err, nil)
		}
		select {
		case err = <-doErr:
		default:
			t.Errorf("Do must be finished before Shutdown")
		}
		if err == nil || err.Error() != expectedErrorMsg {
			t.Errorf("Do() error = %v, wantErr %v", err, expectedErrorMsg)
		}
		if counter != 1 {
			t.Errorf("Do() mismatch called count actutal: %v, expected: %v", counter, 1)
		}
	})
}
