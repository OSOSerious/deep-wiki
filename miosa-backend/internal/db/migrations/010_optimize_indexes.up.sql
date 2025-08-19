-- Migration 010: Performance Optimization and Advanced Indexes
-- This migration creates performance indexes, HNSW for vectors, and materialized views

-- Create materialized view for user activity summary
CREATE MATERIALIZED VIEW user_activity_summary AS
SELECT 
    u.id,
    u.tenant_id,
    u.email,
    u.full_name,
    u.last_active_at,
    w.id as workspace_id,
    w.name as workspace_name,
    COUNT(DISTINCT p.id) as project_count,
    COUNT(DISTINCT cs.id) as consultation_count,
    COUNT(DISTINCT pb.id) as build_count,
    MAX(cs.last_activity_at) as last_consultation_at,
    MAX(pb.completed_at) as last_build_at,
    SUM(CASE WHEN cs.status = 'completed' THEN 1 ELSE 0 END) as completed_consultations,
    AVG(cs.user_rating) as avg_consultation_rating
FROM users u
LEFT JOIN workspace_members wm ON u.id = wm.user_id
LEFT JOIN workspaces w ON wm.workspace_id = w.id
LEFT JOIN projects p ON w.id = p.workspace_id
LEFT JOIN consultation_sessions cs ON u.id = cs.user_id
LEFT JOIN project_builds pb ON p.id = pb.project_id
WHERE u.status = 'active'
GROUP BY u.id, u.tenant_id, u.email, u.full_name, u.last_active_at, w.id, w.name
ORDER BY u.last_active_at DESC NULLS LAST;

-- Create unique index for the materialized view
CREATE UNIQUE INDEX idx_user_activity_summary_user_workspace 
ON user_activity_summary (id, workspace_id);

-- Create materialized view for project performance metrics
CREATE MATERIALIZED VIEW project_performance_metrics AS
SELECT 
    p.id,
    p.tenant_id,
    p.workspace_id,
    p.name,
    p.framework,
    p.status,
    COUNT(DISTINCT pb.id) as total_builds,
    COUNT(DISTINCT CASE WHEN pb.status = 'success' THEN pb.id END) as successful_builds,
    COUNT(DISTINCT CASE WHEN pb.status = 'failed' THEN pb.id END) as failed_builds,
    ROUND(
        COUNT(DISTINCT CASE WHEN pb.status = 'success' THEN pb.id END) * 100.0 / 
        NULLIF(COUNT(DISTINCT pb.id), 0), 2
    ) as success_rate,
    AVG(pb.duration_seconds) as avg_build_time,
    MIN(pb.duration_seconds) as min_build_time,
    MAX(pb.duration_seconds) as max_build_time,
    COUNT(DISTINCT ad.id) as deployment_count,
    MAX(pb.completed_at) as last_build_at,
    MAX(ad.deployment_time) as last_deployment_at,
    p.created_at,
    p.updated_at
FROM projects p
LEFT JOIN project_builds pb ON p.id = pb.project_id
LEFT JOIN app_deployments ad ON p.id = ad.project_id
WHERE p.status = 'active'
GROUP BY p.id, p.tenant_id, p.workspace_id, p.name, p.framework, p.status, p.created_at, p.updated_at
ORDER BY p.updated_at DESC;

-- Create unique index for project performance metrics
CREATE UNIQUE INDEX idx_project_performance_metrics_id 
ON project_performance_metrics (id);

-- Create materialized view for agent performance summary
CREATE MATERIALIZED VIEW agent_performance_summary AS
SELECT 
    a.name,
    a.agent_type,
    a.category,
    a.status,
    COUNT(DISTINCT ae.id) as total_executions,
    COUNT(DISTINCT CASE WHEN ae.status = 'completed' THEN ae.id END) as successful_executions,
    COUNT(DISTINCT CASE WHEN ae.status = 'failed' THEN ae.id END) as failed_executions,
    ROUND(
        COUNT(DISTINCT CASE WHEN ae.status = 'completed' THEN ae.id END) * 100.0 / 
        NULLIF(COUNT(DISTINCT ae.id), 0), 2
    ) as success_rate,
    AVG(ae.execution_time_ms) as avg_execution_time_ms,
    MIN(ae.execution_time_ms) as min_execution_time_ms,
    MAX(ae.execution_time_ms) as max_execution_time_ms,
    SUM(ae.tokens_used) as total_tokens_used,
    AVG(ae.tokens_used) as avg_tokens_per_execution,
    MAX(ae.started_at) as last_execution_at,
    COUNT(DISTINCT ae.session_id) as unique_sessions,
    a.created_at,
    a.updated_at
FROM agents a
LEFT JOIN agent_executions ae ON a.name = ae.agent_name
WHERE a.status = 'active'
GROUP BY a.name, a.agent_type, a.category, a.status, a.created_at, a.updated_at
ORDER BY total_executions DESC NULLS LAST;

-- Create unique index for agent performance summary
CREATE UNIQUE INDEX idx_agent_performance_summary_name 
ON agent_performance_summary (name);

-- Create materialized view for workspace billing summary
CREATE MATERIALIZED VIEW workspace_billing_summary AS
SELECT 
    w.id,
    w.tenant_id,
    w.name,
    w.plan_type,
    s.status as subscription_status,
    s.price_cents,
    s.currency,
    s.current_period_start,
    s.current_period_end,
    COUNT(DISTINCT p.id) as project_count,
    COUNT(DISTINCT wm.id) as member_count,
    SUM(CASE WHEN um.is_billable = true THEN um.value * COALESCE(um.billing_rate, 0) ELSE 0 END) as current_usage_cost,
    COUNT(DISTINCT i.id) as invoice_count,
    SUM(i.total_cents) as total_invoiced_cents,
    COUNT(DISTINCT CASE WHEN i.status = 'paid' THEN i.id END) as paid_invoices,
    MAX(i.paid_at) as last_payment_at
FROM workspaces w
LEFT JOIN subscriptions s ON w.id = s.workspace_id
LEFT JOIN projects p ON w.id = p.workspace_id AND p.status = 'active'
LEFT JOIN workspace_members wm ON w.id = wm.workspace_id AND wm.status = 'active'
LEFT JOIN usage_metrics um ON w.id = um.workspace_id 
    AND um.period_start >= CURRENT_DATE - INTERVAL '30 days'
LEFT JOIN invoices i ON s.id = i.subscription_id
WHERE w.status = 'active'
GROUP BY w.id, w.tenant_id, w.name, w.plan_type, s.status, s.price_cents, s.currency, 
         s.current_period_start, s.current_period_end
ORDER BY w.created_at DESC;

-- Create unique index for workspace billing summary
CREATE UNIQUE INDEX idx_workspace_billing_summary_id 
ON workspace_billing_summary (id);

-- Create composite indexes for frequently queried combinations
CREATE INDEX idx_consultation_sessions_user_status_activity 
ON consultation_sessions (user_id, status, last_activity_at DESC) 
WHERE status IN ('active', 'completed');

CREATE INDEX idx_project_builds_project_status_completed 
ON project_builds (project_id, status, completed_at DESC NULLS LAST)
WHERE status IN ('success', 'failed');

CREATE INDEX idx_agent_executions_session_agent_started 
ON agent_executions (session_id, agent_name, started_at DESC);

-- Create covering indexes to avoid table lookups
CREATE INDEX idx_messages_session_sequence_covering 
ON messages (session_id, sequence_number) 
INCLUDE (sender_type, content, message_type, created_at);

CREATE INDEX idx_workspace_members_workspace_covering 
ON workspace_members (workspace_id) 
INCLUDE (user_id, role, status, joined_at);

-- Create partial indexes for common filtered queries
CREATE INDEX idx_users_active_last_activity 
ON users (last_active_at DESC) 
WHERE status = 'active' AND last_active_at IS NOT NULL;

CREATE INDEX idx_projects_active_by_workspace 
ON projects (workspace_id, updated_at DESC) 
WHERE status = 'active';

CREATE INDEX idx_subscriptions_active_period 
ON subscriptions (workspace_id, current_period_end) 
WHERE status = 'active';

-- Create GIN indexes for JSONB columns frequently queried
CREATE INDEX idx_consultation_insights_details_gin 
ON consultation_insights USING gin (details);

CREATE INDEX idx_business_insights_metrics_gin 
ON business_insights USING gin (metrics);

CREATE INDEX idx_agent_executions_input_data_gin 
ON agent_executions USING gin (input_data);

CREATE INDEX idx_usage_metrics_dimensions_gin 
ON usage_metrics USING gin (dimensions);

-- Create indexes for text search
CREATE INDEX idx_consultation_sessions_title_trgm 
ON consultation_sessions USING gin (title gin_trgm_ops);

CREATE INDEX idx_projects_name_description_trgm 
ON projects USING gin ((name || ' ' || COALESCE(description, '')) gin_trgm_ops);

CREATE INDEX idx_app_templates_name_description_trgm 
ON app_templates USING gin ((name || ' ' || COALESCE(description, '')) gin_trgm_ops);

-- Create additional HNSW vector indexes for optimal performance
CREATE INDEX idx_conversation_memory_content_embedding_hnsw 
ON conversation_memory USING hnsw (content_embedding vector_cosine_ops) 
WITH (m = 24, ef_construction = 100);

CREATE INDEX idx_knowledge_entities_embedding_hnsw 
ON knowledge_entities USING hnsw (embedding vector_cosine_ops) 
WITH (m = 24, ef_construction = 100);

-- Create function for refreshing materialized views
CREATE OR REPLACE FUNCTION refresh_materialized_views()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY user_activity_summary;
    REFRESH MATERIALIZED VIEW CONCURRENTLY project_performance_metrics;
    REFRESH MATERIALIZED VIEW CONCURRENTLY agent_performance_summary;
    REFRESH MATERIALIZED VIEW CONCURRENTLY workspace_billing_summary;
END;
$$ LANGUAGE plpgsql;

-- Create function for automatic partition management
CREATE OR REPLACE FUNCTION create_monthly_partitions(
    table_name text,
    start_date date DEFAULT CURRENT_DATE,
    months_ahead integer DEFAULT 6
)
RETURNS void AS $$
DECLARE
    partition_date date;
    partition_name text;
    next_month_date date;
BEGIN
    FOR i IN 0..months_ahead LOOP
        partition_date := date_trunc('month', start_date + (i || ' months')::interval)::date;
        next_month_date := (partition_date + interval '1 month')::date;
        partition_name := table_name || '_' || to_char(partition_date, 'YYYY_MM');
        
        -- Check if partition already exists
        IF NOT EXISTS (
            SELECT 1 FROM pg_tables 
            WHERE tablename = partition_name
        ) THEN
            EXECUTE format('CREATE TABLE %I PARTITION OF %I 
                           FOR VALUES FROM (%L) TO (%L)',
                          partition_name, table_name, 
                          partition_date, next_month_date);
            
            RAISE NOTICE 'Created partition: %', partition_name;
        END IF;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Create function for cleaning old partitions
CREATE OR REPLACE FUNCTION drop_old_partitions(
    table_name text,
    months_to_keep integer DEFAULT 12
)
RETURNS void AS $$
DECLARE
    cutoff_date date;
    partition_name text;
    partition_record record;
BEGIN
    cutoff_date := (date_trunc('month', CURRENT_DATE) - (months_to_keep || ' months')::interval)::date;
    
    FOR partition_record IN
        SELECT schemaname, tablename 
        FROM pg_tables 
        WHERE tablename LIKE table_name || '_%'
        AND tablename ~ '^\w+_\d{4}_\d{2}$'
    LOOP
        -- Extract date from partition name
        SELECT to_date(substring(partition_record.tablename from '\d{4}_\d{2}$'), 'YYYY_MM') 
        INTO partition_date;
        
        IF partition_date < cutoff_date THEN
            EXECUTE format('DROP TABLE IF EXISTS %I.%I', 
                          partition_record.schemaname, partition_record.tablename);
            RAISE NOTICE 'Dropped old partition: %', partition_record.tablename;
        END IF;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Create function for database maintenance
CREATE OR REPLACE FUNCTION perform_maintenance()
RETURNS void AS $$
BEGIN
    -- Refresh materialized views
    PERFORM refresh_materialized_views();
    
    -- Create future partitions
    PERFORM create_monthly_partitions('messages');
    PERFORM create_monthly_partitions('usage_metrics');
    PERFORM create_monthly_partitions('audit_logs');
    PERFORM create_monthly_partitions('api_usage_logs');
    
    -- Clean old partitions (keep 12 months)
    PERFORM drop_old_partitions('messages', 12);
    PERFORM drop_old_partitions('usage_metrics', 12);
    PERFORM drop_old_partitions('audit_logs', 24); -- Keep audit logs longer
    PERFORM drop_old_partitions('api_usage_logs', 6);
    
    -- Analyze tables for query planning
    ANALYZE users;
    ANALYZE workspaces;
    ANALYZE projects;
    ANALYZE consultation_sessions;
    ANALYZE messages;
    ANALYZE agent_executions;
    
    RAISE NOTICE 'Database maintenance completed successfully';
END;
$$ LANGUAGE plpgsql;

-- Create advisory function for vector similarity search optimization
CREATE OR REPLACE FUNCTION optimize_vector_search(
    table_name text,
    column_name text,
    target_recall decimal DEFAULT 0.95
)
RETURNS text AS $$
DECLARE
    current_ef_search integer;
    recommended_ef_search integer;
    result_text text;
BEGIN
    -- Calculate recommended ef_search based on target recall
    -- This is a simplified heuristic - in production, this would be based on benchmarking
    recommended_ef_search := CASE
        WHEN target_recall >= 0.95 THEN 100
        WHEN target_recall >= 0.90 THEN 80
        WHEN target_recall >= 0.85 THEN 60
        ELSE 40
    END;
    
    -- Set the ef_search parameter for the current session
    EXECUTE format('SET hnsw.ef_search = %s', recommended_ef_search);
    
    result_text := format('Optimized vector search for %s.%s with ef_search=%s (target recall: %s)',
                         table_name, column_name, recommended_ef_search, target_recall);
    
    RETURN result_text;
END;
$$ LANGUAGE plpgsql;

-- Add comments for the optimization functions
COMMENT ON FUNCTION refresh_materialized_views() IS 'Refreshes all materialized views concurrently for performance analytics';
COMMENT ON FUNCTION create_monthly_partitions(text, date, integer) IS 'Creates monthly partitions for time-series tables';
COMMENT ON FUNCTION drop_old_partitions(text, integer) IS 'Drops old partitions to manage storage space';
COMMENT ON FUNCTION perform_maintenance() IS 'Comprehensive database maintenance routine';
COMMENT ON FUNCTION optimize_vector_search(text, text, decimal) IS 'Optimizes HNSW vector search parameters for target recall';

-- Add comments for materialized views
COMMENT ON MATERIALIZED VIEW user_activity_summary IS 'Aggregated user activity metrics for dashboard and analytics';
COMMENT ON MATERIALIZED VIEW project_performance_metrics IS 'Project build and deployment performance statistics';
COMMENT ON MATERIALIZED VIEW agent_performance_summary IS 'Agent execution performance and reliability metrics';
COMMENT ON MATERIALIZED VIEW workspace_billing_summary IS 'Workspace billing and usage cost summary for financial reporting';