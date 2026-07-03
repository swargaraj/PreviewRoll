-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = ?;

-- name: GetUserCredentials :one
SELECT id, password_hash FROM users WHERE username = ?;

-- name: CreateUser :one
INSERT INTO users (username, password_hash)
VALUES (?, ?)
RETURNING id, username, created_at, updated_at;

-- name: UpdateUser :exec
UPDATE users SET username = ?, updated_at = datetime('now') WHERE id = ?;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;

-- name: CreateSession :exec
INSERT INTO sessions (id, user_id, token, expires_at) VALUES (?, ?, ?, ?);

-- name: GetSessionByToken :one
SELECT user_id FROM sessions WHERE token = ? AND expires_at > datetime('now');

-- name: GetUserBySessionToken :one
SELECT u.id, u.username, u.created_at
FROM users u
INNER JOIN sessions s ON u.id = s.user_id
WHERE s.token = ? AND s.expires_at > datetime('now');

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at <= datetime('now');

-- name: CreateProject :one
INSERT INTO projects (user_id, name, repo_url, vcs_provider, webhook_secret)
VALUES (?, ?, ?, ?, ?)
RETURNING id, user_id, name, repo_url, vcs_provider, created_at, updated_at;

-- name: GetProjectByID :one
SELECT * FROM projects WHERE id = ?;

-- name: GetProjectOwnerByID :one
SELECT user_id FROM projects WHERE id = ?;

-- name: GetProjectByRepoURL :one
SELECT id FROM projects WHERE repo_url = ?;

-- name: ListProjectsByUserID :many
SELECT * FROM projects WHERE user_id = ? ORDER BY created_at DESC;

-- name: CountProjectsByUserID :one
SELECT COUNT(*) FROM projects WHERE user_id = ?;

-- name: ListProjectsByUserIDPaginated :many
SELECT * FROM projects WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountProjectsByUserIDSearch :one
SELECT COUNT(*) FROM projects WHERE user_id = ? AND (name LIKE '%' || ? || '%' OR repo_url LIKE '%' || ? || '%');

-- name: ListProjectsByUserIDPaginatedSearch :many
SELECT * FROM projects WHERE user_id = ? AND (name LIKE '%' || ? || '%' OR repo_url LIKE '%' || ? || '%') ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: UpdateProject :exec
UPDATE projects SET name = ?, updated_at = datetime('now') WHERE id = ?;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = ?;

-- name: CreateDeployment :one
INSERT INTO deployments (project_id, pr_number, commit_sha, branch, state)
VALUES (?, ?, ?, ?, 'queued')
RETURNING id, project_id, pr_number, commit_sha, branch, state, created_at, updated_at;

-- name: GetDeploymentByID :one
SELECT * FROM deployments WHERE id = ?;

-- name: GetDeploymentByProjectAndPR :one
SELECT id FROM deployments WHERE project_id = ? AND pr_number = ?;

-- name: ListDeploymentsByProjectID :many
SELECT * FROM deployments WHERE project_id = ? ORDER BY created_at DESC;

-- name: ListDeploymentsByWorkerID :many
SELECT * FROM deployments WHERE worker_id = ? ORDER BY created_at DESC;

-- name: ListDeploymentsByState :many
SELECT * FROM deployments WHERE state = ? ORDER BY created_at DESC;

-- name: UpdateDeploymentState :exec
UPDATE deployments SET state = ?, error_message = ?, updated_at = datetime('now') WHERE id = ?;

-- name: AssignDeploymentToWorker :exec
UPDATE deployments SET worker_id = ?, state = 'assigned', updated_at = datetime('now') WHERE id = ?;

-- name: StartDeployment :exec
UPDATE deployments SET state = 'building', started_at = datetime('now'), updated_at = datetime('now') WHERE id = ?;

-- name: CompleteDeployment :exec
UPDATE deployments SET state = ?, container_id = ?, preview_url = ?, completed_at = datetime('now'), updated_at = datetime('now') WHERE id = ?;

-- name: UpdateDeploymentForRestart :exec
UPDATE deployments SET state = 'queued', worker_id = NULL, container_id = NULL, error_message = NULL, attempt = attempt + 1, updated_at = datetime('now') WHERE id = ? AND state IN ('failed', 'stopped');

-- name: UpdateDeploymentForStop :exec
UPDATE deployments SET state = 'stopped', updated_at = datetime('now'), completed_at = datetime('now') WHERE id = ? AND state NOT IN ('stopped', 'destroyed');

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

-- name: ListAllWorkers :many
SELECT * FROM workers ORDER BY id;

-- name: ListWorkersByState :many
SELECT * FROM workers WHERE state = ? ORDER BY created_at DESC;

-- name: ListOnlineWorkers :many
SELECT * FROM workers WHERE state = 'online' AND last_seen_at > datetime('now', '-45 seconds');

-- name: UpdateWorkerLoad :exec
UPDATE workers SET current_load = ?, last_seen_at = datetime('now'), updated_at = datetime('now') WHERE id = ?;

-- name: UpdateWorkerState :exec
UPDATE workers SET state = ?, updated_at = datetime('now') WHERE id = ?;

-- name: ResetAllWorkers :exec
UPDATE workers SET state = 'offline', updated_at = datetime('now');

-- name: DeleteWorker :exec
DELETE FROM workers WHERE id = ?;

-- name: CreateDomain :one
INSERT INTO domains (user_id, name) VALUES (?, ?) RETURNING id;

-- name: ListDomainsByUserID :many
SELECT * FROM domains WHERE user_id = ? ORDER BY created_at DESC;

-- name: DeleteDomainByIDAndUser :exec
DELETE FROM domains WHERE id = ? AND user_id = ?;

-- name: CreateProjectDomain :one
INSERT INTO project_domains (project_id, domain_id, prefix) VALUES (?, ?, ?) RETURNING id;

-- name: DeleteProjectDomain :exec
DELETE FROM project_domains WHERE domain_id = ? AND project_id = ?;

-- name: GetDomainByID :one
SELECT user_id FROM domains WHERE id = ?;

-- name: GetProjectDomainByProjectID :one
SELECT pd.prefix, d.name
FROM project_domains pd
INNER JOIN domains d ON pd.domain_id = d.id
WHERE pd.project_id = ? LIMIT 1;

-- name: ListProjectDomains :many
SELECT pd.id, pd.project_id, pd.domain_id, d.name as domain_name, pd.prefix
FROM project_domains pd
INNER JOIN domains d ON pd.domain_id = d.id
WHERE pd.project_id = ?;

-- name: GetDeploymentForUpdate :one
SELECT id, state FROM deployments WHERE project_id = ? AND pr_number = ?;

-- name: UpdateDeploymentFromWebhook :exec
UPDATE deployments SET commit_sha = ?, branch = ?, state = 'queued', preview_url = ?, updated_at = datetime('now') WHERE id = ?;

-- name: CreateDeploymentFromWebhook :one
INSERT INTO deployments (project_id, pr_number, commit_sha, branch, state, preview_url)
VALUES (?, ?, ?, ?, 'queued', ?)
RETURNING id;

-- name: StopDeploymentsByProjectAndPR :exec
UPDATE deployments SET state = 'stopped', updated_at = datetime('now'), completed_at = datetime('now')
WHERE project_id = ? AND pr_number = ? AND state NOT IN ('stopped', 'destroyed');
