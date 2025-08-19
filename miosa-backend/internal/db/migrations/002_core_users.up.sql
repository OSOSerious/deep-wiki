-- Migration 002: Core Users and Authentication
-- This migration creates the foundational user management system with business context

-- Enable Row Level Security
SET row_security = on;

-- Create users table with business context
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    email_verified BOOLEAN DEFAULT false,
    password_hash VARCHAR(255),
    full_name VARCHAR(255),
    avatar_url TEXT,
    
    -- Business Context
    company_name VARCHAR(255),
    job_title VARCHAR(255),
    industry VARCHAR(100),
    team_size INT CHECK (team_size >= 1),
    use_case TEXT,
    
    -- Authentication
    auth_provider VARCHAR(50) DEFAULT 'local' CHECK (auth_provider IN ('local', 'google', 'github', 'microsoft')),
    auth_provider_id VARCHAR(255),
    two_factor_enabled BOOLEAN DEFAULT false,
    two_factor_secret VARCHAR(255),
    
    -- Account status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'pending_verification', 'deactivated')),
    last_login_at TIMESTAMPTZ,
    last_active_at TIMESTAMPTZ,
    
    -- Preferences
    timezone VARCHAR(50) DEFAULT 'UTC',
    language VARCHAR(10) DEFAULT 'en',
    theme VARCHAR(20) DEFAULT 'system' CHECK (theme IN ('light', 'dark', 'system')),
    
    -- Onboarding
    onboarding_completed BOOLEAN DEFAULT false,
    onboarding_step VARCHAR(50),
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create user sessions table for authentication tracking
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    session_token VARCHAR(255) NOT NULL UNIQUE,
    refresh_token VARCHAR(255),
    device_info JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    last_accessed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create password reset tokens table
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create email verification tokens table
CREATE TABLE IF NOT EXISTS email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    token VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX idx_users_tenant_id ON users(tenant_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_auth_provider ON users(auth_provider, auth_provider_id);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_last_active ON users(last_active_at);

CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_tenant_id ON user_sessions(tenant_id);
CREATE INDEX idx_user_sessions_token ON user_sessions(session_token);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions(expires_at);

CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
CREATE INDEX idx_password_reset_tokens_token ON password_reset_tokens(token);
CREATE INDEX idx_password_reset_tokens_expires_at ON password_reset_tokens(expires_at);

CREATE INDEX idx_email_verification_tokens_user_id ON email_verification_tokens(user_id);
CREATE INDEX idx_email_verification_tokens_token ON email_verification_tokens(token);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Add updated_at triggers
CREATE TRIGGER set_timestamp_users
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Enable Row Level Security on all tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE password_reset_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE email_verification_tokens ENABLE ROW LEVEL SECURITY;

-- Create RLS policies for multi-tenancy
CREATE POLICY users_tenant_isolation ON users
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY user_sessions_tenant_isolation ON user_sessions
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY password_reset_tokens_tenant_isolation ON password_reset_tokens
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY email_verification_tokens_tenant_isolation ON email_verification_tokens
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

-- Add table comments
COMMENT ON TABLE users IS 'Core user accounts with business context and authentication';
COMMENT ON TABLE user_sessions IS 'Active user sessions for authentication tracking';
COMMENT ON TABLE password_reset_tokens IS 'Temporary tokens for password reset flows';
COMMENT ON TABLE email_verification_tokens IS 'Tokens for email verification process';

-- Add column comments
COMMENT ON COLUMN users.tenant_id IS 'Multi-tenant isolation key';
COMMENT ON COLUMN users.metadata IS 'Flexible JSON storage for additional user data';
COMMENT ON COLUMN users.onboarding_step IS 'Current step in the onboarding flow';
COMMENT ON COLUMN user_sessions.device_info IS 'Browser/device information for security';
COMMENT ON COLUMN user_sessions.expires_at IS 'Session expiration timestamp';