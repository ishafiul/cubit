package runtime_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ishaf/cubit/internal/modules/runtime"
)

func TestNodeWorkerExecutor(t *testing.T) {
	executor := runtime.NewNodeWorkerExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("Given a NodeWorkerExecutor When executing a simple fetch worker Then it returns status 200 and expected body", func(t *testing.T) {
		bundle := []byte(`
export default {
    fetch(request, env, ctx) {
        return new Response("Hello from runtime isolate!", {
            status: 200,
            headers: { "x-custom-res": "cubit-val" }
        });
    }
};
`)
		payload := runtime.ExecutionPayload{
			Bundle:  bundle,
			Method:  "GET",
			Path:    "/test",
			Headers: map[string]string{"x-request-test": "ping"},
			Body:    nil,
		}

		res, err := executor.Execute(ctx, payload)
		if err != nil {
			t.Fatalf("unexpected execution error: %v", err)
		}
		if res.Status != 200 {
			t.Errorf("expected status 200, got %d", res.Status)
		}
		if string(res.Body) != "Hello from runtime isolate!" {
			t.Errorf("expected body 'Hello from runtime isolate!', got %q", string(res.Body))
		}
		if res.Headers["x-custom-res"] != "cubit-val" {
			t.Errorf("expected header x-custom-res: cubit-val, got %q", res.Headers["x-custom-res"])
		}
		if res.DurationMs < 0 {
			t.Errorf("expected positive duration, got %f", res.DurationMs)
		}
	})

	t.Run("Given a NodeWorkerExecutor When worker logs to console Then console logs are captured", func(t *testing.T) {
		bundle := []byte(`
export default {
    fetch(request, env, ctx) {
        console.log("standard log message");
        console.warn("warning message");
        console.error("error message");
        return new Response("ok", { status: 200 });
    }
};
`)
		payload := runtime.ExecutionPayload{
			Bundle: bundle,
			Method: "GET",
			Path:   "/logs",
		}

		res, err := executor.Execute(ctx, payload)
		if err != nil {
			t.Fatalf("unexpected execution error: %v", err)
		}
		if res.Status != 200 {
			t.Fatalf("expected status 200, got %d", res.Status)
		}
		if len(res.Logs) < 3 {
			t.Fatalf("expected at least 3 log entries, got %d", len(res.Logs))
		}

		foundLog, foundWarn, foundErr := false, false, false
		for _, l := range res.Logs {
			if l.Level == "log" && strings.Contains(l.Message, "standard log message") {
				foundLog = true
			}
			if l.Level == "warn" && strings.Contains(l.Message, "warning message") {
				foundWarn = true
			}
			if l.Level == "error" && strings.Contains(l.Message, "error message") {
				foundErr = true
			}
		}

		if !foundLog || !foundWarn || !foundErr {
			t.Errorf("expected all log levels captured: log=%v, warn=%v, error=%v", foundLog, foundWarn, foundErr)
		}
	})

	t.Run("Given a NodeWorkerExecutor When worker throws an unhandled error Then returns 500 with exception details", func(t *testing.T) {
		bundle := []byte(`
export default {
    fetch(request, env, ctx) {
        throw new Error("fatal isolate exception");
    }
};
`)
		payload := runtime.ExecutionPayload{
			Bundle: bundle,
			Method: "POST",
			Path:   "/fail",
		}

		res, err := executor.Execute(ctx, payload)
		if err != nil {
			t.Fatalf("unexpected Go error: %v", err)
		}
		if res.Status != 500 {
			t.Errorf("expected status 500, got %d", res.Status)
		}
		if len(res.Exceptions) == 0 {
			t.Errorf("expected captured exceptions, got 0")
		} else if !strings.Contains(res.Exceptions[0], "fatal isolate exception") {
			t.Errorf("expected exception text, got %q", res.Exceptions[0])
		}
	})

	t.Run("Given a NodeWorkerExecutor When worker uses Cache API Then match and put succeed", func(t *testing.T) {
		bundle := []byte(`
export default {
    async fetch(request, env, ctx) {
        const cache = caches.default;
        const cacheKey = "http://localhost/cached-item";
        let cached = await cache.match(cacheKey);
        if (!cached) {
            await cache.put(cacheKey, new Response("from-cache", { status: 200, headers: { "x-cached": "true" } }));
            return new Response("miss", { status: 200 });
        }
        return cached;
    }
};
`)
		payload := runtime.ExecutionPayload{
			Bundle: bundle,
			Method: "GET",
			Path:   "/cache",
		}

		res, err := executor.Execute(ctx, payload)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != 200 || string(res.Body) != "miss" {
			t.Errorf("expected first call to miss, got status=%d body=%s", res.Status, string(res.Body))
		}
	})
}
