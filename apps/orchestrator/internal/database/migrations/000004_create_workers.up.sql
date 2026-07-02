CREATE TABLE IF NOT EXISTS workers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    address TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'offline',
    capacity INTEGER NOT NULL DEFAULT 5,
    current_load INTEGER NOT NULL DEFAULT 0,
    capabilities TEXT NOT NULL DEFAULT '{}',
    last_seen_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_workers_name ON workers(name);
CREATE INDEX IF NOT EXISTS idx_workers_state ON workers(state);
