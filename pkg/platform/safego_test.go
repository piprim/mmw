package platform_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/piprim/mmw/pkg/platform"
	"github.com/stretchr/testify/assert"
)

func TestSafeGo_NormalExecution(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	called := false
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	platform.SafeGo(context.Background(), logger, func() {
		defer wg.Done()
		called = true
	})

	wg.Wait()
	assert.True(t, called)
}

func TestSafeGo_PanicWithError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))

	done := make(chan struct{})

	platform.SafeGo(context.Background(), logger, func() {
		defer close(done)
		panic(errors.New("boom"))
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("goroutine did not complete")
	}

	// The recover defer runs after fn()'s defers; give it a moment to log.
	time.Sleep(10 * time.Millisecond)
	assert.Contains(t, buf.String(), "background worker crashed")
}

func TestSafeGo_PanicWithNonError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))

	done := make(chan struct{})

	platform.SafeGo(context.Background(), logger, func() {
		defer close(done)
		panic("unexpected string panic")
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("goroutine did not complete")
	}

	time.Sleep(10 * time.Millisecond)
	assert.Contains(t, buf.String(), "background worker crashed")
}

func TestSafeGo_NoPanic_NoLog(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))

	var wg sync.WaitGroup
	wg.Add(1)

	platform.SafeGo(context.Background(), logger, func() {
		defer wg.Done()
		// no panic
	})

	wg.Wait()
	assert.Empty(t, buf.String())
}
