package deployment

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/ishaf/cubit/internal/domain"
)

const (
	// DefaultPortRangeStart is the first assignable worker port.
	DefaultPortRangeStart = 9000
	// DefaultPortRangeEnd is the last assignable worker port (inclusive).
	DefaultPortRangeEnd = 9999
)

// AppPortLister lists applications with their assigned worker ports.
type AppPortLister interface {
	List(ctx context.Context) ([]*domain.Application, error)
}

// AppPortUpdater persists worker port assignments to applications.
type AppPortUpdater interface {
	GetByID(ctx context.Context, id string) (*domain.Application, error)
	Update(ctx context.Context, app *domain.Application) error
}

// PortAllocator manages discrete worker port assignments for applications.
// Each application receives a unique port from a configurable range.
type PortAllocator struct {
	lister     AppPortLister
	updater    AppPortUpdater
	rangeStart int
	rangeEnd   int
	mu         sync.Mutex
}

// NewPortAllocator creates a PortAllocator with the default port range [9000, 9999].
func NewPortAllocator(lister AppPortLister, updater AppPortUpdater) *PortAllocator {
	return NewPortAllocatorWithRange(lister, updater, DefaultPortRangeStart, DefaultPortRangeEnd)
}

// NewPortAllocatorWithRange creates a PortAllocator with a custom port range [start, end].
func NewPortAllocatorWithRange(lister AppPortLister, updater AppPortUpdater, rangeStart, rangeEnd int) *PortAllocator {
	if rangeStart <= 0 {
		rangeStart = DefaultPortRangeStart
	}
	if rangeEnd <= 0 || rangeEnd < rangeStart {
		rangeEnd = DefaultPortRangeEnd
	}
	return &PortAllocator{
		lister:     lister,
		updater:    updater,
		rangeStart: rangeStart,
		rangeEnd:   rangeEnd,
	}
}

// AllocatePort assigns the next available port to the given application.
// If the application already has a port assigned (WorkerPort > 0), it returns that port.
// Otherwise, it scans all existing applications to find the lowest unused port in the range.
func (pa *PortAllocator) AllocatePort(ctx context.Context, appID string) (int, error) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	// Check if the app already has a port
	app, err := pa.updater.GetByID(ctx, appID)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve application %s for port allocation: %w", appID, err)
	}
	if app.WorkerPort > 0 {
		return app.WorkerPort, nil
	}

	// Find all currently used ports
	apps, err := pa.lister.List(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list applications for port allocation: %w", err)
	}

	usedPorts := make(map[int]struct{})
	for _, a := range apps {
		if a.WorkerPort > 0 {
			usedPorts[a.WorkerPort] = struct{}{}
		}
	}

	// Find the lowest available port in the range
	for port := pa.rangeStart; port <= pa.rangeEnd; port++ {
		if _, used := usedPorts[port]; !used {
			app.WorkerPort = port
			if err := pa.updater.Update(ctx, app); err != nil {
				return 0, fmt.Errorf("failed to persist port %d for application %s: %w", port, appID, err)
			}
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports in range [%d, %d]: all %d ports are allocated",
		pa.rangeStart, pa.rangeEnd, pa.rangeEnd-pa.rangeStart+1)
}

// ReleasePort clears the worker port assignment for an application.
func (pa *PortAllocator) ReleasePort(ctx context.Context, appID string) error {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	app, err := pa.updater.GetByID(ctx, appID)
	if err != nil {
		return fmt.Errorf("failed to retrieve application %s for port release: %w", appID, err)
	}

	if app.WorkerPort == 0 {
		return nil // already released
	}

	app.WorkerPort = 0
	if err := pa.updater.Update(ctx, app); err != nil {
		return fmt.Errorf("failed to clear port for application %s: %w", appID, err)
	}
	return nil
}

// ListAllocations returns a sorted list of (appID, port) allocations.
func (pa *PortAllocator) ListAllocations(ctx context.Context) ([]PortAllocation, error) {
	apps, err := pa.lister.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list applications for port allocations: %w", err)
	}

	var allocations []PortAllocation
	for _, a := range apps {
		if a.WorkerPort > 0 {
			allocations = append(allocations, PortAllocation{
				AppID:   a.ID,
				AppName: a.Name,
				Port:    a.WorkerPort,
			})
		}
	}

	sort.Slice(allocations, func(i, j int) bool {
		return allocations[i].Port < allocations[j].Port
	})
	return allocations, nil
}

// PortAllocation records a worker port assignment.
type PortAllocation struct {
	AppID   string `json:"app_id"`
	AppName string `json:"app_name"`
	Port    int    `json:"port"`
}
