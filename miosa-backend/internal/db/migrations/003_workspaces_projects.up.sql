-- Migration 003: Workspaces and Projects
-- This migration creates the workspace and project management system with builds tracking

-- Create workspaces table
CREATE TABLE IF NOT EXISTS workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Workspace settings
    is_public BOOLEAN DEFAULT false,
    allow_team_creation BOOLEAN DEFAULT true,
    max_projects INTEGER DEFAULT 10,
    max_team_members INTEGER DEFAULT 5,
    
    -- Billing context
    plan_type VARCHAR(20) DEFAULT 'free' CHECK (plan_type IN ('free', 'starter', 'pro', 'enterprise')),
    billing_email VARCHAR(255),
    
    -- Workspace metadata
    avatar_url TEXT,
    website_url TEXT,
    github_org VARCHAR(255),
    
    -- Status and activity
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'archived')),
    last_activity_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    -- Metadata
    settings JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(tenant_id, slug)
);

-- Create workspace members table for team management
CREATE TABLE IF NOT EXISTS workspace_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    
    -- Role and permissions
    role VARCHAR(20) DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member', 'guest')),
    permissions JSONB DEFAULT '{}',
    
    -- Invitation tracking
    invited_by UUID REFERENCES users(id),
    invited_at TIMESTAMPTZ,
    joined_at TIMESTAMPTZ,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('pending', 'active', 'suspended')),
    
    -- Activity
    last_active_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(workspace_id, user_id)
);

-- Create projects table
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Project identity
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    description TEXT,
    readme TEXT,
    
    -- Project type and framework
    project_type VARCHAR(50) DEFAULT 'web' CHECK (project_type IN ('web', 'mobile', 'desktop', 'api', 'library', 'other')),
    framework VARCHAR(100),
    language VARCHAR(50),
    
    -- Repository integration
    git_provider VARCHAR(20) CHECK (git_provider IN ('github', 'gitlab', 'bitbucket')),
    git_repo_url TEXT,
    git_branch VARCHAR(255) DEFAULT 'main',
    
    -- Deployment settings
    domain_name VARCHAR(255),
    custom_domain VARCHAR(255),
    environment_vars JSONB DEFAULT '{}',
    
    -- Project status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'archived', 'template')),
    is_template BOOLEAN DEFAULT false,
    template_category VARCHAR(100),
    
    -- Activity tracking
    last_build_at TIMESTAMPTZ,
    last_deployment_at TIMESTAMPTZ,
    last_activity_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    -- Statistics
    total_builds INTEGER DEFAULT 0,
    total_deployments INTEGER DEFAULT 0,
    total_sessions INTEGER DEFAULT 0,
    
    -- Metadata
    tags TEXT[],
    settings JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(workspace_id, slug)
);

-- Create project builds table for tracking build history
CREATE TABLE IF NOT EXISTS project_builds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    triggered_by UUID REFERENCES users(id),
    
    -- Build identification
    build_number SERIAL,
    commit_sha VARCHAR(40),
    branch VARCHAR(255),
    
    -- Build process
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'building', 'success', 'failed', 'cancelled')),
    build_command TEXT,
    build_logs TEXT,
    
    -- Timing
    started_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMPTZ,
    duration_seconds INTEGER,
    
    -- Deployment info
    deployment_url TEXT,
    preview_url TEXT,
    
    -- Build artifacts
    artifacts JSONB DEFAULT '{}',
    
    -- Error tracking
    error_message TEXT,
    error_code VARCHAR(50),
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create project templates table
CREATE TABLE IF NOT EXISTS project_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Template identity
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    tags TEXT[],
    
    -- Template content
    framework VARCHAR(100),
    language VARCHAR(50),
    template_files JSONB NOT NULL DEFAULT '{}',
    
    -- Template settings
    is_public BOOLEAN DEFAULT false,
    is_featured BOOLEAN DEFAULT false,
    sort_order INTEGER DEFAULT 0,
    
    -- Usage statistics
    usage_count INTEGER DEFAULT 0,
    
    -- Metadata
    github_repo_url TEXT,
    demo_url TEXT,
    screenshot_url TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX idx_workspaces_tenant_id ON workspaces(tenant_id);
CREATE INDEX idx_workspaces_owner_id ON workspaces(owner_id);
CREATE INDEX idx_workspaces_slug ON workspaces(tenant_id, slug);
CREATE INDEX idx_workspaces_status ON workspaces(status);
CREATE INDEX idx_workspaces_plan_type ON workspaces(plan_type);

CREATE INDEX idx_workspace_members_workspace_id ON workspace_members(workspace_id);
CREATE INDEX idx_workspace_members_user_id ON workspace_members(user_id);
CREATE INDEX idx_workspace_members_tenant_id ON workspace_members(tenant_id);
CREATE INDEX idx_workspace_members_role ON workspace_members(role);

CREATE INDEX idx_projects_workspace_id ON projects(workspace_id);
CREATE INDEX idx_projects_tenant_id ON projects(tenant_id);
CREATE INDEX idx_projects_owner_id ON projects(owner_id);
CREATE INDEX idx_projects_slug ON projects(workspace_id, slug);
CREATE INDEX idx_projects_status ON projects(status);
CREATE INDEX idx_projects_framework ON projects(framework);
CREATE INDEX idx_projects_is_template ON projects(is_template);

CREATE INDEX idx_project_builds_project_id ON project_builds(project_id);
CREATE INDEX idx_project_builds_tenant_id ON project_builds(tenant_id);
CREATE INDEX idx_project_builds_status ON project_builds(status);
CREATE INDEX idx_project_builds_started_at ON project_builds(started_at DESC);
CREATE INDEX idx_project_builds_build_number ON project_builds(project_id, build_number DESC);

CREATE INDEX idx_project_templates_tenant_id ON project_templates(tenant_id);
CREATE INDEX idx_project_templates_category ON project_templates(category);
CREATE INDEX idx_project_templates_public ON project_templates(is_public);
CREATE INDEX idx_project_templates_featured ON project_templates(is_featured);

-- Add updated_at triggers
CREATE TRIGGER set_timestamp_workspaces
    BEFORE UPDATE ON workspaces
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_workspace_members
    BEFORE UPDATE ON workspace_members
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_projects
    BEFORE UPDATE ON projects
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_project_templates
    BEFORE UPDATE ON project_templates
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Enable Row Level Security
ALTER TABLE workspaces ENABLE ROW LEVEL SECURITY;
ALTER TABLE workspace_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE project_builds ENABLE ROW LEVEL SECURITY;
ALTER TABLE project_templates ENABLE ROW LEVEL SECURITY;

-- Create RLS policies for multi-tenancy
CREATE POLICY workspaces_tenant_isolation ON workspaces
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY workspace_members_tenant_isolation ON workspace_members
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY projects_tenant_isolation ON projects
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY project_builds_tenant_isolation ON project_builds
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY project_templates_tenant_isolation ON project_templates
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

-- Add table comments
COMMENT ON TABLE workspaces IS 'Team workspaces containing multiple projects';
COMMENT ON TABLE workspace_members IS 'Team members and their roles in workspaces';
COMMENT ON TABLE projects IS 'Individual projects within workspaces';
COMMENT ON TABLE project_builds IS 'Build history and deployment tracking';
COMMENT ON TABLE project_templates IS 'Reusable project templates and starters';

-- Add column comments
COMMENT ON COLUMN workspaces.slug IS 'URL-friendly workspace identifier';
COMMENT ON COLUMN workspaces.settings IS 'Workspace configuration and preferences';
COMMENT ON COLUMN projects.environment_vars IS 'Environment variables for deployments';
COMMENT ON COLUMN project_builds.artifacts IS 'Build output files and metadata';
COMMENT ON COLUMN project_templates.template_files IS 'Template file structure and content';