-- name: GetAgent :one
SELECT * FROM agents
WHERE id = $1 AND is_active = true;

-- name: GetAgentByType :one
SELECT * FROM agents
WHERE type = $1 AND is_active = true;

-- name: ListActiveAgents :many
SELECT * FROM agents
WHERE is_active = true
ORDER BY execution_order ASC;

-- name: CreateAgentExecution :one
INSERT INTO agent_executions (
    agent_id, session_id, workspace_id, input_data,
    status, tenant_id
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: UpdateAgentExecution :one
UPDATE agent_executions
SET 
    status = $2,
    output_data = $3,
    error_message = $4,
    execution_time_ms = $5,
    completed_at = CASE 
        WHEN $2 IN ('completed', 'failed') THEN CURRENT_TIMESTAMP 
        ELSE completed_at 
    END
WHERE id = $1
RETURNING *;

-- name: GetAgentExecution :one
SELECT ae.*, a.name as agent_name, a.type as agent_type
FROM agent_executions ae
JOIN agents a ON ae.agent_id = a.id
WHERE ae.id = $1;

-- name: ListSessionExecutions :many
SELECT ae.*, a.name as agent_name, a.type as agent_type
FROM agent_executions ae
JOIN agents a ON ae.agent_id = a.id
WHERE ae.session_id = $1
ORDER BY ae.created_at DESC;

-- name: CreateAgentCommunication :one
INSERT INTO agent_communications (
    from_agent_id, to_agent_id, execution_id,
    message_type, payload, tenant_id
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: ListAgentCommunications :many
SELECT * FROM agent_communications
WHERE execution_id = $1
ORDER BY created_at ASC;

-- name: UpdateAgentMetrics :exec
UPDATE agent_performance_metrics
SET 
    total_executions = total_executions + 1,
    successful_executions = successful_executions + CASE WHEN $2 = 'completed' THEN 1 ELSE 0 END,
    failed_executions = failed_executions + CASE WHEN $2 = 'failed' THEN 1 ELSE 0 END,
    total_execution_time_ms = total_execution_time_ms + $3,
    average_execution_time_ms = (total_execution_time_ms + $3) / (total_executions + 1),
    last_execution_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE agent_id = $1 AND date_bucket = DATE_TRUNC('day', CURRENT_TIMESTAMP);