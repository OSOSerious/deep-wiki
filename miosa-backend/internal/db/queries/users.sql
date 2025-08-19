-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: CreateUser :one
INSERT INTO users (
    email, username, full_name, password_hash, 
    company_name, company_size, industry, role, 
    goals, tenant_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET 
    full_name = COALESCE($2, full_name),
    company_name = COALESCE($3, company_name),
    company_size = COALESCE($4, company_size),
    industry = COALESCE($5, industry),
    role = COALESCE($6, role),
    goals = COALESCE($7, goals),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteUser :exec
UPDATE users
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: CreateUserSession :one
INSERT INTO user_sessions (
    user_id, token, ip_address, user_agent, expires_at
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetUserSession :one
SELECT s.*, u.*
FROM user_sessions s
JOIN users u ON s.user_id = u.id
WHERE s.token = $1 
  AND s.expires_at > CURRENT_TIMESTAMP
  AND s.revoked_at IS NULL;

-- name: RevokeUserSession :exec
UPDATE user_sessions
SET revoked_at = CURRENT_TIMESTAMP
WHERE token = $1;

-- name: CreatePasswordResetToken :one
INSERT INTO password_reset_tokens (
    user_id, token, expires_at
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetPasswordResetToken :one
SELECT * FROM password_reset_tokens
WHERE token = $1 
  AND expires_at > CURRENT_TIMESTAMP
  AND used_at IS NULL;

-- name: MarkPasswordResetTokenUsed :exec
UPDATE password_reset_tokens
SET used_at = CURRENT_TIMESTAMP
WHERE token = $1;