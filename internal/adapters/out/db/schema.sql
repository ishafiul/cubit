CREATE TABLE IF NOT EXISTS nodes (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    ip_address TEXT NOT NULL,
    internal_port INTEGER NOT NULL,
    worker_port INTEGER NOT NULL,
    status TEXT NOT NULL,
    celld_version TEXT NOT NULL,
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
    git_repo TEXT,
    branch TEXT NOT NULL,
    inline_code TEXT,
    status TEXT NOT NULL,
    env_vars TEXT,
    bindings TEXT,
    active_deployment_id TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS deployments (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
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
