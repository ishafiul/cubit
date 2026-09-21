package node

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ishaf/cubit/internal/domain"
)

// Repository defines data access for fleet nodes.
type Repository interface {
	Save(ctx context.Context, n *domain.Node) error
	GetByID(ctx context.Context, id string) (*domain.Node, error)
	List(ctx context.Context) ([]*domain.Node, error)
	Update(ctx context.Context, n *domain.Node) error
	Delete(ctx context.Context, id string) error
}

// SQLiteRepository implements Repository using SQLite.
type SQLiteRepository struct {
	db *sql.DB
}

// NewRepository creates a new SQLite node repository.
func NewRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// Save persists a new node.
func (r *SQLiteRepository) Save(ctx context.Context, n *domain.Node) error {
	query := `
		INSERT INTO nodes (id, name, ip_address, internal_port, worker_port, status, celld_version, cpu_cores, memory_bytes, disk_free_bytes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		n.ID, n.Name, n.IPAddress, n.InternalPort, n.WorkerPort, string(n.Status), n.CelldVersion,
		n.Specs.CPUCores, n.Specs.MemoryBytes, n.Specs.DiskFreeBytes,
		n.CreatedAt.Format(time.RFC3339), n.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

// GetByID fetches a node by ID.
func (r *SQLiteRepository) GetByID(ctx context.Context, id string) (*domain.Node, error) {
	query := `
		SELECT id, name, ip_address, internal_port, worker_port, status, celld_version, cpu_cores, memory_bytes, disk_free_bytes, created_at, updated_at
		FROM nodes WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var (
		n                                     domain.Node
		statusStr, createdAtStr, updatedAtStr string
	)

	err := row.Scan(
		&n.ID, &n.Name, &n.IPAddress, &n.InternalPort, &n.WorkerPort, &statusStr, &n.CelldVersion,
		&n.Specs.CPUCores, &n.Specs.MemoryBytes, &n.Specs.DiskFreeBytes,
		&createdAtStr, &updatedAtStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.NewNotFoundError("node not found: " + id)
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
func (r *SQLiteRepository) List(ctx context.Context) ([]*domain.Node, error) {
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
			n                                     domain.Node
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

// Update updates an existing node.
func (r *SQLiteRepository) Update(ctx context.Context, n *domain.Node) error {
	query := `
		UPDATE nodes
		SET name = ?, ip_address = ?, internal_port = ?, worker_port = ?, status = ?, celld_version = ?,
		    cpu_cores = ?, memory_bytes = ?, disk_free_bytes = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query,
		n.Name, n.IPAddress, n.InternalPort, n.WorkerPort, string(n.Status), n.CelldVersion,
		n.Specs.CPUCores, n.Specs.MemoryBytes, n.Specs.DiskFreeBytes,
		n.UpdatedAt.Format(time.RFC3339), n.ID,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewNotFoundError("node not found: " + n.ID)
	}
	return nil
}

// Delete removes a node.
func (r *SQLiteRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM nodes WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewNotFoundError("node not found: " + id)
	}
	return nil
}
