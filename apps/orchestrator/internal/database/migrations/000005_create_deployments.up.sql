CREATE TABLE IF NOT EXISTS deployments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    worker_id INTEGER,
    pr_number INTEGER NOT NULL,
    commit_sha TEXT NOT NULL,
    branch TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'queued',
    preview_url TEXT,
    container_id TEXT,
    build_log_path TEXT,
    error_message TEXT,
    attempt INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    completed_at DATETIME,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (worker_id) REFERENCES workers(id) ON DELETE SET NULL,
    UNIQUE(project_id, pr_number)
);

CREATE INDEX IF NOT EXISTS idx_deployments_project_id ON deployments(project_id);
CREATE INDEX IF NOT EXISTS idx_deployments_worker_id ON deployments(worker_id);
CREATE INDEX IF NOT EXISTS idx_deployments_state ON deployments(state);
CREATE INDEX IF NOT EXISTS idx_deployments_pr_number ON deployments(pr_number);
