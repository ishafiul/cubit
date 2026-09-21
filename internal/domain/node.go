package domain

import (
	"net"
	"strings"
	"time"
)

// NodeStatus represents the operational lifecycle state of a fleet node.
type NodeStatus string

const (
	NodeStatusActive   NodeStatus = "active"
	NodeStatusDraining NodeStatus = "draining"
	NodeStatusOffline  NodeStatus = "offline"
)

// DefaultCelldVersion represents the current stable release of the celld runtime.
const DefaultCelldVersion = "0.5.1"

// NodeSpecs stores telemetry and capacity specifications for a node.
type NodeSpecs struct {
	CPUCores      int
	MemoryBytes   int64
	DiskFreeBytes int64
}

// Node represents a bare-metal server participating in the celld fleet.
type Node struct {
	ID           string
	Name         string
	IPAddress    string
	InternalPort int
	WorkerPort   int
	Status       NodeStatus
	CelldVersion string
	Specs        NodeSpecs
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewNode constructs and validates a new Node entity.
func NewNode(id, name, ipAddress string, internalPort, workerPort int, celldVersion string) (*Node, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, NewValidationError("node name cannot be empty")
	}

	ipAddress = strings.TrimSpace(ipAddress)
	if ipAddress == "" {
		return nil, NewValidationError("node IP address cannot be empty")
	}
	if parsed := net.ParseIP(ipAddress); parsed == nil {
		return nil, NewValidationError("invalid node IP address format")
	}

	if internalPort <= 0 || internalPort > 65535 {
		return nil, NewValidationError("internal port must be between 1 and 65535")
	}

	if workerPort <= 0 || workerPort > 65535 {
		return nil, NewValidationError("worker port must be between 1 and 65535")
	}

	if internalPort == workerPort {
		return nil, NewValidationError("internal port and worker port cannot be the same")
	}

	now := time.Now().UTC()
	return &Node{
		ID:           id,
		Name:         name,
		IPAddress:    ipAddress,
		InternalPort: internalPort,
		WorkerPort:   workerPort,
		Status:       NodeStatusActive,
		CelldVersion: celldVersion,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// Drain marks a node as draining to allow its active Durable Object cell leases to migrate.
func (n *Node) Drain() error {
	if n.Status != NodeStatusActive {
		return NewInvalidStateError("only active nodes can be transitioned to draining")
	}
	n.Status = NodeStatusDraining
	n.UpdatedAt = time.Now().UTC()
	return nil
}

// MarkActive marks a node as ready to accept new work.
func (n *Node) MarkActive() error {
	n.Status = NodeStatusActive
	n.UpdatedAt = time.Now().UTC()
	return nil
}

// MarkOffline marks a node as offline.
func (n *Node) MarkOffline() {
	n.Status = NodeStatusOffline
	n.UpdatedAt = time.Now().UTC()
}

// UpdateSpecs updates the telemetry information for the node.
func (n *Node) UpdateSpecs(specs NodeSpecs) {
	n.Specs = specs
	n.UpdatedAt = time.Now().UTC()
}

// UpdateCelldVersion records an upgraded celld daemon version.
func (n *Node) UpdateCelldVersion(version string) {
	n.CelldVersion = strings.TrimSpace(version)
	n.UpdatedAt = time.Now().UTC()
}
