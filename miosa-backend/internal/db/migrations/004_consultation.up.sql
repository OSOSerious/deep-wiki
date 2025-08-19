-- Migration 004: 3-Layer Consultation Flow
-- This migration creates the consultation system with sessions, messages, and insights

-- Create consultation sessions table
CREATE TABLE IF NOT EXISTS consultation_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    
    -- Session identity and context
    title VARCHAR(255) NOT NULL,
    description TEXT,
    session_type VARCHAR(50) DEFAULT 'general' CHECK (session_type IN ('general', 'debugging', 'architecture', 'optimization', 'review')),
    
    -- Session flow
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'paused', 'completed', 'abandoned')),
    priority VARCHAR(10) DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    
    -- Context and requirements
    business_context TEXT,
    technical_context TEXT,
    requirements JSONB DEFAULT '{}',
    constraints JSONB DEFAULT '{}',
    
    -- Progress tracking
    current_phase VARCHAR(50) DEFAULT 'analysis',
    phases_completed TEXT[],
    estimated_duration_minutes INTEGER,
    actual_duration_minutes INTEGER,
    
    -- Agent assignments
    primary_agent VARCHAR(100),
    assigned_agents TEXT[],
    
    -- Timing
    started_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    last_activity_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMPTZ,
    
    -- Results
    summary TEXT,
    recommendations JSONB DEFAULT '{}',
    deliverables JSONB DEFAULT '{}',
    
    -- Feedback
    user_rating INTEGER CHECK (user_rating >= 1 AND user_rating <= 5),
    user_feedback TEXT,
    
    -- Metadata
    tags TEXT[],
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create consultation messages table
CREATE TABLE IF NOT EXISTS consultation_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES consultation_sessions(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    
    -- Message source
    sender_type VARCHAR(20) NOT NULL CHECK (sender_type IN ('user', 'agent', 'system')),
    sender_id VARCHAR(255), -- user_id or agent_name
    sender_name VARCHAR(255),
    
    -- Message content
    message_type VARCHAR(50) DEFAULT 'text' CHECK (message_type IN ('text', 'code', 'file', 'image', 'command', 'result', 'error', 'suggestion')),
    content TEXT NOT NULL,
    formatted_content JSONB,
    
    -- Message context
    phase VARCHAR(50),
    thread_id UUID, -- For message threading
    parent_message_id UUID REFERENCES consultation_messages(id),
    
    -- Attachments and references
    attachments JSONB DEFAULT '[]',
    code_snippets JSONB DEFAULT '[]',
    file_references TEXT[],
    
    -- AI processing
    embedding vector(1536),
    tokens_used INTEGER,
    processing_time_ms INTEGER,
    
    -- Message status
    status VARCHAR(20) DEFAULT 'sent' CHECK (status IN ('sending', 'sent', 'delivered', 'read', 'error')),
    is_internal BOOLEAN DEFAULT false,
    requires_action BOOLEAN DEFAULT false,
    
    -- Reactions and feedback
    reactions JSONB DEFAULT '{}',
    user_feedback TEXT,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    -- Sequence for ordering
    sequence_number SERIAL
);

-- Create consultation insights table
CREATE TABLE IF NOT EXISTS consultation_insights (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES consultation_sessions(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    
    -- Insight classification
    insight_type VARCHAR(50) NOT NULL CHECK (insight_type IN ('pattern', 'recommendation', 'warning', 'opportunity', 'best_practice', 'anti_pattern')),
    category VARCHAR(100),
    severity VARCHAR(10) DEFAULT 'info' CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
    
    -- Insight content
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    details JSONB DEFAULT '{}',
    
    -- Context and evidence
    context JSONB DEFAULT '{}',
    evidence TEXT[],
    code_references JSONB DEFAULT '[]',
    
    -- AI analysis
    confidence_score DECIMAL(3,2) CHECK (confidence_score >= 0 AND confidence_score <= 1),
    generated_by VARCHAR(100), -- Agent that generated this insight
    embedding vector(1536),
    
    -- Actionability
    is_actionable BOOLEAN DEFAULT true,
    estimated_effort VARCHAR(20) CHECK (estimated_effort IN ('trivial', 'minor', 'moderate', 'major', 'epic')),
    suggested_actions JSONB DEFAULT '[]',
    
    -- User interaction
    user_status VARCHAR(20) DEFAULT 'new' CHECK (user_status IN ('new', 'acknowledged', 'accepted', 'rejected', 'implemented')),
    user_notes TEXT,
    
    -- Relationships
    related_insights UUID[],
    blocks_insights UUID[],
    
    -- Tracking
    times_referenced INTEGER DEFAULT 0,
    last_referenced_at TIMESTAMPTZ,
    
    -- Metadata
    tags TEXT[],
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create consultation session analytics table
CREATE TABLE IF NOT EXISTS consultation_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES consultation_sessions(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    
    -- Analytics data
    total_messages INTEGER DEFAULT 0,
    user_messages INTEGER DEFAULT 0,
    agent_messages INTEGER DEFAULT 0,
    system_messages INTEGER DEFAULT 0,
    
    -- Token usage
    total_tokens_used INTEGER DEFAULT 0,
    input_tokens INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    
    -- Response metrics
    avg_response_time_ms DECIMAL(10,2),
    max_response_time_ms INTEGER,
    total_processing_time_ms INTEGER,
    
    -- Insights generated
    insights_generated INTEGER DEFAULT 0,
    insights_by_type JSONB DEFAULT '{}',
    
    -- User engagement
    user_satisfaction_score DECIMAL(3,2),
    session_completion_rate DECIMAL(3,2),
    
    -- Performance metrics
    error_count INTEGER DEFAULT 0,
    retry_count INTEGER DEFAULT 0,
    
    -- Metadata
    computed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX idx_consultation_sessions_tenant_id ON consultation_sessions(tenant_id);
CREATE INDEX idx_consultation_sessions_user_id ON consultation_sessions(user_id);
CREATE INDEX idx_consultation_sessions_project_id ON consultation_sessions(project_id);
CREATE INDEX idx_consultation_sessions_workspace_id ON consultation_sessions(workspace_id);
CREATE INDEX idx_consultation_sessions_status ON consultation_sessions(status);
CREATE INDEX idx_consultation_sessions_session_type ON consultation_sessions(session_type);
CREATE INDEX idx_consultation_sessions_started_at ON consultation_sessions(started_at DESC);

CREATE INDEX idx_consultation_messages_session_id ON consultation_messages(session_id);
CREATE INDEX idx_consultation_messages_tenant_id ON consultation_messages(tenant_id);
CREATE INDEX idx_consultation_messages_sender_type ON consultation_messages(sender_type);
CREATE INDEX idx_consultation_messages_message_type ON consultation_messages(message_type);
CREATE INDEX idx_consultation_messages_created_at ON consultation_messages(created_at DESC);
CREATE INDEX idx_consultation_messages_sequence ON consultation_messages(session_id, sequence_number);

CREATE INDEX idx_consultation_insights_session_id ON consultation_insights(session_id);
CREATE INDEX idx_consultation_insights_tenant_id ON consultation_insights(tenant_id);
CREATE INDEX idx_consultation_insights_insight_type ON consultation_insights(insight_type);
CREATE INDEX idx_consultation_insights_category ON consultation_insights(category);
CREATE INDEX idx_consultation_insights_severity ON consultation_insights(severity);
CREATE INDEX idx_consultation_insights_user_status ON consultation_insights(user_status);
CREATE INDEX idx_consultation_insights_confidence ON consultation_insights(confidence_score DESC);

CREATE INDEX idx_consultation_analytics_session_id ON consultation_analytics(session_id);
CREATE INDEX idx_consultation_analytics_tenant_id ON consultation_analytics(tenant_id);

-- Vector similarity search indexes using HNSW
CREATE INDEX idx_consultation_messages_embedding ON consultation_messages
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

CREATE INDEX idx_consultation_insights_embedding ON consultation_insights
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- Add updated_at triggers
CREATE TRIGGER set_timestamp_consultation_sessions
    BEFORE UPDATE ON consultation_sessions
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_consultation_insights
    BEFORE UPDATE ON consultation_insights
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Enable Row Level Security
ALTER TABLE consultation_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE consultation_messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE consultation_insights ENABLE ROW LEVEL SECURITY;
ALTER TABLE consultation_analytics ENABLE ROW LEVEL SECURITY;

-- Create RLS policies for multi-tenancy
CREATE POLICY consultation_sessions_tenant_isolation ON consultation_sessions
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY consultation_messages_tenant_isolation ON consultation_messages
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY consultation_insights_tenant_isolation ON consultation_insights
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY consultation_analytics_tenant_isolation ON consultation_analytics
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

-- Add table comments
COMMENT ON TABLE consultation_sessions IS '3-layer consultation sessions with context and progress tracking';
COMMENT ON TABLE consultation_messages IS 'Conversation messages with AI embeddings and rich content';
COMMENT ON TABLE consultation_insights IS 'AI-generated insights and recommendations from consultations';
COMMENT ON TABLE consultation_analytics IS 'Session analytics and performance metrics';

-- Add column comments
COMMENT ON COLUMN consultation_sessions.requirements IS 'Business and technical requirements in JSON format';
COMMENT ON COLUMN consultation_sessions.constraints IS 'Project constraints and limitations';
COMMENT ON COLUMN consultation_messages.embedding IS 'Message embedding for semantic search (1536 dimensions)';
COMMENT ON COLUMN consultation_messages.formatted_content IS 'Rich formatted content (markdown, code blocks, etc.)';
COMMENT ON COLUMN consultation_insights.confidence_score IS 'AI confidence in insight accuracy (0.0-1.0)';
COMMENT ON COLUMN consultation_insights.embedding IS 'Insight embedding for similarity search (1536 dimensions)';