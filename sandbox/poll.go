package sandbox

import (
	"context"
	"time"
)

// PollOption 配置轮询行为的选项。
type PollOption func(*pollOpts)

type pollOpts struct {
	interval    time.Duration
	maxInterval time.Duration
	backoff     float64 // 退避倍数，默认 1.0（无退避）
	onPoll      func(attempt int)
}

func defaultPollOpts(defaultInterval time.Duration) *pollOpts {
	return &pollOpts{
		interval:    defaultInterval,
		maxInterval: 0,
		backoff:     1.0,
	}
}

// WithPollInterval 设置轮询间隔。
func WithPollInterval(d time.Duration) PollOption {
	return func(o *pollOpts) { o.interval = d }
}

// WithBackoff 设置指数退避倍数和最大间隔。
// multiplier 为每次轮询后间隔的乘数（如 1.5 表示每次增加 50%），
// maxInterval 为间隔上限（0 表示不限制）。
func WithBackoff(multiplier float64, maxInterval time.Duration) PollOption {
	return func(o *pollOpts) {
		o.backoff = multiplier
		o.maxInterval = maxInterval
	}
}

// WithOnPoll 设置每次轮询时的回调函数。
// attempt 从 1 开始递增。
func WithOnPoll(fn func(attempt int)) PollOption {
	return func(o *pollOpts) { o.onPoll = fn }
}

// pollLoop 是 WaitForReady 和 WaitForBuild 共享的轮询循环。
// pollFn 在每次轮询时被调用，返回 (done, result, error)。
func pollLoop[T any](ctx context.Context, opts *pollOpts, pollFn func() (bool, T, error)) (T, error) {
	if opts.interval <= 0 {
		opts.interval = time.Second
	}

	interval := opts.interval
	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	attempt := 0
	for {
		attempt++
		if opts.onPoll != nil {
			opts.onPoll(attempt)
		}

		done, result, err := pollFn()
		if err != nil {
			return result, err
		}
		if done {
			return result, nil
		}

		// 计算下次间隔（退避）
		if opts.backoff > 1.0 {
			interval = time.Duration(float64(interval) * opts.backoff)
			if opts.maxInterval > 0 && interval > opts.maxInterval {
				interval = opts.maxInterval
			}
		}

		if timer == nil {
			timer = time.NewTimer(interval)
		} else {
			timer.Reset(interval)
		}
		select {
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		case <-timer.C:
		}
	}
}
