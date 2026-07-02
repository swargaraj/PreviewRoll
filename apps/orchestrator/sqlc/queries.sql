-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = ?;

-- name: CreateUser :one
INSERT INTO users (email, username, password_hash)
VALUES (?, ?, ?)
RETURNING id, email, username, created_at, updated_at;

-- name: UpdateUser :exec
UPDATE users
SET username = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;

-- name: CreateSession :exec
INSERT INTO sessions (id, user_id, token, expires_at)
VALUES (?, ?, ?, ?);

-- name: GetSessionByToken :one
SELECT * FROM sessions WHERE token = ? AND expires_at > datetime('now');

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at <= datetime('now');

-- name: CreateProject :one
INSERT INTO projects (user_id, name, github_repo, github_repo_id, webhook_secret)
VALUES (?, ?, ?, ?, ?)
RETURNING id, user_id, name, github_repo, github_repo_id, created_at, updated_at;

-- name: GetProjectByID :one
SELECT * FROM projects WHERE id = ?;

-- name: GetProjectByGithubRepo :one
SELECT * FROM projects WHERE github_repo = ?;

-- name: ListProjectsByUserID :many
SELECT * FROM projects WHERE user_id = ? ORDER BY created_at DESC;

-- name: UpdateProject :exec
UPDATE projects
SET name = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = ?;

-- name: CreateDeployment :one
INSERT INTO deployments (project_id, pr_number, commit_sha, branch, state)
VALUES (?, ?, ?, ?, 'queued')
RETURNING id, project_id, pr_number, commit_sha, branch, state, created_at, updated_at;

-- name: GetDeploymentByID :one
SELECT * FROM deployments WHERE id = ?;

-- name: GetDeploymentByProjectAndPR :one
SELECT * FROM deployments WHERE project_id = ? AND pr_number = ?;

-- name: ListDeploymentsByProjectID :many
SELECT * FROM deployments WHERE project_id = ? ORDER BY created_at DESC;

-- name: ListDeploymentsByWorkerID :many
SELECT * FROM deployments WHERE worker_id = ? ORDER BY created_at DESC;

-- name: ListDeploymentsByState :many
SELECT * FROM deployments WHERE state = ? ORDER BY created_at DESC;

-- name: UpdateDeploymentState :exec
UPDATE deployments
SET state = ?, error_message = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: AssignDeploymentToWorker :exec
UPDATE deployments
SET worker_id = ?, state = 'assigned', updated_at = datetime('now')
WHERE id = ?;

-- name: StartDeployment :exec
UPDATE deployments
SET state = 'building', started_at = datetime('now'), updated_at = datetime('now')
WHERE id = ?;

-- name: CompleteDeployment :exec
UPDATE deployments
SET state = ?, container_id = ?, preview_url = ?, completed_at = datetime('now'), updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteDeployment :exec
DELETE FROM deployments WHERE id = ?;

-- name: CreateWorker :one
INSERT INTO workers (name, address, state, capacity, current_load, capabilities)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id, name, address, state, capacity, current_load, created_at, updated_at;

-- name: GetWorkerByID :one
SELECT * FROM workers WHERE id = ?;

-- name: GetWorkerByName :one
SELECT * FROM workers WHERE name = ?;

-- name: ListWorkersByState :many
SELECT * FROM workers WHERE state = ? ORDER BY created_at DESC;

-- name: ListOnlineWorkers :many
SELECT * FROM workers WHERE state = 'online' AND last_seen_at > datetime('now', '-45 seconds');

-- name: UpdateWorkerLoad :exec
UPDATE workers
SET current_load = ?, last_seen_at = datetime('now'), updated_at = datetime('now')
WHERE id = ?;

-- name: UpdateWorkerState :exec
UPDATE workers
SET state = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: ResetAllWorkers :exec
UPDATE workers SET state = 'offline', updated_at = datetime('now');

-- name: DeleteWorker :exec
DELETE FROM workers WHERE id = ?;

-- name: CreateDeploymentLog :exec
INSERT INTO deployment_logs (deployment_id, level, message, source)
VALUES (?, ?, ?, ?);

-- name: GetDeploymentLogs :many
SELECT * FROM deployment_logs WHERE deployment_id = ? ORDER BY timestamp ASC;

-- name: GetDeploymentLogsByLevel :many
SELECT * FROM deployment_logs WHERE deployment_id = ? AND level = ? ORDER BY timestamp ASC;

-- name: DeleteDeploymentLogs :exec
DELETE FROM deployment_logs WHERE deployment_id = ?;

-- name: DeleteOldDeploymentLogs :exec
DELETE FROM deployment_logs WHERE timestamp < datetime('now', ? || ' days');
