package runtime_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/ishaf/cubit/internal/modules/runtime"
)

func TestFakeWorkerExecutor(t *testing.T) {
	ctx := context.Background()

	t.Run("Given a FakeWorkerExecutor When Execute is called Then returns default 200 response and records invocation", func(t *testing.T) {
		fake := runtime.NewFakeWorkerExecutor()
		payload := runtime.ExecutionPayload{
			Method: "POST",
			Path:   "/api/data",
			Body:   []byte(`{"key":"value"}`),
		}

		res, err := fake.Execute(ctx, payload)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != 200 {
			t.Errorf("expected status 200, got %d", res.Status)
		}
		if string(res.Body) != "fake worker response" {
			t.Errorf("expected body 'fake worker response', got %q", string(res.Body))
		}

		invocations := fake.GetInvocations()
		if len(invocations) != 1 {
			t.Fatalf("expected 1 invocation, got %d", len(invocations))
		}
		if invocations[0].Path != "/api/data" || invocations[0].Method != "POST" {
			t.Errorf("expected recorded path and method to match, got %s %s", invocations[0].Method, invocations[0].Path)
		}
	})

	t.Run("Given a FakeWorkerExecutor When SetError is configured Then Execute returns the error", func(t *testing.T) {
		fake := runtime.NewFakeWorkerExecutor()
		expectedErr := errors.New("simulated runtime crash")
		fake.SetError(expectedErr)

		_, err := fake.Execute(ctx, runtime.ExecutionPayload{})
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})

	t.Run("Given a FakeWorkerExecutor When SetHandler is configured Then custom handler produces the result", func(t *testing.T) {
		fake := runtime.NewFakeWorkerExecutor()
		fake.SetHandler(func(ctx context.Context, payload runtime.ExecutionPayload) (*runtime.WorkerExecutionResult, error) {
			return &runtime.WorkerExecutionResult{
				Status: 201,
				Body:   []byte("created: " + payload.Path),
			}, nil
		})

		res, err := fake.Execute(ctx, runtime.ExecutionPayload{Path: "/custom"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != 201 || string(res.Body) != "created: /custom" {
			t.Errorf("unexpected result: status=%d, body=%s", res.Status, string(res.Body))
		}
	})

	t.Run("Given a FakeWorkerExecutor When invoked concurrently Then operations are thread-safe", func(t *testing.T) {
		fake := runtime.NewFakeWorkerExecutor()
		var wg sync.WaitGroup
		concurrency := 20

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				_, _ = fake.Execute(ctx, runtime.ExecutionPayload{Path: "/concurrent"})
			}(i)
		}

		wg.Wait()
		if len(fake.GetInvocations()) != concurrency {
			t.Errorf("expected %d invocations, got %d", concurrency, len(fake.GetInvocations()))
		}
	})
}
