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

type fixedBackoff struct {
	wait time.Duration
}

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

func NewExponentialBackoff(wait time.Duration, baseNumber int64) Backoff {
	return exponentialBackoff{wait: wait, baseNumber: baseNumber}
}

func (e exponentialBackoff) Time(ctx context.Context, opts *BackoffOptions) time.Duration {
	return e.wait * time.Duration(math.Pow(float64(e.baseNumber), float64(opts.Attempts)))
}
