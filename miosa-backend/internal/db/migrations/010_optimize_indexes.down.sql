-- Migration 010 Down: Drop optimization features and indexes

-- Drop maintenance functions
DROP FUNCTION IF EXISTS optimize_vector_search(text, text, decimal);
DROP FUNCTION IF EXISTS perform_maintenance();
DROP FUNCTION IF EXISTS drop_old_partitions(text, integer);
DROP FUNCTION IF EXISTS create_monthly_partitions(text, date, integer);
DROP FUNCTION IF EXISTS refresh_materialized_views();

-- Drop materialized views
DROP MATERIALIZED VIEW IF EXISTS workspace_billing_summary;
DROP MATERIALIZED VIEW IF EXISTS agent_performance_summary;
DROP MATERIALIZED VIEW IF EXISTS project_performance_metrics;
DROP MATERIALIZED VIEW IF EXISTS user_activity_summary;

-- Drop HNSW vector indexes
DROP INDEX IF EXISTS idx_knowledge_entities_embedding_hnsw;
DROP INDEX IF EXISTS idx_conversation_memory_content_embedding_hnsw;

-- Drop text search indexes
DROP INDEX IF EXISTS idx_app_templates_name_description_trgm;
DROP INDEX IF EXISTS idx_projects_name_description_trgm;
DROP INDEX IF EXISTS idx_consultation_sessions_title_trgm;

-- Drop GIN indexes for JSONB columns
DROP INDEX IF EXISTS idx_usage_metrics_dimensions_gin;
DROP INDEX IF EXISTS idx_agent_executions_input_data_gin;
DROP INDEX IF EXISTS idx_business_insights_metrics_gin;
DROP INDEX IF EXISTS idx_consultation_insights_details_gin;

-- Drop partial indexes
DROP INDEX IF EXISTS idx_subscriptions_active_period;
DROP INDEX IF EXISTS idx_projects_active_by_workspace;
DROP INDEX IF EXISTS idx_users_active_last_activity;

-- Drop covering indexes
DROP INDEX IF EXISTS idx_workspace_members_workspace_covering;
DROP INDEX IF EXISTS idx_messages_session_sequence_covering;

-- Drop composite indexes
DROP INDEX IF EXISTS idx_agent_executions_session_agent_started;
DROP INDEX IF EXISTS idx_project_builds_project_status_completed;
DROP INDEX IF EXISTS idx_consultation_sessions_user_status_activity;