package backoff

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/alex-ant/gomath/rational"
)

type (
	// Backoff 退避器接口
	Backoff interface {
		// Time 获取重试的退避时长间隔
		Time(context.Context, *BackoffOptions) time.Duration
	}

	// BackoffOptions 退避器选项
	BackoffOptions struct {
		// Attempts 重试次数
		Attempts int
	}
)

type customizedBackoff struct {
	backoffFn func(context.Context, *BackoffOptions) time.Duration
}

// NewBackoff 创建自定义时长的退避器
func NewBackoff(fn func(context.Context, *BackoffOptions) time.Duration) Backoff {
	return customizedBackoff{backoffFn: fn}
}

func (s customizedBackoff) Time(ctx context.Context, options *BackoffOptions) time.Duration {
	return s.backoffFn(ctx, options)
}

type fixedBackoff struct {
	wait time.Duration
}

// NewFixedBackoff 创建固定时长的退避器
func NewFixedBackoff(wait time.Duration) Backoff {
	return fixedBackoff{wait: wait}
}

func (s fixedBackoff) Time(context.Context, *BackoffOptions) time.Duration {
	return s.wait
}

type randomizedBackoff struct {
	base                        Backoff
	minification, magnification rational.Rational
	r                           *rand.Rand
	mutex                       sync.Mutex
}

// NewRandomizedBackoff 创建随机时长的退避器
func NewRandomizedBackoff(base Backoff, minification, magnification rational.Rational) Backoff {
	if minification.LessThanNum(0) {
		panic("minification must be greater than or equal to 0")
	}
	if magnification.LessThanNum(0) || magnification.GetNumerator() == 0 {
		panic("magnification must be greater than 0")
	}
	return &randomizedBackoff{
		base:          base,
		minification:  minification,
		magnification: magnification,
		r:             rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *randomizedBackoff) Time(ctx context.Context, opts *BackoffOptions) time.Duration {
	b := s.base.Time(ctx, opts)
	min := s.minification.MultiplyByNum(int64(b))
	max := s.magnification.MultiplyByNum(int64(b))
	diff := int64(max.Subtract(min).Float64())
	s.mutex.Lock()
	r := s.r.Int63n(diff)
	s.mutex.Unlock()
	return time.Duration(min.AddNum(r).Float64())
}

type limitedBackoff struct {
	base     Backoff
	min, max time.Duration
}

// NewLimitedBackoff 创建限制时长的退避器
func NewLimitedBackoff(base Backoff, min, max time.Duration) Backoff {
	return &limitedBackoff{
		base: base,
		min:  min,
		max:  max,
	}
}

func (s limitedBackoff) Time(ctx context.Context, opts *BackoffOptions) time.Duration {
	b := s.base.Time(ctx, opts)
	if b < s.min {
		return s.min
	} else if b > s.max {
		return s.max
	}
	return b
}

type exponentialBackoff struct {
	wait       time.Duration
	baseNumber int64
}

// NewExponentialBackoff 创建时长指数级增长的退避器
func NewExponentialBackoff(wait time.Duration, baseNumber int64) Backoff {
	return exponentialBackoff{wait: wait, baseNumber: baseNumber}
}

func (e exponentialBackoff) Time(ctx context.Context, opts *BackoffOptions) time.Duration {
	return e.wait * time.Duration(math.Pow(float64(e.baseNumber), float64(opts.Attempts)))
}
