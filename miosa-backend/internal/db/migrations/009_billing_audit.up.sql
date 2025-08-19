-- Migration 009: Billing, Usage Metrics, and Audit Logs
-- This migration creates comprehensive billing, usage tracking, and audit systems

-- Create subscriptions table
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Subscription identity
    subscription_id VARCHAR(255) UNIQUE, -- External billing system ID (Stripe, etc.)
    plan_name VARCHAR(100) NOT NULL,
    plan_type VARCHAR(50) DEFAULT 'monthly' CHECK (plan_type IN ('monthly', 'yearly', 'lifetime', 'trial')),
    
    -- Pricing
    price_cents INTEGER NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    billing_cycle_days INTEGER DEFAULT 30,
    
    -- Subscription status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN (
        'trialing', 'active', 'past_due', 'canceled', 'unpaid', 'expired'
    )),
    
    -- Important dates
    trial_start_date DATE,
    trial_end_date DATE,
    current_period_start DATE NOT NULL,
    current_period_end DATE NOT NULL,
    cancel_at_period_end BOOLEAN DEFAULT false,
    canceled_at TIMESTAMPTZ,
    
    -- Features and limits
    features JSONB NOT NULL DEFAULT '{}',
    usage_limits JSONB NOT NULL DEFAULT '{}',
    
    -- Payment method
    payment_method JSONB DEFAULT '{}',
    
    -- Billing address
    billing_address JSONB DEFAULT '{}',
    
    -- Tax information
    tax_ids JSONB DEFAULT '[]',
    tax_percentage DECIMAL(5,2) DEFAULT 0,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create usage metrics table for tracking resource consumption
CREATE TABLE IF NOT EXISTS usage_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    
    -- Metric identification
    metric_type VARCHAR(100) NOT NULL,
    metric_name VARCHAR(255) NOT NULL,
    category VARCHAR(100),
    
    -- Metric values
    value DECIMAL(15,4) NOT NULL,
    unit VARCHAR(50) NOT NULL,
    
    -- Aggregation period
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    aggregation_level VARCHAR(20) CHECK (aggregation_level IN ('minute', 'hour', 'day', 'month')),
    
    -- Context and metadata
    dimensions JSONB DEFAULT '{}',
    tags JSONB DEFAULT '{}',
    
    -- Billing relevance
    is_billable BOOLEAN DEFAULT false,
    billing_rate DECIMAL(10,4),
    
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
) PARTITION BY RANGE (created_at);

-- Create monthly partitions for usage metrics
CREATE TABLE usage_metrics_2024_01 PARTITION OF usage_metrics
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE usage_metrics_2024_02 PARTITION OF usage_metrics
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
CREATE TABLE usage_metrics_2024_03 PARTITION OF usage_metrics
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');
CREATE TABLE usage_metrics_2024_04 PARTITION OF usage_metrics
    FOR VALUES FROM ('2024-04-01') TO ('2024-05-01');
CREATE TABLE usage_metrics_2024_05 PARTITION OF usage_metrics
    FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');
CREATE TABLE usage_metrics_2024_06 PARTITION OF usage_metrics
    FOR VALUES FROM ('2024-06-01') TO ('2024-07-01');
CREATE TABLE usage_metrics_2024_07 PARTITION OF usage_metrics
    FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');
CREATE TABLE usage_metrics_2024_08 PARTITION OF usage_metrics
    FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');
CREATE TABLE usage_metrics_2024_09 PARTITION OF usage_metrics
    FOR VALUES FROM ('2024-09-01') TO ('2024-10-01');
CREATE TABLE usage_metrics_2024_10 PARTITION OF usage_metrics
    FOR VALUES FROM ('2024-10-01') TO ('2024-11-01');
CREATE TABLE usage_metrics_2024_11 PARTITION OF usage_metrics
    FOR VALUES FROM ('2024-11-01') TO ('2024-12-01');
CREATE TABLE usage_metrics_2024_12 PARTITION OF usage_metrics
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

-- Create invoices table
CREATE TABLE IF NOT EXISTS invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    
    -- Invoice identity
    invoice_number VARCHAR(100) UNIQUE NOT NULL,
    external_invoice_id VARCHAR(255), -- Stripe invoice ID, etc.
    
    -- Invoice details
    status VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft', 'open', 'paid', 'void', 'uncollectible')),
    
    -- Amounts
    subtotal_cents INTEGER NOT NULL,
    tax_cents INTEGER DEFAULT 0,
    total_cents INTEGER NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    
    -- Dates
    invoice_date DATE NOT NULL,
    due_date DATE NOT NULL,
    paid_at TIMESTAMPTZ,
    
    -- Period covered
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    
    -- Line items
    line_items JSONB NOT NULL DEFAULT '[]',
    
    -- Payment information
    payment_method JSONB DEFAULT '{}',
    payment_transaction_id VARCHAR(255),
    
    -- PDF and documents
    invoice_pdf_url TEXT,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create comprehensive audit logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    -- Event identification
    event_type VARCHAR(100) NOT NULL,
    event_category VARCHAR(50) NOT NULL CHECK (event_category IN (
        'authentication', 'authorization', 'data_access', 'data_modification',
        'system_configuration', 'user_management', 'billing', 'api_access'
    )),
    
    -- Actor information
    actor_type VARCHAR(20) NOT NULL CHECK (actor_type IN ('user', 'service', 'system', 'api_key')),
    actor_id VARCHAR(255),
    actor_name VARCHAR(255),
    
    -- Target resource
    resource_type VARCHAR(100),
    resource_id VARCHAR(255),
    resource_name VARCHAR(255),
    
    -- Action details
    action VARCHAR(100) NOT NULL,
    description TEXT,
    
    -- Context
    session_id UUID,
    request_id VARCHAR(100),
    correlation_id VARCHAR(100),
    
    -- Request details
    ip_address INET,
    user_agent TEXT,
    request_path VARCHAR(500),
    request_method VARCHAR(10),
    
    -- Data changes
    old_values JSONB,
    new_values JSONB,
    
    -- Outcome
    status VARCHAR(20) DEFAULT 'success' CHECK (status IN ('success', 'failure', 'error', 'warning')),
    error_code VARCHAR(50),
    error_message TEXT,
    
    -- Metadata and tags
    tags JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    
    -- Timestamp
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
) PARTITION BY RANGE (created_at);

-- Create monthly partitions for audit logs
CREATE TABLE audit_logs_2024_01 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE audit_logs_2024_02 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
CREATE TABLE audit_logs_2024_03 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');
CREATE TABLE audit_logs_2024_04 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-04-01') TO ('2024-05-01');
CREATE TABLE audit_logs_2024_05 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');
CREATE TABLE audit_logs_2024_06 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-06-01') TO ('2024-07-01');
CREATE TABLE audit_logs_2024_07 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');
CREATE TABLE audit_logs_2024_08 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');
CREATE TABLE audit_logs_2024_09 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-09-01') TO ('2024-10-01');
CREATE TABLE audit_logs_2024_10 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-10-01') TO ('2024-11-01');
CREATE TABLE audit_logs_2024_11 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-11-01') TO ('2024-12-01');
CREATE TABLE audit_logs_2024_12 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

-- Create API keys table for programmatic access
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Key identification
    key_name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE, -- Hashed version of the key
    key_prefix VARCHAR(20) NOT NULL, -- First few characters for identification
    
    -- Permissions and scope
    scopes JSONB NOT NULL DEFAULT '[]',
    permissions JSONB DEFAULT '{}',
    
    -- Usage limits
    rate_limit_per_hour INTEGER,
    rate_limit_per_day INTEGER,
    
    -- Key lifecycle
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    usage_count INTEGER DEFAULT 0,
    
    -- Status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'expired', 'revoked')),
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create API usage logs table
CREATE TABLE IF NOT EXISTS api_usage_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    api_key_id UUID REFERENCES api_keys(id) ON DELETE CASCADE,
    
    -- Request details
    endpoint VARCHAR(500) NOT NULL,
    method VARCHAR(10) NOT NULL,
    
    -- Response
    status_code INTEGER NOT NULL,
    response_time_ms INTEGER,
    response_size_bytes INTEGER,
    
    -- Usage tracking
    tokens_used INTEGER DEFAULT 0,
    credits_consumed DECIMAL(10,4) DEFAULT 0,
    
    -- Context
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(100),
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
) PARTITION BY RANGE (created_at);

-- Create monthly partitions for API usage logs
CREATE TABLE api_usage_logs_2024_01 PARTITION OF api_usage_logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE api_usage_logs_2024_02 PARTITION OF api_usage_logs
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
CREATE TABLE api_usage_logs_2024_03 PARTITION OF api_usage_logs
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');
CREATE TABLE api_usage_logs_2024_04 PARTITION OF api_usage_logs
    FOR VALUES FROM ('2024-04-01') TO ('2024-05-01');
CREATE TABLE api_usage_logs_2024_05 PARTITION OF api_usage_logs
    FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');
CREATE TABLE api_usage_logs_2024_06 PARTITION OF api_usage_logs
    FOR VALUES FROM ('2024-06-01') TO ('2024-07-01');
CREATE TABLE api_usage_logs_2024_07 PARTITION OF api_usage_logs
    FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');
CREATE TABLE api_usage_logs_2024_08 PARTITION OF api_usage_logs
    FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');
CREATE TABLE api_usage_logs_2024_09 PARTITION OF api_usage_logs
    FOR VALUES FROM ('2024-09-01') TO ('2024-10-01');
CREATE TABLE api_usage_logs_2024_10 PARTITION OF api_usage_logs
    FOR VALUES FROM ('2024-10-01') TO ('2024-11-01');
CREATE TABLE api_usage_logs_2024_11 PARTITION OF api_usage_logs
    FOR VALUES FROM ('2024-11-01') TO ('2024-12-01');
CREATE TABLE api_usage_logs_2024_12 PARTITION OF api_usage_logs
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

-- Create indexes for performance
CREATE INDEX idx_subscriptions_tenant_id ON subscriptions(tenant_id);
CREATE INDEX idx_subscriptions_workspace_id ON subscriptions(workspace_id);
CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_subscription_id ON subscriptions(subscription_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_period ON subscriptions(current_period_start, current_period_end);

-- Indexes for partitioned usage metrics
CREATE INDEX idx_usage_metrics_tenant_id ON usage_metrics(tenant_id, created_at);
CREATE INDEX idx_usage_metrics_workspace_id ON usage_metrics(workspace_id, created_at);
CREATE INDEX idx_usage_metrics_metric_type ON usage_metrics(metric_type, created_at);
CREATE INDEX idx_usage_metrics_period ON usage_metrics(period_start, period_end);
CREATE INDEX idx_usage_metrics_billable ON usage_metrics(is_billable, created_at);

CREATE INDEX idx_invoices_tenant_id ON invoices(tenant_id);
CREATE INDEX idx_invoices_subscription_id ON invoices(subscription_id);
CREATE INDEX idx_invoices_number ON invoices(invoice_number);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_due_date ON invoices(due_date);
CREATE INDEX idx_invoices_period ON invoices(period_start, period_end);

-- Indexes for partitioned audit logs
CREATE INDEX idx_audit_logs_tenant_id ON audit_logs(tenant_id, created_at);
CREATE INDEX idx_audit_logs_event_type ON audit_logs(event_type, created_at);
CREATE INDEX idx_audit_logs_event_category ON audit_logs(event_category, created_at);
CREATE INDEX idx_audit_logs_actor ON audit_logs(actor_type, actor_id, created_at);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id, created_at);
CREATE INDEX idx_audit_logs_session_id ON audit_logs(session_id, created_at);
CREATE INDEX idx_audit_logs_ip_address ON audit_logs(ip_address, created_at);

CREATE INDEX idx_api_keys_tenant_id ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_workspace_id ON api_keys(workspace_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_key_prefix ON api_keys(key_prefix);
CREATE INDEX idx_api_keys_status ON api_keys(status);
CREATE INDEX idx_api_keys_expires_at ON api_keys(expires_at);

-- Indexes for partitioned API usage logs
CREATE INDEX idx_api_usage_logs_tenant_id ON api_usage_logs(tenant_id, created_at);
CREATE INDEX idx_api_usage_logs_api_key_id ON api_usage_logs(api_key_id, created_at);
CREATE INDEX idx_api_usage_logs_endpoint ON api_usage_logs(endpoint, created_at);
CREATE INDEX idx_api_usage_logs_status_code ON api_usage_logs(status_code, created_at);

-- Add updated_at triggers
CREATE TRIGGER set_timestamp_subscriptions
    BEFORE UPDATE ON subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_invoices
    BEFORE UPDATE ON invoices
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_api_keys
    BEFORE UPDATE ON api_keys
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Enable Row Level Security
ALTER TABLE subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE usage_metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE invoices ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE api_usage_logs ENABLE ROW LEVEL SECURITY;

-- Create RLS policies for multi-tenancy
CREATE POLICY subscriptions_tenant_isolation ON subscriptions
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY usage_metrics_tenant_isolation ON usage_metrics
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY invoices_tenant_isolation ON invoices
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY audit_logs_tenant_isolation ON audit_logs
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY api_keys_tenant_isolation ON api_keys
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY api_usage_logs_tenant_isolation ON api_usage_logs
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

-- Add table comments
COMMENT ON TABLE subscriptions IS 'Customer subscriptions with billing and feature management';
COMMENT ON TABLE usage_metrics IS 'Partitioned usage metrics for billing and analytics';
COMMENT ON TABLE invoices IS 'Generated invoices with line items and payment tracking';
COMMENT ON TABLE audit_logs IS 'Comprehensive audit trail of all system activities';
COMMENT ON TABLE api_keys IS 'API keys for programmatic access with scopes and limits';
COMMENT ON TABLE api_usage_logs IS 'Partitioned API usage logs for monitoring and billing';

-- Add column comments
COMMENT ON COLUMN subscriptions.features IS 'Available features for this subscription plan';
COMMENT ON COLUMN subscriptions.usage_limits IS 'Usage limits and quotas for this subscription';
COMMENT ON COLUMN usage_metrics.dimensions IS 'Additional metadata dimensions for the metric';
COMMENT ON COLUMN audit_logs.old_values IS 'Previous values before modification';
COMMENT ON COLUMN audit_logs.new_values IS 'New values after modification';
COMMENT ON COLUMN api_keys.key_hash IS 'Securely hashed API key for verification';