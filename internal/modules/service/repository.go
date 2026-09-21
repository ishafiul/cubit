package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ishaf/cubit/internal/domain"
)

// ServicesRepo handles persistence for the celld service suite.
// Repository handles persistence for the celld service suite.
type Repository interface {
	// KV
	SaveKVNamespace(ctx context.Context, ns *domain.KVNamespace) error
	ListKVNamespaces(ctx context.Context) ([]*domain.KVNamespace, error)
	GetKVNamespace(ctx context.Context, id string) (*domain.KVNamespace, error)
	DeleteKVNamespace(ctx context.Context, id string) error
	PutKVPair(ctx context.Context, pair *domain.KVPair) error
	GetKVPair(ctx context.Context, namespaceID, key string) (*domain.KVPair, error)
	ListKVPairs(ctx context.Context, namespaceID string) ([]*domain.KVPair, error)
	DeleteKVPair(ctx context.Context, namespaceID, key string) error

	// D1
	SaveD1Database(ctx context.Context, db *domain.D1Database) error
	ListD1Databases(ctx context.Context) ([]*domain.D1Database, error)
	GetD1Database(ctx context.Context, id string) (*domain.D1Database, error)
	DeleteD1Database(ctx context.Context, id string) error
	ExecuteD1Query(ctx context.Context, id, query string) (*domain.D1QueryResult, error)

	// Queues
	SaveQueue(ctx context.Context, q *domain.Queue) error
	ListQueues(ctx context.Context) ([]*domain.Queue, error)
	DeleteQueue(ctx context.Context, id string) error
	EnqueueMessage(ctx context.Context, msg *domain.QueueMessage) error
	ListQueueMessages(ctx context.Context, queueID string, limit int) ([]*domain.QueueMessage, error)

	// Cron Triggers
	SaveCronTrigger(ctx context.Context, trigger *domain.CronTrigger) error
	ListCronTriggers(ctx context.Context) ([]*domain.CronTrigger, error)
	DeleteCronTrigger(ctx context.Context, id string) error
	RecordCronRun(ctx context.Context, run *domain.CronRun) error
	ListCronRuns(ctx context.Context, triggerID string) ([]*domain.CronRun, error)

	// Workflows
	SaveWorkflow(ctx context.Context, wf *domain.Workflow) error
	ListWorkflows(ctx context.Context) ([]*domain.Workflow, error)
	DeleteWorkflow(ctx context.Context, id string) error
	SaveWorkflowRun(ctx context.Context, run *domain.WorkflowRun) error
	ListWorkflowRuns(ctx context.Context, workflowID string) ([]*domain.WorkflowRun, error)

	// Durable Objects
	SaveDurableObjectClass(ctx context.Context, doc *domain.DurableObjectClass) error
	ListDurableObjectClasses(ctx context.Context) ([]*domain.DurableObjectClass, error)
	DeleteDurableObjectClass(ctx context.Context, id string) error
	SaveDOInstance(ctx context.Context, inst *domain.DurableObjectInstance) error
	ListDOInstances(ctx context.Context, classID string) ([]*domain.DurableObjectInstance, error)

	// Containers
	SaveContainer(ctx context.Context, ct *domain.ContainerWorkload) error
	ListContainers(ctx context.Context) ([]*domain.ContainerWorkload, error)
	DeleteContainer(ctx context.Context, id string) error

	// Static Assets
	SaveStaticSite(ctx context.Context, site *domain.StaticSite) error
	ListStaticSites(ctx context.Context) ([]*domain.StaticSite, error)
	DeleteStaticSite(ctx context.Context, id string) error
}

type SQLiteRepository struct {
	db      *sql.DB
	baseDir string
}

// NewServicesRepo constructs a new ServicesRepo.
func NewRepository(database *sql.DB, baseDir string) *SQLiteRepository {
	if baseDir == "" {
		baseDir = ".data"
	}
	_ = os.MkdirAll(filepath.Join(baseDir, "d1"), 0755)
	return &SQLiteRepository{
		db:      database,
		baseDir: baseDir,
	}
}

// -----------------------------------------------------------------------------
// KV Operations
// -----------------------------------------------------------------------------

func (r *SQLiteRepository) SaveKVNamespace(ctx context.Context, ns *domain.KVNamespace) error {
	query := `INSERT INTO kv_namespaces (id, name, created_at) VALUES (?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, ns.ID, ns.Name, ns.CreatedAt.Format(time.RFC3339))
	return err
}

func (r *SQLiteRepository) ListKVNamespaces(ctx context.Context) ([]*domain.KVNamespace, error) {
	query := `
		SELECT n.id, n.name, n.created_at, COUNT(e.key) as key_count
		FROM kv_namespaces n
		LEFT JOIN kv_entries e ON n.id = e.namespace_id
		GROUP BY n.id, n.name, n.created_at
		ORDER BY n.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var namespaces []*domain.KVNamespace
	for rows.Next() {
		var ns domain.KVNamespace
		var createdAtStr string
		if err := rows.Scan(&ns.ID, &ns.Name, &createdAtStr, &ns.KeyCount); err != nil {
			return nil, err
		}
		ns.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		namespaces = append(namespaces, &ns)
	}
	return namespaces, nil
}

func (r *SQLiteRepository) GetKVNamespace(ctx context.Context, id string) (*domain.KVNamespace, error) {
	query := `
		SELECT n.id, n.name, n.created_at, COUNT(e.key) as key_count
		FROM kv_namespaces n
		LEFT JOIN kv_entries e ON n.id = e.namespace_id
		WHERE n.id = ?
		GROUP BY n.id, n.name, n.created_at
	`
	var ns domain.KVNamespace
	var createdAtStr string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&ns.ID, &ns.Name, &createdAtStr, &ns.KeyCount)
	if err == sql.ErrNoRows {
		return nil, domain.NewNotFoundError("KV namespace not found")
	}
	if err != nil {
		return nil, err
	}
	ns.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	return &ns, nil
}

func (r *SQLiteRepository) DeleteKVNamespace(ctx context.Context, id string) error {
	query := `DELETE FROM kv_namespaces WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *SQLiteRepository) PutKVPair(ctx context.Context, pair *domain.KVPair) error {
	query := `
		INSERT INTO kv_entries (namespace_id, key, value, expiration_ttl, metadata, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(namespace_id, key) DO UPDATE SET
			value = excluded.value,
			expiration_ttl = excluded.expiration_ttl,
			metadata = excluded.metadata,
			updated_at = excluded.updated_at
	`
	_, err := r.db.ExecContext(ctx, query,
		pair.NamespaceID, pair.Key, pair.Value, pair.ExpirationTTL, pair.Metadata, pair.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *SQLiteRepository) GetKVPair(ctx context.Context, namespaceID, key string) (*domain.KVPair, error) {
	query := `SELECT namespace_id, key, value, expiration_ttl, metadata, updated_at FROM kv_entries WHERE namespace_id = ? AND key = ?`
	var pair domain.KVPair
	var updatedAtStr string
	err := r.db.QueryRowContext(ctx, query, namespaceID, key).Scan(
		&pair.NamespaceID, &pair.Key, &pair.Value, &pair.ExpirationTTL, &pair.Metadata, &updatedAtStr,
	)
	if err == sql.ErrNoRows {
		return nil, domain.NewNotFoundError("KV key not found")
	}
	if err != nil {
		return nil, err
	}
	pair.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	return &pair, nil
}

func (r *SQLiteRepository) ListKVPairs(ctx context.Context, namespaceID string) ([]*domain.KVPair, error) {
	query := `SELECT namespace_id, key, value, expiration_ttl, metadata, updated_at FROM kv_entries WHERE namespace_id = ? ORDER BY key ASC`
	rows, err := r.db.QueryContext(ctx, query, namespaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pairs []*domain.KVPair
	for rows.Next() {
		var pair domain.KVPair
		var updatedAtStr string
		if err := rows.Scan(&pair.NamespaceID, &pair.Key, &pair.Value, &pair.ExpirationTTL, &pair.Metadata, &updatedAtStr); err != nil {
			return nil, err
		}
		pair.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		pairs = append(pairs, &pair)
	}
	return pairs, nil
}

func (r *SQLiteRepository) DeleteKVPair(ctx context.Context, namespaceID, key string) error {
	query := `DELETE FROM kv_entries WHERE namespace_id = ? AND key = ?`
	_, err := r.db.ExecContext(ctx, query, namespaceID, key)
	return err
}

// -----------------------------------------------------------------------------
// D1 Operations (Serverless SQLite)
// -----------------------------------------------------------------------------

func (r *SQLiteRepository) SaveD1Database(ctx context.Context, d1 *domain.D1Database) error {
	query := `INSERT INTO d1_databases (id, name, size_bytes, created_at) VALUES (?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, d1.ID, d1.Name, d1.SizeBytes, d1.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return err
	}

	// Initialize database file
	dbPath := filepath.Join(r.baseDir, "d1", fmt.Sprintf("%s.db", d1.ID))
	subDB, err := sql.Open("sqlite", dbPath)
	if err == nil {
		_, _ = subDB.Exec("CREATE TABLE IF NOT EXISTS _d1_migrations (id INTEGER PRIMARY KEY, name TEXT, applied_at TIMESTAMP)")
		_ = subDB.Close()
	}
	return nil
}

func (r *SQLiteRepository) ListD1Databases(ctx context.Context) ([]*domain.D1Database, error) {
	query := `SELECT id, name, size_bytes, created_at FROM d1_databases ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []*domain.D1Database
	for rows.Next() {
		var d1 domain.D1Database
		var createdAtStr string
		if err := rows.Scan(&d1.ID, &d1.Name, &d1.SizeBytes, &createdAtStr); err != nil {
			return nil, err
		}
		d1.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)

		// Check file size
		dbPath := filepath.Join(r.baseDir, "d1", fmt.Sprintf("%s.db", d1.ID))
		if fi, err := os.Stat(dbPath); err == nil {
			d1.SizeBytes = fi.Size()
		}
		databases = append(databases, &d1)
	}
	return databases, nil
}

func (r *SQLiteRepository) GetD1Database(ctx context.Context, id string) (*domain.D1Database, error) {
	query := `SELECT id, name, size_bytes, created_at FROM d1_databases WHERE id = ?`
	var d1 domain.D1Database
	var createdAtStr string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&d1.ID, &d1.Name, &d1.SizeBytes, &createdAtStr)
	if err == sql.ErrNoRows {
		return nil, domain.NewNotFoundError("D1 database not found")
	}
	if err != nil {
		return nil, err
	}
	d1.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)

	dbPath := filepath.Join(r.baseDir, "d1", fmt.Sprintf("%s.db", d1.ID))
	if fi, err := os.Stat(dbPath); err == nil {
		d1.SizeBytes = fi.Size()
	}
	return &d1, nil
}

func (r *SQLiteRepository) DeleteD1Database(ctx context.Context, id string) error {
	query := `DELETE FROM d1_databases WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	dbPath := filepath.Join(r.baseDir, "d1", fmt.Sprintf("%s.db", id))
	_ = os.Remove(dbPath)
	return nil
}

func (r *SQLiteRepository) ExecuteD1Query(ctx context.Context, id, queryStr string) (*domain.D1QueryResult, error) {
	dbPath := filepath.Join(r.baseDir, "d1", fmt.Sprintf("%s.db", id))
	subDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed opening D1 database: %w", err)
	}
	defer subDB.Close()

	start := time.Now()
	queryStr = strings.TrimSpace(queryStr)

	// If query is a SELECT or PRAGMA
	isQuery := strings.HasPrefix(strings.ToUpper(queryStr), "SELECT") || strings.HasPrefix(strings.ToUpper(queryStr), "PRAGMA")
	if isQuery {
		rows, err := subDB.QueryContext(ctx, queryStr)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return nil, err
		}

		var results []map[string]interface{}
		for rows.Next() {
			vals := make([]interface{}, len(columns))
			valPtrs := make([]interface{}, len(columns))
			for i := range vals {
				valPtrs[i] = &vals[i]
			}
			if err := rows.Scan(valPtrs...); err != nil {
				return nil, err
			}
			rowMap := make(map[string]interface{})
			for i, col := range columns {
				val := vals[i]
				if b, ok := val.([]byte); ok {
					rowMap[col] = string(b)
				} else {
					rowMap[col] = val
				}
			}
			results = append(results, rowMap)
		}
		duration := time.Since(start).Seconds() * 1000
		return &domain.D1QueryResult{
			Columns:      columns,
			Rows:         results,
			RowsAffected: int64(len(results)),
			DurationMs:   duration,
		}, nil
	}

	// For DDL / INSERT / UPDATE
	res, err := subDB.ExecContext(ctx, queryStr)
	if err != nil {
		return nil, err
	}
	affected, _ := res.RowsAffected()
	duration := time.Since(start).Seconds() * 1000
	return &domain.D1QueryResult{
		Columns:      []string{"status"},
		Rows:         []map[string]interface{}{{"status": "ok"}},
		RowsAffected: affected,
		DurationMs:   duration,
	}, nil
}

// -----------------------------------------------------------------------------
// Queues Operations
// -----------------------------------------------------------------------------

func (r *SQLiteRepository) SaveQueue(ctx context.Context, q *domain.Queue) error {
	query := `INSERT INTO queues (id, name, consumer_app_id, max_retries, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, q.ID, q.Name, q.ConsumerAppID, q.MaxRetries, q.CreatedAt.Format(time.RFC3339))
	return err
}

func (r *SQLiteRepository) ListQueues(ctx context.Context) ([]*domain.Queue, error) {
	query := `
		SELECT q.id, q.name, q.consumer_app_id, q.max_retries, q.created_at, COUNT(m.id) as backlog
		FROM queues q
		LEFT JOIN queue_messages m ON q.id = m.queue_id
		GROUP BY q.id, q.name, q.consumer_app_id, q.max_retries, q.created_at
		ORDER BY q.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queues []*domain.Queue
	for rows.Next() {
		var q domain.Queue
		var createdAtStr string
		if err := rows.Scan(&q.ID, &q.Name, &q.ConsumerAppID, &q.MaxRetries, &createdAtStr, &q.BacklogCount); err != nil {
			return nil, err
		}
		q.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		queues = append(queues, &q)
	}
	return queues, nil
}

func (r *SQLiteRepository) DeleteQueue(ctx context.Context, id string) error {
	query := `DELETE FROM queues WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *SQLiteRepository) EnqueueMessage(ctx context.Context, msg *domain.QueueMessage) error {
	query := `INSERT INTO queue_messages (id, queue_id, body, attempts, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, msg.ID, msg.QueueID, msg.Body, msg.Attempts, msg.CreatedAt.Format(time.RFC3339))
	return err
}

func (r *SQLiteRepository) ListQueueMessages(ctx context.Context, queueID string, limit int) ([]*domain.QueueMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT id, queue_id, body, attempts, created_at FROM queue_messages WHERE queue_id = ? ORDER BY created_at DESC LIMIT ?`
	rows, err := r.db.QueryContext(ctx, query, queueID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*domain.QueueMessage
	for rows.Next() {
		var m domain.QueueMessage
		var createdAtStr string
		if err := rows.Scan(&m.ID, &m.QueueID, &m.Body, &m.Attempts, &createdAtStr); err != nil {
			return nil, err
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		messages = append(messages, &m)
	}
	return messages, nil
}

// -----------------------------------------------------------------------------
// Cron Triggers Operations
// -----------------------------------------------------------------------------

func (r *SQLiteRepository) SaveCronTrigger(ctx context.Context, c *domain.CronTrigger) error {
	query := `INSERT INTO cron_triggers (id, name, cron_expression, target_app_id, status, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, c.ID, c.Name, c.CronExpression, c.TargetAppID, c.Status, c.CreatedAt.Format(time.RFC3339))
	return err
}

func (r *SQLiteRepository) ListCronTriggers(ctx context.Context) ([]*domain.CronTrigger, error) {
	query := `SELECT id, name, cron_expression, target_app_id, status, last_run_at, next_run_at, created_at FROM cron_triggers ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triggers []*domain.CronTrigger
	for rows.Next() {
		var c domain.CronTrigger
		var createdAtStr string
		var lastRunStr, nextRunStr sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.CronExpression, &c.TargetAppID, &c.Status, &lastRunStr, &nextRunStr, &createdAtStr); err != nil {
			return nil, err
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		if lastRunStr.Valid {
			t, _ := time.Parse(time.RFC3339, lastRunStr.String)
			c.LastRunAt = &t
		}
		if nextRunStr.Valid {
			t, _ := time.Parse(time.RFC3339, nextRunStr.String)
			c.NextRunAt = &t
		}
		triggers = append(triggers, &c)
	}
	return triggers, nil
}

func (r *SQLiteRepository) DeleteCronTrigger(ctx context.Context, id string) error {
	query := `DELETE FROM cron_triggers WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *SQLiteRepository) RecordCronRun(ctx context.Context, run *domain.CronRun) error {
	query := `INSERT INTO cron_runs (id, trigger_id, status, status_code, duration_ms, executed_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, run.ID, run.TriggerID, run.Status, run.StatusCode, run.DurationMs, run.ExecutedAt.Format(time.RFC3339))
	if err != nil {
		return err
	}
	now := run.ExecutedAt.Format(time.RFC3339)
	_, _ = r.db.ExecContext(ctx, "UPDATE cron_triggers SET last_run_at = ? WHERE id = ?", now, run.TriggerID)
	return nil
}

func (r *SQLiteRepository) ListCronRuns(ctx context.Context, triggerID string) ([]*domain.CronRun, error) {
	query := `SELECT id, trigger_id, status, status_code, duration_ms, executed_at FROM cron_runs WHERE trigger_id = ? ORDER BY executed_at DESC LIMIT 50`
	rows, err := r.db.QueryContext(ctx, query, triggerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*domain.CronRun
	for rows.Next() {
		var run domain.CronRun
		var executedAtStr string
		if err := rows.Scan(&run.ID, &run.TriggerID, &run.Status, &run.StatusCode, &run.DurationMs, &executedAtStr); err != nil {
			return nil, err
		}
		run.ExecutedAt, _ = time.Parse(time.RFC3339, executedAtStr)
		runs = append(runs, &run)
	}
	return runs, nil
}

// -----------------------------------------------------------------------------
// Workflows Operations
// -----------------------------------------------------------------------------

func (r *SQLiteRepository) SaveWorkflow(ctx context.Context, wf *domain.Workflow) error {
	stepsJSON, _ := json.Marshal(wf.Steps)
	query := `INSERT INTO workflows (id, name, target_app_id, steps, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, wf.ID, wf.Name, wf.TargetAppID, string(stepsJSON), wf.CreatedAt.Format(time.RFC3339))
	return err
}

func (r *SQLiteRepository) ListWorkflows(ctx context.Context) ([]*domain.Workflow, error) {
	query := `SELECT id, name, target_app_id, steps, created_at FROM workflows ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.Workflow
	for rows.Next() {
		var wf domain.Workflow
		var stepsJSON, createdAtStr string
		if err := rows.Scan(&wf.ID, &wf.Name, &wf.TargetAppID, &stepsJSON, &createdAtStr); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(stepsJSON), &wf.Steps)
		wf.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		list = append(list, &wf)
	}
	return list, nil
}

func (r *SQLiteRepository) DeleteWorkflow(ctx context.Context, id string) error {
	query := `DELETE FROM workflows WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *SQLiteRepository) SaveWorkflowRun(ctx context.Context, run *domain.WorkflowRun) error {
	logsJSON, _ := json.Marshal(run.Logs)
	var finishedAtStr sql.NullString
	if run.FinishedAt != nil {
		finishedAtStr = sql.NullString{String: run.FinishedAt.Format(time.RFC3339), Valid: true}
	}
	query := `
		INSERT INTO workflow_runs (id, workflow_id, status, current_step, logs, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			current_step = excluded.current_step,
			logs = excluded.logs,
			finished_at = excluded.finished_at
	`
	_, err := r.db.ExecContext(ctx, query,
		run.ID, run.WorkflowID, run.Status, run.CurrentStep, string(logsJSON), run.StartedAt.Format(time.RFC3339), finishedAtStr,
	)
	return err
}

func (r *SQLiteRepository) ListWorkflowRuns(ctx context.Context, workflowID string) ([]*domain.WorkflowRun, error) {
	query := `SELECT id, workflow_id, status, current_step, logs, started_at, finished_at FROM workflow_runs WHERE workflow_id = ? ORDER BY started_at DESC LIMIT 50`
	rows, err := r.db.QueryContext(ctx, query, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*domain.WorkflowRun
	for rows.Next() {
		var run domain.WorkflowRun
		var logsJSON, startedAtStr string
		var finishedAtStr sql.NullString
		if err := rows.Scan(&run.ID, &run.WorkflowID, &run.Status, &run.CurrentStep, &logsJSON, &startedAtStr, &finishedAtStr); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(logsJSON), &run.Logs)
		run.StartedAt, _ = time.Parse(time.RFC3339, startedAtStr)
		if finishedAtStr.Valid {
			t, _ := time.Parse(time.RFC3339, finishedAtStr.String)
			run.FinishedAt = &t
		}
		runs = append(runs, &run)
	}
	return runs, nil
}

// -----------------------------------------------------------------------------
// Durable Objects Operations
// -----------------------------------------------------------------------------

func (r *SQLiteRepository) SaveDurableObjectClass(ctx context.Context, doc *domain.DurableObjectClass) error {
	facetsJSON, _ := json.Marshal(doc.Facets)
	query := `INSERT INTO durable_objects (id, name, app_id, facets, storage_backend, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, doc.ID, doc.Name, doc.AppID, string(facetsJSON), doc.StorageBackend, doc.CreatedAt.Format(time.RFC3339))
	return err
}

func (r *SQLiteRepository) ListDurableObjectClasses(ctx context.Context) ([]*domain.DurableObjectClass, error) {
	query := `
		SELECT d.id, d.name, d.app_id, d.facets, d.storage_backend, d.created_at, COUNT(i.id) as instance_count
		FROM durable_objects d
		LEFT JOIN do_instances i ON d.id = i.class_id
		GROUP BY d.id, d.name, d.app_id, d.facets, d.storage_backend, d.created_at
		ORDER BY d.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.DurableObjectClass
	for rows.Next() {
		var doc domain.DurableObjectClass
		var facetsJSON, createdAtStr string
		if err := rows.Scan(&doc.ID, &doc.Name, &doc.AppID, &facetsJSON, &doc.StorageBackend, &createdAtStr, &doc.InstancesCount); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(facetsJSON), &doc.Facets)
		doc.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		list = append(list, &doc)
	}
	return list, nil
}

func (r *SQLiteRepository) DeleteDurableObjectClass(ctx context.Context, id string) error {
	query := `DELETE FROM durable_objects WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *SQLiteRepository) SaveDOInstance(ctx context.Context, inst *domain.DurableObjectInstance) error {
	var alarmAtStr sql.NullString
	if inst.AlarmAt != nil {
		alarmAtStr = sql.NullString{String: inst.AlarmAt.Format(time.RFC3339), Valid: true}
	}
	query := `INSERT INTO do_instances (id, class_id, object_id, status, storage_keys_count, alarm_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, inst.ID, inst.ClassID, inst.ObjectID, inst.Status, inst.StorageKeysCount, alarmAtStr, inst.CreatedAt.Format(time.RFC3339))
	return err
}

func (r *SQLiteRepository) ListDOInstances(ctx context.Context, classID string) ([]*domain.DurableObjectInstance, error) {
	query := `SELECT id, class_id, object_id, status, storage_keys_count, alarm_at, created_at FROM do_instances WHERE class_id = ? ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, classID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.DurableObjectInstance
	for rows.Next() {
		var inst domain.DurableObjectInstance
		var createdAtStr string
		var alarmAtStr sql.NullString
		if err := rows.Scan(&inst.ID, &inst.ClassID, &inst.ObjectID, &inst.Status, &inst.StorageKeysCount, &alarmAtStr, &createdAtStr); err != nil {
			return nil, err
		}
		inst.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		if alarmAtStr.Valid {
			t, _ := time.Parse(time.RFC3339, alarmAtStr.String)
			inst.AlarmAt = &t
		}
		list = append(list, &inst)
	}
	return list, nil
}

// -----------------------------------------------------------------------------
// Containers Operations
// -----------------------------------------------------------------------------

func (r *SQLiteRepository) SaveContainer(ctx context.Context, ct *domain.ContainerWorkload) error {
	envVarsJSON, _ := json.Marshal(ct.EnvVars)
	query := `INSERT INTO containers (id, name, image, port, status, env_vars, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, ct.ID, ct.Name, ct.Image, ct.Port, ct.Status, string(envVarsJSON), ct.CreatedAt.Format(time.RFC3339))
	return err
}

func (r *SQLiteRepository) ListContainers(ctx context.Context) ([]*domain.ContainerWorkload, error) {
	query := `SELECT id, name, image, port, status, env_vars, created_at FROM containers ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.ContainerWorkload
	for rows.Next() {
		var ct domain.ContainerWorkload
		var envVarsJSON, createdAtStr string
		if err := rows.Scan(&ct.ID, &ct.Name, &ct.Image, &ct.Port, &ct.Status, &envVarsJSON, &createdAtStr); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(envVarsJSON), &ct.EnvVars)
		ct.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		list = append(list, &ct)
	}
	return list, nil
}

func (r *SQLiteRepository) DeleteContainer(ctx context.Context, id string) error {
	query := `DELETE FROM containers WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// -----------------------------------------------------------------------------
// Static Assets Operations
// -----------------------------------------------------------------------------

func (r *SQLiteRepository) SaveStaticSite(ctx context.Context, site *domain.StaticSite) error {
	spaRouting := 0
	if site.SPARouting {
		spaRouting = 1
	}
	query := `INSERT INTO static_assets (id, name, subdomain, files_count, total_size, index_document, spa_routing, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, site.ID, site.Name, site.Subdomain, site.FilesCount, site.TotalSize, site.IndexDocument, spaRouting, site.CreatedAt.Format(time.RFC3339))
	return err
}

func (r *SQLiteRepository) ListStaticSites(ctx context.Context) ([]*domain.StaticSite, error) {
	query := `SELECT id, name, subdomain, files_count, total_size, index_document, spa_routing, created_at FROM static_assets ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.StaticSite
	for rows.Next() {
		var site domain.StaticSite
		var spaInt int
		var createdAtStr string
		if err := rows.Scan(&site.ID, &site.Name, &site.Subdomain, &site.FilesCount, &site.TotalSize, &site.IndexDocument, &spaInt, &createdAtStr); err != nil {
			return nil, err
		}
		site.SPARouting = spaInt == 1
		site.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		list = append(list, &site)
	}
	return list, nil
}

func (r *SQLiteRepository) DeleteStaticSite(ctx context.Context, id string) error {
	query := `DELETE FROM static_assets WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
