CREATE TABLE IF NOT EXISTS nodes (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    ip_address TEXT NOT NULL,
    internal_port INTEGER NOT NULL,
    worker_port INTEGER NOT NULL,
    status TEXT NOT NULL,
    celld_version TEXT NOT NULL,
    is_protected INTEGER NOT NULL DEFAULT 0,
    cpu_cores INTEGER DEFAULT 0,
    memory_bytes INTEGER DEFAULT 0,
    disk_free_bytes INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS applications (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    source_type TEXT NOT NULL DEFAULT 'git',
    subdomain TEXT NOT NULL DEFAULT '',
    git_repo TEXT,
    branch TEXT NOT NULL,
    inline_code TEXT,
    status TEXT NOT NULL,
    env_vars TEXT,
    bindings TEXT,
    active_deployment_id TEXT,
    auto_deploy INTEGER NOT NULL DEFAULT 1,
    compatibility_date TEXT NOT NULL DEFAULT '2024-09-23',
    compatibility_flags TEXT NOT NULL DEFAULT '[]',
    memory_limit_mb INTEGER NOT NULL DEFAULT 128,
    max_duration_ms INTEGER NOT NULL DEFAULT 50,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS deployments (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    build_version INTEGER NOT NULL DEFAULT 1,
    commit_hash TEXT NOT NULL,
    commit_message TEXT,
    status TEXT NOT NULL,
    bundle_size INTEGER DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL,
    finished_at TIMESTAMP,
    FOREIGN KEY(application_id) REFERENCES applications(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS deployment_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    deployment_id TEXT NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    step TEXT NOT NULL,
    message TEXT NOT NULL,
    level TEXT NOT NULL,
    FOREIGN KEY(deployment_id) REFERENCES deployments(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS domains (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    hostname TEXT UNIQUE NOT NULL,
    path_prefix TEXT NOT NULL DEFAULT '/',
    ssl_active INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY(application_id) REFERENCES applications(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_deployments_app_id ON deployments(application_id);
CREATE INDEX IF NOT EXISTS idx_deployment_logs_dep_id ON deployment_logs(deployment_id);
CREATE INDEX IF NOT EXISTS idx_domains_app_id ON domains(application_id);

-- celld Service Suite Tables
CREATE TABLE IF NOT EXISTS kv_namespaces (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS kv_entries (
    namespace_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    expiration_ttl INTEGER DEFAULT 0,
    metadata TEXT DEFAULT '',
    updated_at TIMESTAMP NOT NULL,
    PRIMARY KEY (namespace_id, key),
    FOREIGN KEY(namespace_id) REFERENCES kv_namespaces(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS d1_databases (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    size_bytes INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS queues (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    consumer_app_id TEXT DEFAULT '',
    max_retries INTEGER DEFAULT 3,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS queue_messages (
    id TEXT PRIMARY KEY,
    queue_id TEXT NOT NULL,
    body TEXT NOT NULL,
    attempts INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY(queue_id) REFERENCES queues(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS cron_triggers (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    cron_expression TEXT NOT NULL,
    target_app_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    last_run_at TIMESTAMP,
    next_run_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS cron_runs (
    id TEXT PRIMARY KEY,
    trigger_id TEXT NOT NULL,
    status TEXT NOT NULL,
    status_code INTEGER NOT NULL,
    duration_ms REAL NOT NULL,
    executed_at TIMESTAMP NOT NULL,
    FOREIGN KEY(trigger_id) REFERENCES cron_triggers(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS workflows (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    target_app_id TEXT NOT NULL,
    steps TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS workflow_runs (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL,
    status TEXT NOT NULL,
    current_step TEXT NOT NULL,
    logs TEXT NOT NULL,
    started_at TIMESTAMP NOT NULL,
    finished_at TIMESTAMP,
    FOREIGN KEY(workflow_id) REFERENCES workflows(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS durable_objects (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    app_id TEXT NOT NULL,
    facets TEXT NOT NULL,
    storage_backend TEXT NOT NULL DEFAULT 'sqlite',
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS do_instances (
    id TEXT PRIMARY KEY,
    class_id TEXT NOT NULL,
    object_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    storage_keys_count INTEGER DEFAULT 0,
    alarm_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY(class_id) REFERENCES durable_objects(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS containers (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    image TEXT NOT NULL,
    port INTEGER NOT NULL DEFAULT 8080,
    status TEXT NOT NULL DEFAULT 'running',
    env_vars TEXT,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS static_assets (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    subdomain TEXT NOT NULL DEFAULT '',
    files_count INTEGER DEFAULT 0,
    total_size INTEGER DEFAULT 0,
    index_document TEXT NOT NULL DEFAULT 'index.html',
    spa_routing INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS github_app_settings (
    id TEXT PRIMARY KEY,
    app_id TEXT NOT NULL DEFAULT '',
    app_name TEXT NOT NULL DEFAULT '',
    client_id TEXT NOT NULL DEFAULT '',
    client_secret TEXT NOT NULL DEFAULT '',
    webhook_secret TEXT NOT NULL DEFAULT '',
    private_key TEXT NOT NULL DEFAULT '',
    installation_id TEXT NOT NULL DEFAULT '',
    is_configured INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS r2_buckets (
    name TEXT PRIMARY KEY,
    created_at TIMESTAMP NOT NULL
);

