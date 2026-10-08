package runtime

import (
	"context"
	"sync"
)

// FakeWorkerExecutor provides an in-memory, thread-safe test double satisfying the WorkerExecutor interface.
type FakeWorkerExecutor struct {
	mu              sync.RWMutex
	invocations     []ExecutionPayload
	defaultResponse *WorkerExecutionResult
	err             error
	customHandler   func(ctx context.Context, payload ExecutionPayload) (*WorkerExecutionResult, error)
}

// NewFakeWorkerExecutor creates a new initialized FakeWorkerExecutor with a default 200 OK response.
func NewFakeWorkerExecutor() *FakeWorkerExecutor {
	return &FakeWorkerExecutor{
		defaultResponse: &WorkerExecutionResult{
			Status:     200,
			Headers:    map[string]string{"Content-Type": "text/plain"},
			Body:       []byte("fake worker response"),
			DurationMs: 1.0,
			CF: map[string]interface{}{
				"country": "US",
				"colo":    "SFO",
				"city":    "San Francisco",
				"asn":     13335,
			},
		},
	}
}

// Execute records the execution payload and returns the configured response or error.
func (f *FakeWorkerExecutor) Execute(ctx context.Context, payload ExecutionPayload) (*WorkerExecutionResult, error) {
	f.mu.Lock()
	f.invocations = append(f.invocations, payload)
	handler := f.customHandler
	defaultRes := f.defaultResponse
	forcedErr := f.err
	f.mu.Unlock()

	if handler != nil {
		return handler(ctx, payload)
	}

	if forcedErr != nil {
		return nil, forcedErr
	}

	return defaultRes, nil
}

// SetDefaultResponse configures the canned response returned by Execute.
func (f *FakeWorkerExecutor) SetDefaultResponse(res *WorkerExecutionResult) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.defaultResponse = res
}

// SetError configures a forced error to return on Execute.
func (f *FakeWorkerExecutor) SetError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

// SetHandler configures a custom callback for dynamic execution handling.
func (f *FakeWorkerExecutor) SetHandler(handler func(ctx context.Context, payload ExecutionPayload) (*WorkerExecutionResult, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.customHandler = handler
}

// GetInvocations returns a copy of all recorded execution payloads.
func (f *FakeWorkerExecutor) GetInvocations() []ExecutionPayload {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make([]ExecutionPayload, len(f.invocations))
	copy(out, f.invocations)
	return out
}

// Reset clears recorded invocations, errors, and custom handlers.
func (f *FakeWorkerExecutor) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.invocations = nil
	f.err = nil
	f.customHandler = nil
}
