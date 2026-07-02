package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/swargaraj/previewroll/apps/orchestrator/internal/worker"
)

// LoadWorkers loads all workers from the database into the pool
func (db *DB) LoadWorkers() ([]*worker.Worker, error) {
	rows, err := db.Q().ListAllWorkers(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to query workers: %w", err)
	}

	workers := make([]*worker.Worker, 0, len(rows))
	for _, r := range rows {
		w := &worker.Worker{
			ID:          r.ID,
			Name:        r.Name,
			Address:     r.Address,
			State:       worker.State(r.State),
			Capacity:    int(r.Capacity),
			CurrentLoad: 0,
			LastSeenAt:  r.LastSeenAt.Time,
			CreatedAt:   r.CreatedAt.Time,
			UpdatedAt:   r.UpdatedAt.Time,
		}

		if r.Capabilities != "" {
			if err := json.Unmarshal([]byte(r.Capabilities), &w.Capabilities); err != nil {
				db.logger.Warn("failed to parse worker capabilities", "worker_id", w.ID, "error", err)
			}
		}

		// Reset runtime state after restart
		w.State = worker.StateOffline

		workers = append(workers, w)
	}

	return workers, nil
}
