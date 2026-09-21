package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/usecase"
)

var _ usecase.NodeRepository = (*NodeRepo)(nil)

// NodeRepo implements usecase.NodeRepository using SQLite.
type NodeRepo struct {
	db *sql.DB
}

// NewNodeRepo creates a new NodeRepo.
func NewNodeRepo(db *sql.DB) *NodeRepo {
	return &NodeRepo{db: db}
}

// Save persists a new node.
func (r *NodeRepo) Save(ctx context.Context, n *domain.Node) error {
	query := `
		INSERT INTO nodes (id, name, ip_address, internal_port, worker_port, status, celld_version, cpu_cores, memory_bytes, disk_free_bytes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		n.ID, n.Name, n.IPAddress, n.InternalPort, n.WorkerPort, string(n.Status), n.CelldVersion,
		n.Specs.CPUCores, n.Specs.MemoryBytes, n.Specs.DiskFreeBytes,
		n.CreatedAt.Format(time.RFC3339), n.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return err
	}
	return nil
}

// GetByID fetches a node by ID.
func (r *NodeRepo) GetByID(ctx context.Context, id string) (*domain.Node, error) {
	query := `
		SELECT id, name, ip_address, internal_port, worker_port, status, celld_version, cpu_cores, memory_bytes, disk_free_bytes, created_at, updated_at
		FROM nodes WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var (
		n                                    domain.Node
		statusStr, createdAtStr, updatedAtStr string
	)

	err := row.Scan(
		&n.ID, &n.Name, &n.IPAddress, &n.InternalPort, &n.WorkerPort, &statusStr, &n.CelldVersion,
		&n.Specs.CPUCores, &n.Specs.MemoryBytes, &n.Specs.DiskFreeBytes,
		&createdAtStr, &updatedAtStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.NewNotFoundError("node with id " + id + " not found")
	}
	if err != nil {
		return nil, err
	}

	n.Status = domain.NodeStatus(statusStr)
	n.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	n.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	return &n, nil
}

// List returns all registered nodes.
func (r *NodeRepo) List(ctx context.Context) ([]*domain.Node, error) {
	query := `
		SELECT id, name, ip_address, internal_port, worker_port, status, celld_version, cpu_cores, memory_bytes, disk_free_bytes, created_at, updated_at
		FROM nodes ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*domain.Node
	for rows.Next() {
		var (
			n                                    domain.Node
			statusStr, createdAtStr, updatedAtStr string
		)
		if err := rows.Scan(
			&n.ID, &n.Name, &n.IPAddress, &n.InternalPort, &n.WorkerPort, &statusStr, &n.CelldVersion,
			&n.Specs.CPUCores, &n.Specs.MemoryBytes, &n.Specs.DiskFreeBytes,
			&createdAtStr, &updatedAtStr,
		); err != nil {
			return nil, err
		}
		n.Status = domain.NodeStatus(statusStr)
		n.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		n.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		nodes = append(nodes, &n)
	}
	return nodes, rows.Err()
}

// Delete removes a node by ID.
func (r *NodeRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM nodes WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.NewNotFoundError("node not found")
	}
	return nil
}

// Update updates an existing node.
func (r *NodeRepo) Update(ctx context.Context, n *domain.Node) error {
	query := `
		UPDATE nodes
		SET name = ?, ip_address = ?, internal_port = ?, worker_port = ?, status = ?, celld_version = ?, cpu_cores = ?, memory_bytes = ?, disk_free_bytes = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query,
		n.Name, n.IPAddress, n.InternalPort, n.WorkerPort, string(n.Status), n.CelldVersion,
		n.Specs.CPUCores, n.Specs.MemoryBytes, n.Specs.DiskFreeBytes,
		time.Now().UTC().Format(time.RFC3339), n.ID,
	)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.NewNotFoundError("node not found")
	}
	return nil
}
