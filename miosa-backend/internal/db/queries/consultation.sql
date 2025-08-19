-- name: CreateConsultationSession :one
INSERT INTO consultation_sessions (
    user_id, tenant_id, phase, topic,
    context, initial_assessment
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetConsultationSession :one
SELECT * FROM consultation_sessions
WHERE id = $1;

-- name: UpdateConsultationPhase :one
UPDATE consultation_sessions
SET 
    phase = $2,
    phase_transitions = phase_transitions + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: CreateConsultationMessage :one
INSERT INTO consultation_messages (
    session_id, role, content, message_type,
    embedding
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: ListConsultationMessages :many
SELECT * FROM consultation_messages
WHERE session_id = $1
ORDER BY created_at ASC;

-- name: CreateConsultationInsight :one
INSERT INTO consultation_insights (
    session_id, insight_type, title, content,
    confidence_score, action_items, priority
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: ListConsultationInsights :many
SELECT * FROM consultation_insights
WHERE session_id = $1
ORDER BY priority DESC, confidence_score DESC;

-- name: SearchSimilarConsultations :many
SELECT 
    cs.*,
    1 - (cs.initial_assessment_embedding <=> $1::vector) AS similarity
FROM consultation_sessions cs
WHERE cs.tenant_id = $2
    AND cs.initial_assessment_embedding IS NOT NULL
ORDER BY cs.initial_assessment_embedding <=> $1::vector
LIMIT $3;

-- name: GetConsultationAnalytics :one
SELECT * FROM consultation_analytics
WHERE session_id = $1;