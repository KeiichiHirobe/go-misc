package asyncretry

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
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

var ctx context.Context
var cancel context.CancelFunc

func Test_asyncRetry_DoWithConfigContext(t *testing.T) {
	type args struct {
		f    AsyncRetryFunc
		ctx  func() context.Context
		opts func() []Option
	}
	tests := []struct {
		name            string
		args            args
		wantErr         bool
		expectedErr     error
		expectedCounter int
	}{
		{
			name: "Stop Retry when CancelWhenConfigContextCanceled is true",
			args: args{
				f: func(ctx context.Context) error {
					counter++
					return fmt.Errorf("always error")
				},
				ctx: func() context.Context {
					return context.Background()
				},
				opts: func() []Option {
					return []Option{
						Context(ctx),
						Delay(time.Minute),
						CancelWhenConfigContextCanceled(true),
						OnRetry(func(n uint, err error) {
							cancel()
						}),
					}
				},
			},
			wantErr: true,
			// fixme: this error message is wrong due to retry-go bug
			expectedErr: fmt.Errorf(`All attempts fail:
#1: context canceled`),
			expectedCounter: 1,
		},
		{
			name: "Stop Retry when CancelWhenConfigContextCanceled is false",
			args: args{
				f: func(ctx context.Context) error {
					counter++
					return fmt.Errorf("always error")
				},
				ctx: func() context.Context {
					return context.Background()
				},
				opts: func() []Option {
					return []Option{
						Context(ctx),
						Delay(time.Minute),
						CancelWhenConfigContextCanceled(false),
						OnRetry(func(n uint, err error) {
							cancel()
						}),
					}
				},
			},
			wantErr: true,
			// fixme: this error message is wrong due to retry-go bug
			expectedErr: fmt.Errorf(`All attempts fail:
#1: context canceled`),
			expectedCounter: 1,
		},
		{
			name: "Context, argument of AsyncRetryFunc is canceled when CancelWhenConfigContextCanceled is true",
			args: args{
				f: func(ctx context.Context) error {
					counter++
					if counter == 1 {
						cancel()
					}
					select {
					case <-time.After(time.Second):
						return fmt.Errorf("context must be canceled")
					case <-ctx.Done():
						return nil
					}
				},
				ctx: func() context.Context {
					return context.Background()
				},
				opts: func() []Option {
					return []Option{
						Context(ctx),
						Timeout(0),
						Delay(time.Minute),
						CancelWhenConfigContextCanceled(true),
					}
				},
			},
			wantErr:         false,
			expectedCounter: 1,
		},
		{
			name: "Context, argument of AsyncRetryFunc is NOT canceled when CancelWhenConfigContextCanceled is false",
			args: args{
				f: func(ctx context.Context) error {
					counter++
					if counter == 1 {
						cancel()
					}
					select {
					case <-ctx.Done():
						return fmt.Errorf("context must not be canceled")
					case <-time.After(time.Second):
						return nil
					}
				},
				ctx: func() context.Context {
					return context.Background()
				},
				opts: func() []Option {
					return []Option{
						Context(ctx),
						Timeout(0),
						Delay(time.Minute),
						CancelWhenConfigContextCanceled(false),
					}
				},
			},
			wantErr:         false,
			expectedCounter: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter = 0
			ctx, cancel = context.WithCancel(context.Background())
			defer cancel()
			a := NewAsyncRetry()
			var err error
			// Be careful not call Do synchronously when actually using
			if err = a.Do(tt.args.f, tt.args.ctx(), tt.args.opts()...); (err != nil) != tt.wantErr {
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

var ch chan struct{}

func Test_asyncRetry_DoAndShutdown(t *testing.T) {
	type args struct {
		f    AsyncRetryFunc
		ctx  func() context.Context
		opts func() []Option
	}
	tests := []struct {
		name            string
		args            args
		wantErr         bool
		expectedErr     error
		expectedCounter int
	}{
		{
			name: "Stop Retry in shutdown when CancelWhenShutdown is true",
			args: args{
				f: func(ctx context.Context) error {
					counter++
					return fmt.Errorf("always error")
				},
				ctx: func() context.Context {
					return context.Background()
				},
				opts: func() []Option {
					return []Option{
						Delay(time.Minute),
						CancelWhenShutdown(true),
						OnRetry(func(n uint, err error) {
							if n == 0 {
								close(ch)
							}
						}),
					}
				},
			},
			wantErr: true,
			// fixme: this error message is wrong due to retry-go bug
			expectedErr: fmt.Errorf(`All attempts fail:
#1: context canceled`),
			expectedCounter: 1,
		},
		{
			name: "Stop Retry in shutdown when CancelWhenShutdown is false",
			args: args{
				f: func(ctx context.Context) error {
					counter++
					return fmt.Errorf("always error")
				},
				ctx: func() context.Context {
					return context.Background()
				},
				opts: func() []Option {
					return []Option{
						Delay(time.Minute),
						CancelWhenShutdown(false),
						OnRetry(func(n uint, err error) {
							if n == 0 {
								close(ch)
							}
						}),
					}
				},
			},
			wantErr: true,
			// fixme: this error message is wrong due to retry-go bug
			expectedErr: fmt.Errorf(`All attempts fail:
#1: context canceled`),
			expectedCounter: 1,
		},
		{
			name: "Context, argument of AsyncRetryFunc is canceled when CancelWhenShutdown is true",
			args: args{
				f: func(ctx context.Context) error {
					counter++
					if counter == 1 {
						close(ch)
					}
					select {
					case <-time.After(time.Second):
						return fmt.Errorf("context must be canceled")
					case <-ctx.Done():
						return nil
					}
				},
				ctx: func() context.Context {
					return context.Background()
				},
				opts: func() []Option {
					return []Option{
						Delay(time.Minute),
						CancelWhenShutdown(true),
					}
				},
			},
			wantErr:         false,
			expectedCounter: 1,
		},
		{
			name: "Context, argument of AsyncRetryFunc is canceled when CancelWhenShutdown is false",
			args: args{
				f: func(ctx context.Context) error {
					counter++
					if counter == 1 {
						close(ch)
					}
					select {
					case <-ctx.Done():
						return fmt.Errorf("context must not be canceled")
					case <-time.After(time.Second):
						return nil
					}
				},
				ctx: func() context.Context {
					return context.Background()
				},
				opts: func() []Option {
					return []Option{
						Delay(time.Minute),
						CancelWhenShutdown(false),
					}
				},
			},
			wantErr:         false,
			expectedCounter: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch = make(chan struct{})
			counter = 0
			a := NewAsyncRetry()

			var doErr = make(chan error)
			var shutdownErr = make(chan error)
			go func() {
				doErr <- a.Do(
					tt.args.f, tt.args.ctx(), tt.args.opts()...,
				)
			}()
			go func() {
				<-ch
				shutdownErr <- a.Shutdown(context.Background())
			}()

			var err error
			select {
			case err = <-shutdownErr:
			case <-time.After(time.Second * 10):
				t.Errorf("too long")
			}
			if err != nil {
				t.Errorf("Shutdown() error = %v, wantErr %v", err, nil)
			}
			select {
			case err = <-doErr:
			default:
				t.Errorf("Do must be finished before Shutdown")
			}
			if (err != nil) != tt.wantErr {
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

func Test_ShutdownOrder(t *testing.T) {
	type args struct {
		f    AsyncRetryFunc
		ctx  func() context.Context
		opts []Option
	}
	tests := []struct {
		name       string
		szDo       int
		szShutdown int
	}{
		{
			"Calls of Do which happens before call of shutdown blocks shutdown, and calls of Do which happen after call of shutdown return InShutdownErr",
			1000,
			1,
		},
		{
			"Multiple shutdown call is OK",
			1000,
			100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			szDo := tt.szDo
			szShutdown := tt.szShutdown
			var results = make(chan int)
			a := NewAsyncRetry()
			var wg sync.WaitGroup
			for i := 0; i < szDo; i++ {
				wg.Add(1)
				go func() {
					err := a.Do(
						func(ctx context.Context) error {
							wg.Done()
							time.Sleep(time.Millisecond * time.Duration(rand.Intn(1000)))
							results <- 1
							return nil
						},
						context.Background(),
						Timeout(0),
					)
					if err != nil {
						t.Errorf("Do() error = %v, wantErr %v", err, nil)
					}
				}()
			}
			for i := 0; i < szShutdown; i++ {
				go func() {
					wg.Wait()
					err := a.Shutdown(context.Background())
					results <- 2
					if err != nil {
						t.Errorf("Shutdown() error = %v, wantErr %v", err, nil)
					}
				}()
			}
			i := 0
			for i < szDo+szShutdown {
				v := <-results
				if i < szDo {
					if v != 1 {
						t.Errorf("must be 1")
					}
				} else {
					if v != 2 {
						t.Errorf("must be 2")
					}
				}
				i++
			}
			// after shutdown
			err := a.Do(
				func(ctx context.Context) error {
					return nil
				},
				context.Background(),
			)
			if err == nil || err.Error() != InShutdownErr.Error() {
				t.Errorf("call of Do after shudown must returns InShutdownErr")
			}
		})
	}
}
