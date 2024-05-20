//go:build unit
// +build unit

package backoff_test

import (
	"context"
	"testing"

	"github.com/alex-ant/gomath/rational"
	"github.com/qiniu/go-sdk/v7/storagev2/backoff"
)

func TestFixedBackoff(t *testing.T) {
	b := backoff.NewFixedBackoff(100)
	for i := 0; i < 1000; i++ {
		if b.Time(context.Background(), nil) != 100 {
			t.Fatal("unexpected")
		}
	}
}

func TestRandomizedBackoff(t *testing.T) {
	b := backoff.NewRandomizedBackoff(backoff.NewFixedBackoff(100), rational.New(1, 2), rational.New(3, 2))
	for i := 0; i < 1000; i++ {
		if wait := b.Time(context.Background(), nil); wait < 50 || wait > 150 {
			t.Fatal("unexpected")
		}
	}
}

func TestLimitedBackoff(t *testing.T) {
	b := backoff.NewLimitedBackoff(backoff.NewRandomizedBackoff(backoff.NewFixedBackoff(100), rational.New(1, 2), rational.New(3, 2)), 80, 120)
	for i := 0; i < 1000; i++ {
		if wait := b.Time(context.Background(), nil); wait < 80 || wait > 120 {
			t.Fatal("unexpected")
		}
	}
}

func TestExponentialBackoff(t *testing.T) {
	b := backoff.NewExponentialBackoff(100, 2)
	if b.Time(context.Background(), &backoff.BackoffOptions{Attempts: 0}) != 100 {
		t.Fatal("unexpected")
	}
	if b.Time(context.Background(), &backoff.BackoffOptions{Attempts: 1}) != 200 {
		t.Fatal("unexpected")
	}
	if b.Time(context.Background(), &backoff.BackoffOptions{Attempts: 2}) != 400 {
		t.Fatal("unexpected")
	}
	if b.Time(context.Background(), &backoff.BackoffOptions{Attempts: 3}) != 800 {
		t.Fatal("unexpected")
	}
	if b.Time(context.Background(), &backoff.BackoffOptions{Attempts: 4}) != 1600 {
		t.Fatal("unexpected")
	}
}
