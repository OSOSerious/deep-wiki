-- Migration 005: Multi-Agent System and Orchestration
-- This migration creates the 8-agent system with execution tracking and communication

-- Create agents registry table
CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    -- Agent identity
    name VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    version VARCHAR(20) DEFAULT '1.0.0',
    
    -- Agent classification
    category VARCHAR(50) NOT NULL CHECK (category IN ('core', 'specialized', 'integration', 'monitoring')),
    agent_type VARCHAR(50) NOT NULL CHECK (agent_type IN (
        'architect', 'development', 'quality', 'deployment', 
        'monitoring', 'ai_providers', 'integration', 'analysis', 
        'strategy', 'recommender', 'communication'
    )),
    
    -- Agent capabilities
    capabilities JSONB NOT NULL DEFAULT '[]',
    tools JSONB DEFAULT '[]',
    models JSONB DEFAULT '[]',
    
    -- Agent configuration
    config JSONB DEFAULT '{}',
    max_concurrent_tasks INTEGER DEFAULT 5,
    timeout_seconds INTEGER DEFAULT 300,
    
    -- Agent status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'maintenance', 'error')),
    health_status VARCHAR(20) DEFAULT 'healthy' CHECK (health_status IN ('healthy', 'degraded', 'unhealthy')),
    
    -- Performance metrics
    total_executions INTEGER DEFAULT 0,
    successful_executions INTEGER DEFAULT 0,
    avg_execution_time_ms DECIMAL(10,2) DEFAULT 0,
    last_execution_at TIMESTAMPTZ,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create agent executions table for tracking all agent activities
CREATE TABLE IF NOT EXISTS agent_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    agent_name VARCHAR(100) NOT NULL,
    session_id UUID REFERENCES consultation_sessions(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    
    -- Execution context
    execution_type VARCHAR(50) NOT NULL CHECK (execution_type IN (
        'consultation', 'analysis', 'code_generation', 'deployment', 
        'monitoring', 'integration', 'quality_check', 'optimization'
    )),
    task_name VARCHAR(255) NOT NULL,
    task_description TEXT,
    
    -- Execution flow
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN (
        'pending', 'running', 'completed', 'failed', 'cancelled', 'timeout'
    )),
    priority VARCHAR(10) DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    
    -- Input and output
    input_data JSONB DEFAULT '{}',
    output_data JSONB DEFAULT '{}',
    error_details JSONB DEFAULT '{}',
    
    -- Execution metrics
    started_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMPTZ,
    execution_time_ms INTEGER,
    
    -- Resource usage
    tokens_used INTEGER DEFAULT 0,
    api_calls_made INTEGER DEFAULT 0,
    memory_used_mb DECIMAL(10,2),
    
    -- Dependencies and relationships
    parent_execution_id UUID REFERENCES agent_executions(id),
    dependent_executions UUID[],
    triggered_by VARCHAR(100), -- user_id or agent_name
    
    -- Results and artifacts
    artifacts JSONB DEFAULT '{}',
    logs TEXT,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create agent communications table for inter-agent messaging
CREATE TABLE IF NOT EXISTS agent_communications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    session_id UUID REFERENCES consultation_sessions(id) ON DELETE CASCADE,
    execution_id UUID REFERENCES agent_executions(id) ON DELETE CASCADE,
    
    -- Communication parties
    sender_agent VARCHAR(100) NOT NULL,
    receiver_agent VARCHAR(100) NOT NULL,
    
    -- Message details
    message_type VARCHAR(50) NOT NULL CHECK (message_type IN (
        'request', 'response', 'notification', 'error', 'data_exchange', 'coordination'
    )),
    subject VARCHAR(255),
    content JSONB NOT NULL,
    
    -- Communication flow
    thread_id UUID, -- For grouping related messages
    correlation_id UUID, -- For request-response matching
    
    -- Status and timing
    status VARCHAR(20) DEFAULT 'sent' CHECK (status IN ('pending', 'sent', 'received', 'processed', 'failed')),
    sent_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    received_at TIMESTAMPTZ,
    processed_at TIMESTAMPTZ,
    
    -- Priority and routing
    priority VARCHAR(10) DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    routing_key VARCHAR(255),
    
    -- Attachments and context
    attachments JSONB DEFAULT '[]',
    context JSONB DEFAULT '{}',
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create orchestration workflows table
CREATE TABLE IF NOT EXISTS orchestration_workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    session_id UUID NOT NULL REFERENCES consultation_sessions(id) ON DELETE CASCADE,
    
    -- Workflow identity
    name VARCHAR(255) NOT NULL,
    description TEXT,
    workflow_type VARCHAR(50) NOT NULL CHECK (workflow_type IN (
        'consultation', 'code_generation', 'deployment', 'analysis', 'optimization'
    )),
    
    -- Workflow definition
    definition JSONB NOT NULL, -- DAG structure
    variables JSONB DEFAULT '{}',
    
    -- Workflow execution
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN (
        'pending', 'running', 'completed', 'failed', 'cancelled', 'paused'
    )),
    current_step VARCHAR(255),
    steps_completed TEXT[],
    
    -- Progress tracking
    total_steps INTEGER,
    completed_steps INTEGER DEFAULT 0,
    progress_percentage DECIMAL(5,2) DEFAULT 0,
    
    -- Timing
    started_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMPTZ,
    estimated_duration_minutes INTEGER,
    actual_duration_minutes INTEGER,
    
    -- Results
    outputs JSONB DEFAULT '{}',
    artifacts JSONB DEFAULT '{}',
    
    -- Error handling
    error_details JSONB DEFAULT '{}',
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create agent performance metrics table
CREATE TABLE IF NOT EXISTS agent_performance_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    agent_name VARCHAR(100) NOT NULL,
    
    -- Time period for aggregation
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    period_type VARCHAR(20) NOT NULL CHECK (period_type IN ('hour', 'day', 'week', 'month')),
    
    -- Execution metrics
    total_executions INTEGER DEFAULT 0,
    successful_executions INTEGER DEFAULT 0,
    failed_executions INTEGER DEFAULT 0,
    avg_execution_time_ms DECIMAL(10,2),
    
    -- Resource metrics
    total_tokens_used INTEGER DEFAULT 0,
    total_api_calls INTEGER DEFAULT 0,
    avg_memory_used_mb DECIMAL(10,2),
    
    -- Quality metrics
    success_rate DECIMAL(5,2),
    error_rate DECIMAL(5,2),
    timeout_rate DECIMAL(5,2),
    
    -- User satisfaction
    avg_user_rating DECIMAL(3,2),
    total_user_feedback INTEGER DEFAULT 0,
    
    -- Computed at
    computed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(agent_name, period_start, period_type)
);

-- Create indexes for performance
CREATE INDEX idx_agents_tenant_id ON agents(tenant_id);
CREATE INDEX idx_agents_name ON agents(name);
CREATE INDEX idx_agents_category ON agents(category);
CREATE INDEX idx_agents_agent_type ON agents(agent_type);
CREATE INDEX idx_agents_status ON agents(status);

CREATE INDEX idx_agent_executions_tenant_id ON agent_executions(tenant_id);
CREATE INDEX idx_agent_executions_agent_name ON agent_executions(agent_name);
CREATE INDEX idx_agent_executions_session_id ON agent_executions(session_id);
CREATE INDEX idx_agent_executions_project_id ON agent_executions(project_id);
CREATE INDEX idx_agent_executions_status ON agent_executions(status);
CREATE INDEX idx_agent_executions_execution_type ON agent_executions(execution_type);
CREATE INDEX idx_agent_executions_started_at ON agent_executions(started_at DESC);

CREATE INDEX idx_agent_communications_tenant_id ON agent_communications(tenant_id);
CREATE INDEX idx_agent_communications_session_id ON agent_communications(session_id);
CREATE INDEX idx_agent_communications_execution_id ON agent_communications(execution_id);
CREATE INDEX idx_agent_communications_sender ON agent_communications(sender_agent);
CREATE INDEX idx_agent_communications_receiver ON agent_communications(receiver_agent);
CREATE INDEX idx_agent_communications_thread_id ON agent_communications(thread_id);
CREATE INDEX idx_agent_communications_sent_at ON agent_communications(sent_at DESC);

CREATE INDEX idx_orchestration_workflows_tenant_id ON orchestration_workflows(tenant_id);
CREATE INDEX idx_orchestration_workflows_session_id ON orchestration_workflows(session_id);
CREATE INDEX idx_orchestration_workflows_status ON orchestration_workflows(status);
CREATE INDEX idx_orchestration_workflows_workflow_type ON orchestration_workflows(workflow_type);

CREATE INDEX idx_agent_performance_metrics_agent_name ON agent_performance_metrics(agent_name);
CREATE INDEX idx_agent_performance_metrics_period ON agent_performance_metrics(period_start, period_end);
CREATE INDEX idx_agent_performance_metrics_period_type ON agent_performance_metrics(period_type);

-- Add updated_at triggers
CREATE TRIGGER set_timestamp_agents
    BEFORE UPDATE ON agents
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_orchestration_workflows
    BEFORE UPDATE ON orchestration_workflows
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Enable Row Level Security
ALTER TABLE agents ENABLE ROW LEVEL SECURITY;
ALTER TABLE agent_executions ENABLE ROW LEVEL SECURITY;
ALTER TABLE agent_communications ENABLE ROW LEVEL SECURITY;
ALTER TABLE orchestration_workflows ENABLE ROW LEVEL SECURITY;
ALTER TABLE agent_performance_metrics ENABLE ROW LEVEL SECURITY;

-- Create RLS policies for multi-tenancy
CREATE POLICY agents_tenant_isolation ON agents
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY agent_executions_tenant_isolation ON agent_executions
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY agent_communications_tenant_isolation ON agent_communications
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY orchestration_workflows_tenant_isolation ON orchestration_workflows
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY agent_performance_metrics_tenant_isolation ON agent_performance_metrics
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

-- Add table comments
COMMENT ON TABLE agents IS 'Registry of all available agents in the system';
COMMENT ON TABLE agent_executions IS 'Tracking table for all agent task executions';
COMMENT ON TABLE agent_communications IS 'Inter-agent communication and messaging';
COMMENT ON TABLE orchestration_workflows IS 'Workflow definitions and execution tracking';
COMMENT ON TABLE agent_performance_metrics IS 'Aggregated performance metrics by time period';

-- Add column comments
COMMENT ON COLUMN agents.capabilities IS 'List of agent capabilities and skills';
COMMENT ON COLUMN agents.tools IS 'Available tools and integrations for this agent';
COMMENT ON COLUMN agent_executions.input_data IS 'Input parameters and context for execution';
COMMENT ON COLUMN agent_executions.output_data IS 'Results and output from execution';
COMMENT ON COLUMN agent_communications.content IS 'Message content in JSON format';
COMMENT ON COLUMN orchestration_workflows.definition IS 'Workflow DAG structure and step definitions';