-- name: CreateWorkspace :one
INSERT INTO workspaces (
    name, description, owner_id, settings
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: GetWorkspace :one
SELECT * FROM workspaces
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListUserWorkspaces :many
SELECT w.* FROM workspaces w
JOIN workspace_members wm ON w.id = wm.workspace_id
WHERE wm.user_id = $1 
  AND w.deleted_at IS NULL
ORDER BY w.created_at DESC;

-- name: AddWorkspaceMember :one
INSERT INTO workspace_members (
    workspace_id, user_id, role, permissions
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: RemoveWorkspaceMember :exec
DELETE FROM workspace_members
WHERE workspace_id = $1 AND user_id = $2;

-- name: CreateProject :one
INSERT INTO projects (
    workspace_id, name, description, type,
    status, configuration, tenant_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetProject :one
SELECT * FROM projects
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListWorkspaceProjects :many
SELECT * FROM projects
WHERE workspace_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateProjectStatus :one
UPDATE projects
SET 
    status = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: CreateProjectBuild :one
INSERT INTO project_builds (
    project_id, version, commit_hash, build_log,
    artifacts, status, deployed_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetLatestProjectBuild :one
SELECT * FROM project_builds
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: ListProjectBuilds :many
SELECT * FROM project_builds
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;