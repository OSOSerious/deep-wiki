-- Migration 005 Down: Drop Agents and Orchestration tables

-- Drop RLS policies
DROP POLICY IF EXISTS agents_tenant_isolation ON agents;
DROP POLICY IF EXISTS agent_executions_tenant_isolation ON agent_executions;
DROP POLICY IF EXISTS agent_communications_tenant_isolation ON agent_communications;
DROP POLICY IF EXISTS orchestration_workflows_tenant_isolation ON orchestration_workflows;
DROP POLICY IF EXISTS agent_performance_metrics_tenant_isolation ON agent_performance_metrics;

-- Drop triggers
DROP TRIGGER IF EXISTS set_timestamp_agents ON agents;
DROP TRIGGER IF EXISTS set_timestamp_orchestration_workflows ON orchestration_workflows;

-- Drop tables in reverse order (due to foreign key dependencies)
DROP TABLE IF EXISTS agent_performance_metrics;
DROP TABLE IF EXISTS orchestration_workflows;
DROP TABLE IF EXISTS agent_communications;
DROP TABLE IF EXISTS agent_executions;
DROP TABLE IF EXISTS agents;