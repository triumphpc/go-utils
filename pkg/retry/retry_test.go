package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetry_SuccessOnFirstAttempt(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
	}

	called := 0
	op := func() (string, error) {
		called++
		return "success", nil
	}

	result, err := Retry(ctx, config, op)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "success" {
		t.Errorf("expected 'success', got %v", result)
	}
	if called != 1 {
		t.Errorf("expected 1 call, got %d", called)
	}
}

func TestRetry_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
	}

	called := 0
	op := func() (string, error) {
		called++
		if called < 2 {
			return "", errors.New("temporary error")
		}
		return "success", nil
	}

	result, err := Retry(ctx, config, op)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "success" {
		t.Errorf("expected 'success', got %v", result)
	}
	if called != 2 {
		t.Errorf("expected 2 calls, got %d", called)
	}
}

func TestRetry_MaxAttemptsReached(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
	}

	expectedErr := errors.New("operation failed")
	called := 0
	op := func() (string, error) {
		called++
		return "", expectedErr
	}

	_, err := Retry(ctx, config, op)

	if err != expectedErr {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
	if called != 3 {
		t.Errorf("expected 3 calls, got %d", called)
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := Config{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
	}

	called := 0
	op := func() (string, error) {
		called++
		if called == 2 {
			cancel()
		}
		return "", errors.New("operation failed")
	}

	start := time.Now()
	_, err := Retry(ctx, config, op)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if called != 2 {
		t.Errorf("expected 2 calls, got %d", called)
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("cancellation took too long: %v", elapsed)
	}
}

func TestRetry_DelayWithJitter(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
	}

	called := 0
	op := func() (string, error) {
		called++
		return "", errors.New("operation failed")
	}

	start := time.Now()
	Retry(ctx, config, op)
	elapsed := time.Since(start)

	// Проверяем, что общее время выполнения соответствует ожидаемым задержкам
	// с учетом джиттера (100ms + ~200ms + ~400ms = ~700ms)
	minExpected := 600 * time.Millisecond
	maxExpected := 800 * time.Millisecond
	if elapsed < minExpected || elapsed > maxExpected {
		t.Errorf("elapsed time %v outside expected range (%v-%v)", elapsed, minExpected, maxExpected)
	}
}

func TestRetry_MaxDelayRespected(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxAttempts:  4,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     1 * time.Second,
	}

	called := 0
	op := func() (string, error) {
		called++
		return "", errors.New("operation failed")
	}

	start := time.Now()
	Retry(ctx, config, op)
	elapsed := time.Since(start)

	// Ожидаемые задержки:
	// 1 попытка: 500ms (initial) + jitter (~0-500ms)
	// 2 попытка: min(1s + jitter, 1s)
	// 3 попытка: min(1s + jitter, 1s)
	// Общее время: ~500ms-1s + ~1s + ~1s = ~2.5s-3s

	minExpected := 2 * time.Second
	maxExpected := 3 * time.Second
	if elapsed < minExpected {
		t.Errorf("elapsed time %v is less than minimum expected %v", elapsed, minExpected)
	}
	if elapsed > maxExpected {
		t.Errorf("elapsed time %v is greater than maximum expected %v", elapsed, maxExpected)
	}
}
