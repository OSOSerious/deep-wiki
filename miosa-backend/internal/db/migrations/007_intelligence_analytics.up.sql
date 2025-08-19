-- Migration 007: Business Intelligence and Analytics
-- This migration creates business insights, predictive analytics, and communication intelligence

-- Create business insights table
CREATE TABLE IF NOT EXISTS business_insights (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    
    -- Insight classification
    insight_type VARCHAR(50) NOT NULL CHECK (insight_type IN (
        'usage_pattern', 'performance_trend', 'cost_optimization', 
        'security_risk', 'productivity_metric', 'user_behavior', 'predictive_alert'
    )),
    category VARCHAR(100),
    severity VARCHAR(10) DEFAULT 'info' CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
    
    -- Insight content
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    summary TEXT,
    
    -- Insight data and metrics
    metrics JSONB NOT NULL DEFAULT '{}',
    historical_data JSONB DEFAULT '{}',
    predictions JSONB DEFAULT '{}',
    
    -- Context and scope
    scope VARCHAR(50) CHECK (scope IN ('user', 'project', 'workspace', 'organization')),
    affected_resources JSONB DEFAULT '[]',
    time_period JSONB DEFAULT '{}',
    
    -- AI analysis
    confidence_score DECIMAL(3,2) CHECK (confidence_score >= 0 AND confidence_score <= 1),
    generated_by VARCHAR(100),
    model_used VARCHAR(100),
    
    -- Actionability
    is_actionable BOOLEAN DEFAULT true,
    recommended_actions JSONB DEFAULT '[]',
    estimated_impact VARCHAR(20) CHECK (estimated_impact IN ('low', 'medium', 'high', 'critical')),
    
    -- User interaction
    status VARCHAR(20) DEFAULT 'new' CHECK (status IN ('new', 'acknowledged', 'in_progress', 'resolved', 'dismissed')),
    user_notes TEXT,
    
    -- Tracking
    views_count INTEGER DEFAULT 0,
    last_viewed_at TIMESTAMPTZ,
    
    -- Metadata
    tags TEXT[],
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create predictive analytics table
CREATE TABLE IF NOT EXISTS predictive_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    
    -- Prediction details
    prediction_type VARCHAR(50) NOT NULL CHECK (prediction_type IN (
        'usage_forecast', 'cost_projection', 'performance_prediction', 
        'churn_risk', 'capacity_planning', 'trend_analysis'
    )),
    model_name VARCHAR(100) NOT NULL,
    model_version VARCHAR(20) DEFAULT '1.0',
    
    -- Input data and features
    input_features JSONB NOT NULL,
    feature_importance JSONB DEFAULT '{}',
    training_period JSONB NOT NULL,
    
    -- Predictions and results
    predictions JSONB NOT NULL,
    confidence_intervals JSONB DEFAULT '{}',
    probability_scores JSONB DEFAULT '{}',
    
    -- Model performance
    accuracy_score DECIMAL(5,4),
    precision_score DECIMAL(5,4),
    recall_score DECIMAL(5,4),
    f1_score DECIMAL(5,4),
    
    -- Prediction horizon
    forecast_period VARCHAR(50) NOT NULL,
    forecast_start_date DATE NOT NULL,
    forecast_end_date DATE NOT NULL,
    
    -- Validation and monitoring
    is_validated BOOLEAN DEFAULT false,
    validation_results JSONB DEFAULT '{}',
    drift_score DECIMAL(5,4),
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create email intelligence table
CREATE TABLE IF NOT EXISTS email_intelligence (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Email identification
    email_id VARCHAR(255), -- External email system ID
    message_id VARCHAR(255),
    thread_id VARCHAR(255),
    
    -- Email content
    subject VARCHAR(500),
    sender_email VARCHAR(255),
    sender_name VARCHAR(255),
    recipients JSONB DEFAULT '[]',
    
    -- Content analysis
    content_summary TEXT,
    key_topics JSONB DEFAULT '[]',
    sentiment VARCHAR(20) CHECK (sentiment IN ('positive', 'negative', 'neutral')),
    sentiment_score DECIMAL(3,2),
    
    -- Intent and classification
    intent VARCHAR(100),
    category VARCHAR(100),
    priority VARCHAR(10) CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    
    -- Project and context linking
    related_projects UUID[],
    mentioned_entities JSONB DEFAULT '[]',
    
    -- Action items and tasks
    action_items JSONB DEFAULT '[]',
    deadlines JSONB DEFAULT '[]',
    
    -- AI processing
    embedding vector(1536),
    processed_by VARCHAR(100),
    processing_confidence DECIMAL(3,2),
    
    -- Email metadata
    email_timestamp TIMESTAMPTZ,
    is_internal BOOLEAN DEFAULT false,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create meeting intelligence table
CREATE TABLE IF NOT EXISTS meeting_intelligence (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    
    -- Meeting identification
    meeting_id VARCHAR(255), -- External meeting system ID
    title VARCHAR(500),
    meeting_type VARCHAR(50) CHECK (meeting_type IN ('standup', 'planning', 'review', 'retrospective', 'other')),
    
    -- Meeting details
    start_time TIMESTAMPTZ,
    end_time TIMESTAMPTZ,
    duration_minutes INTEGER,
    participants JSONB DEFAULT '[]',
    
    -- Meeting content
    transcript TEXT,
    summary TEXT,
    key_topics JSONB DEFAULT '[]',
    
    -- Analysis results
    sentiment_analysis JSONB DEFAULT '{}',
    speaker_analysis JSONB DEFAULT '{}',
    engagement_metrics JSONB DEFAULT '{}',
    
    -- Action items and outcomes
    action_items JSONB DEFAULT '[]',
    decisions_made JSONB DEFAULT '[]',
    next_steps JSONB DEFAULT '[]',
    
    -- Project and context
    related_projects UUID[],
    mentioned_entities JSONB DEFAULT '[]',
    
    -- AI processing
    embedding vector(1536),
    processed_by VARCHAR(100),
    processing_confidence DECIMAL(3,2),
    
    -- Integration details
    calendar_system VARCHAR(50),
    recording_url TEXT,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create analytics dashboards table
CREATE TABLE IF NOT EXISTS analytics_dashboards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Dashboard details
    name VARCHAR(255) NOT NULL,
    description TEXT,
    dashboard_type VARCHAR(50) CHECK (dashboard_type IN ('executive', 'operational', 'technical', 'custom')),
    
    -- Dashboard configuration
    layout JSONB NOT NULL DEFAULT '{}',
    widgets JSONB NOT NULL DEFAULT '[]',
    filters JSONB DEFAULT '{}',
    
    -- Data sources
    data_sources JSONB NOT NULL DEFAULT '[]',
    refresh_interval_minutes INTEGER DEFAULT 60,
    
    -- Access and sharing
    is_public BOOLEAN DEFAULT false,
    shared_with JSONB DEFAULT '[]',
    
    -- Usage tracking
    view_count INTEGER DEFAULT 0,
    last_viewed_at TIMESTAMPTZ,
    
    -- Status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'archived', 'draft')),
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create usage analytics table
CREATE TABLE IF NOT EXISTS usage_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    
    -- Event details
    event_type VARCHAR(100) NOT NULL,
    event_name VARCHAR(255) NOT NULL,
    event_category VARCHAR(100),
    
    -- Context
    session_id UUID,
    page_path VARCHAR(500),
    user_agent TEXT,
    ip_address INET,
    
    -- Event data
    properties JSONB DEFAULT '{}',
    metrics JSONB DEFAULT '{}',
    
    -- Timing
    event_timestamp TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    duration_ms INTEGER,
    
    -- Device and platform
    device_type VARCHAR(50),
    platform VARCHAR(50),
    browser VARCHAR(100),
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
) PARTITION BY RANGE (created_at);

-- Create monthly partitions for usage analytics
CREATE TABLE usage_analytics_2024_01 PARTITION OF usage_analytics
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE usage_analytics_2024_02 PARTITION OF usage_analytics
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
CREATE TABLE usage_analytics_2024_03 PARTITION OF usage_analytics
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');
CREATE TABLE usage_analytics_2024_04 PARTITION OF usage_analytics
    FOR VALUES FROM ('2024-04-01') TO ('2024-05-01');
CREATE TABLE usage_analytics_2024_05 PARTITION OF usage_analytics
    FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');
CREATE TABLE usage_analytics_2024_06 PARTITION OF usage_analytics
    FOR VALUES FROM ('2024-06-01') TO ('2024-07-01');
CREATE TABLE usage_analytics_2024_07 PARTITION OF usage_analytics
    FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');
CREATE TABLE usage_analytics_2024_08 PARTITION OF usage_analytics
    FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');
CREATE TABLE usage_analytics_2024_09 PARTITION OF usage_analytics
    FOR VALUES FROM ('2024-09-01') TO ('2024-10-01');
CREATE TABLE usage_analytics_2024_10 PARTITION OF usage_analytics
    FOR VALUES FROM ('2024-10-01') TO ('2024-11-01');
CREATE TABLE usage_analytics_2024_11 PARTITION OF usage_analytics
    FOR VALUES FROM ('2024-11-01') TO ('2024-12-01');
CREATE TABLE usage_analytics_2024_12 PARTITION OF usage_analytics
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

-- Create indexes for performance
CREATE INDEX idx_business_insights_tenant_id ON business_insights(tenant_id);
CREATE INDEX idx_business_insights_workspace_id ON business_insights(workspace_id);
CREATE INDEX idx_business_insights_insight_type ON business_insights(insight_type);
CREATE INDEX idx_business_insights_severity ON business_insights(severity);
CREATE INDEX idx_business_insights_status ON business_insights(status);
CREATE INDEX idx_business_insights_created_at ON business_insights(created_at DESC);

CREATE INDEX idx_predictive_analytics_tenant_id ON predictive_analytics(tenant_id);
CREATE INDEX idx_predictive_analytics_workspace_id ON predictive_analytics(workspace_id);
CREATE INDEX idx_predictive_analytics_prediction_type ON predictive_analytics(prediction_type);
CREATE INDEX idx_predictive_analytics_forecast_period ON predictive_analytics(forecast_start_date, forecast_end_date);

CREATE INDEX idx_email_intelligence_tenant_id ON email_intelligence(tenant_id);
CREATE INDEX idx_email_intelligence_user_id ON email_intelligence(user_id);
CREATE INDEX idx_email_intelligence_sender_email ON email_intelligence(sender_email);
CREATE INDEX idx_email_intelligence_email_timestamp ON email_intelligence(email_timestamp DESC);

CREATE INDEX idx_meeting_intelligence_tenant_id ON meeting_intelligence(tenant_id);
CREATE INDEX idx_meeting_intelligence_workspace_id ON meeting_intelligence(workspace_id);
CREATE INDEX idx_meeting_intelligence_meeting_type ON meeting_intelligence(meeting_type);
CREATE INDEX idx_meeting_intelligence_start_time ON meeting_intelligence(start_time DESC);

CREATE INDEX idx_analytics_dashboards_tenant_id ON analytics_dashboards(tenant_id);
CREATE INDEX idx_analytics_dashboards_workspace_id ON analytics_dashboards(workspace_id);
CREATE INDEX idx_analytics_dashboards_created_by ON analytics_dashboards(created_by);
CREATE INDEX idx_analytics_dashboards_dashboard_type ON analytics_dashboards(dashboard_type);

-- Indexes for partitioned usage analytics
CREATE INDEX idx_usage_analytics_tenant_id ON usage_analytics(tenant_id, created_at);
CREATE INDEX idx_usage_analytics_user_id ON usage_analytics(user_id, created_at);
CREATE INDEX idx_usage_analytics_workspace_id ON usage_analytics(workspace_id, created_at);
CREATE INDEX idx_usage_analytics_event_type ON usage_analytics(event_type, created_at);
CREATE INDEX idx_usage_analytics_event_timestamp ON usage_analytics(event_timestamp DESC);

-- Vector similarity search indexes
CREATE INDEX idx_email_intelligence_embedding ON email_intelligence
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

CREATE INDEX idx_meeting_intelligence_embedding ON meeting_intelligence
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- Add updated_at triggers
CREATE TRIGGER set_timestamp_business_insights
    BEFORE UPDATE ON business_insights
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_predictive_analytics
    BEFORE UPDATE ON predictive_analytics
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_email_intelligence
    BEFORE UPDATE ON email_intelligence
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_meeting_intelligence
    BEFORE UPDATE ON meeting_intelligence
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_analytics_dashboards
    BEFORE UPDATE ON analytics_dashboards
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Enable Row Level Security
ALTER TABLE business_insights ENABLE ROW LEVEL SECURITY;
ALTER TABLE predictive_analytics ENABLE ROW LEVEL SECURITY;
ALTER TABLE email_intelligence ENABLE ROW LEVEL SECURITY;
ALTER TABLE meeting_intelligence ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_dashboards ENABLE ROW LEVEL SECURITY;
ALTER TABLE usage_analytics ENABLE ROW LEVEL SECURITY;

-- Create RLS policies for multi-tenancy
CREATE POLICY business_insights_tenant_isolation ON business_insights
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY predictive_analytics_tenant_isolation ON predictive_analytics
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY email_intelligence_tenant_isolation ON email_intelligence
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY meeting_intelligence_tenant_isolation ON meeting_intelligence
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY analytics_dashboards_tenant_isolation ON analytics_dashboards
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY usage_analytics_tenant_isolation ON usage_analytics
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

-- Add table comments
COMMENT ON TABLE business_insights IS 'AI-generated business insights and recommendations';
COMMENT ON TABLE predictive_analytics IS 'Machine learning predictions and forecasts';
COMMENT ON TABLE email_intelligence IS 'Email content analysis and intelligence extraction';
COMMENT ON TABLE meeting_intelligence IS 'Meeting transcript analysis and insights';
COMMENT ON TABLE analytics_dashboards IS 'Custom analytics dashboards and visualizations';
COMMENT ON TABLE usage_analytics IS 'Partitioned usage and behavior analytics';

-- Add column comments
COMMENT ON COLUMN business_insights.metrics IS 'Quantitative metrics supporting the insight';
COMMENT ON COLUMN predictive_analytics.predictions IS 'Model predictions and forecast values';
COMMENT ON COLUMN email_intelligence.embedding IS 'Email content embedding for similarity search';
COMMENT ON COLUMN meeting_intelligence.embedding IS 'Meeting content embedding for analysis';