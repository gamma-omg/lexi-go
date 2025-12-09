package main

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gamma-omg/lexi-go/internal/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Setenv("AUTH_SECRET", "secret")

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	healthCh := make(chan bool, 1)
	readyCh := make(chan bool, 1)
	go func() {
		errCh <- run(ctx)
	}()

	go func() {
		readyCh <- testutil.WaitFor(t, ctx, 500*time.Millisecond, func() bool {
			resp, err := http.Get("http://localhost:8080/readyz")
			if err != nil {
				return false
			}
			_ = resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		})
	}()

	go func() {
		healthCh <- testutil.WaitFor(t, ctx, 500*time.Millisecond, func() bool {
			resp, err := http.Get("http://localhost:8080/healthz")
			if err != nil {
				return false
			}
			_ = resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		})
	}()

	var isHealthy, isReady bool
	for !isHealthy || !isReady {
		select {
		case err := <-errCh:
			require.NoError(t, err)
			return
		case isReady = <-readyCh:
			require.True(t, isReady)
		case isHealthy = <-healthCh:
			require.True(t, isHealthy)
		case <-ctx.Done():
			t.Fatal("test timed out")
		}
	}
}

func TestRun_Cancel(t *testing.T) {
	t.Setenv("AUTH_SECRET", "secret")

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx)
	}()

	go func() {
		time.Sleep(2 * time.Second)
		cancel()
	}()

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("test timed out")
	}
}
