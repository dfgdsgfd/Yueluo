package services

import (
	"context"
	"errors"
	"io"
	"time"
)

var errAIStreamIdleTimeout = errors.New("ai stream idle timeout")

func openAIStreamIdleContext(ctx context.Context, idleTimeout time.Duration, body io.Closer) (context.Context, context.CancelFunc, func()) {
	if idleTimeout <= 0 {
		return ctx, func() {}, func() {}
	}
	streamCtx, cancel := context.WithCancelCause(ctx)
	touch := make(chan struct{}, 1)
	go func() {
		timer := time.NewTimer(idleTimeout)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				cancel(ctx.Err())
				_ = body.Close()
				return
			case <-streamCtx.Done():
				return
			case <-timer.C:
				cancel(errAIStreamIdleTimeout)
				_ = body.Close()
				return
			case <-touch:
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(idleTimeout)
			}
		}
	}()
	cancelIdle := func() {
		cancel(context.Canceled)
	}
	touchIdle := func() {
		select {
		case touch <- struct{}{}:
		default:
		}
	}
	return streamCtx, cancelIdle, touchIdle
}

func openAIStreamContextError(parent context.Context, streamCtx context.Context) error {
	cause := context.Cause(streamCtx)
	switch {
	case errors.Is(cause, errAIStreamIdleTimeout):
		return AIError{Code: "error.ai_timeout", Err: cause}
	case errors.Is(parent.Err(), context.DeadlineExceeded), errors.Is(cause, context.DeadlineExceeded):
		return AIError{Code: "error.ai_timeout", Err: nonNilError(parent.Err(), cause)}
	case errors.Is(parent.Err(), context.Canceled):
		return AIError{Code: "error.ai_request_canceled", Err: parent.Err()}
	default:
		return nil
	}
}

func nonNilError(values ...error) error {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
