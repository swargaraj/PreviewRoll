package database

import (
	"encoding/json"
	"fmt"

	"github.com/swargaraj/previewroll/apps/orchestrator/internal/worker"
)

// LoadWorkers loads all workers from the database into the pool
func (db *DB) LoadWorkers() ([]*worker.Worker, error) {
	rows, err := db.Query(`
		SELECT id, name, address, state, capacity, current_load, capabilities, last_seen_at, created_at, updated_at
		FROM workers
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query workers: %w", err)
	}
	defer rows.Close()

	var workers []*worker.Worker
	for rows.Next() {
		w := &worker.Worker{}
		var capsJSON string
		err := rows.Scan(&w.ID, &w.Name, &w.Address, &w.State, &w.Capacity, &w.CurrentLoad, &capsJSON, &w.LastSeenAt, &w.CreatedAt, &w.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan worker: %w", err)
		}

		if capsJSON != "" {
			if err := json.Unmarshal([]byte(capsJSON), &w.Capabilities); err != nil {
				db.logger.Warn("failed to parse worker capabilities", "worker_id", w.ID, "error", err)
			}
		}

		// Reset runtime state after restart
		w.CurrentLoad = 0
		w.State = worker.StateOffline

		workers = append(workers, w)
	}

	return workers, nil
}
