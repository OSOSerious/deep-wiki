-- Migration 008: Generated Applications and Visual Editor
-- This migration creates the system for app templates, generated schemas, and visual editing

-- Create app templates table
CREATE TABLE IF NOT EXISTS app_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Template identity
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    subcategory VARCHAR(100),
    
    -- Template classification
    template_type VARCHAR(50) NOT NULL CHECK (template_type IN (
        'web_app', 'mobile_app', 'api', 'dashboard', 'landing_page', 
        'e_commerce', 'blog', 'portfolio', 'saas', 'custom'
    )),
    complexity_level VARCHAR(20) CHECK (complexity_level IN ('beginner', 'intermediate', 'advanced', 'expert')),
    
    -- Technology stack
    frontend_framework VARCHAR(100),
    backend_framework VARCHAR(100),
    database_type VARCHAR(50),
    hosting_platform VARCHAR(50),
    
    -- Template content
    file_structure JSONB NOT NULL DEFAULT '{}',
    source_files JSONB NOT NULL DEFAULT '{}',
    configuration JSONB DEFAULT '{}',
    
    -- Visual design
    preview_images TEXT[],
    thumbnail_url TEXT,
    demo_url TEXT,
    
    -- Features and capabilities
    features JSONB DEFAULT '[]',
    components JSONB DEFAULT '[]',
    pages JSONB DEFAULT '[]',
    
    -- Template settings
    is_public BOOLEAN DEFAULT false,
    is_featured BOOLEAN DEFAULT false,
    is_premium BOOLEAN DEFAULT false,
    price_cents INTEGER DEFAULT 0,
    
    -- Usage and popularity
    usage_count INTEGER DEFAULT 0,
    rating_average DECIMAL(3,2) DEFAULT 0,
    rating_count INTEGER DEFAULT 0,
    
    -- Version control
    version VARCHAR(20) DEFAULT '1.0.0',
    changelog TEXT,
    
    -- Status
    status VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived', 'deprecated')),
    
    -- Dependencies and requirements
    dependencies JSONB DEFAULT '{}',
    min_node_version VARCHAR(20),
    
    -- Metadata
    tags TEXT[],
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create generated schemas table
CREATE TABLE IF NOT EXISTS generated_schemas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Schema identity
    schema_name VARCHAR(255) NOT NULL,
    schema_type VARCHAR(50) NOT NULL CHECK (schema_type IN (
        'database', 'api', 'graphql', 'component', 'state', 'form', 'validation'
    )),
    
    -- Schema definition
    schema_definition JSONB NOT NULL,
    json_schema JSONB,
    typescript_types TEXT,
    
    -- Generation context
    generation_prompt TEXT,
    generated_from VARCHAR(100), -- 'conversation', 'template', 'upload', 'manual'
    source_data JSONB DEFAULT '{}',
    
    -- AI processing
    generated_by VARCHAR(100),
    model_used VARCHAR(100),
    generation_confidence DECIMAL(3,2),
    
    -- Validation and quality
    is_validated BOOLEAN DEFAULT false,
    validation_errors JSONB DEFAULT '[]',
    quality_score DECIMAL(3,2),
    
    -- Usage and relationships
    used_in_components TEXT[],
    depends_on_schemas UUID[],
    dependent_schemas UUID[],
    
    -- Version control
    version INTEGER DEFAULT 1,
    parent_schema_id UUID REFERENCES generated_schemas(id),
    
    -- Status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'deprecated', 'archived')),
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(project_id, schema_name, version)
);

-- Create visual editor components table
CREATE TABLE IF NOT EXISTS visual_editor_components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Component identity
    component_name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    
    -- Component classification
    component_type VARCHAR(50) NOT NULL CHECK (component_type IN (
        'layout', 'form', 'display', 'navigation', 'media', 'data', 'custom'
    )),
    category VARCHAR(100),
    
    -- Component definition
    component_config JSONB NOT NULL DEFAULT '{}',
    props_schema JSONB DEFAULT '{}',
    default_props JSONB DEFAULT '{}',
    
    -- Visual properties
    style_config JSONB DEFAULT '{}',
    responsive_config JSONB DEFAULT '{}',
    animations JSONB DEFAULT '{}',
    
    -- Code generation
    component_code TEXT,
    css_code TEXT,
    jsx_template TEXT,
    
    -- Positioning and layout
    position JSONB DEFAULT '{}',
    dimensions JSONB DEFAULT '{}',
    constraints JSONB DEFAULT '{}',
    
    -- Relationships
    parent_component_id UUID REFERENCES visual_editor_components(id),
    child_components UUID[],
    
    -- State and behavior
    state_config JSONB DEFAULT '{}',
    event_handlers JSONB DEFAULT '{}',
    data_bindings JSONB DEFAULT '{}',
    
    -- Version control
    version INTEGER DEFAULT 1,
    
    -- Status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'hidden', 'locked', 'archived')),
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create app deployments table
CREATE TABLE IF NOT EXISTS app_deployments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    deployed_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Deployment identity
    deployment_name VARCHAR(255) NOT NULL,
    environment VARCHAR(50) DEFAULT 'production' CHECK (environment IN ('development', 'staging', 'production')),
    
    -- Deployment configuration
    build_config JSONB NOT NULL DEFAULT '{}',
    environment_variables JSONB DEFAULT '{}',
    deployment_settings JSONB DEFAULT '{}',
    
    -- Target platform
    platform VARCHAR(50) NOT NULL CHECK (platform IN ('vercel', 'netlify', 'aws', 'azure', 'gcp', 'custom')),
    platform_config JSONB DEFAULT '{}',
    
    -- URLs and access
    deployment_url TEXT NOT NULL,
    custom_domain TEXT,
    preview_url TEXT,
    
    -- Build and deployment process
    build_id UUID,
    build_status VARCHAR(20) DEFAULT 'pending' CHECK (build_status IN ('pending', 'building', 'success', 'failed', 'cancelled')),
    build_logs TEXT,
    
    -- Deployment metrics
    build_time_seconds INTEGER,
    bundle_size_bytes BIGINT,
    deployment_time TIMESTAMPTZ,
    
    -- Performance metrics
    lighthouse_score JSONB DEFAULT '{}',
    performance_metrics JSONB DEFAULT '{}',
    
    -- Health and monitoring
    health_status VARCHAR(20) DEFAULT 'unknown' CHECK (health_status IN ('healthy', 'degraded', 'unhealthy', 'unknown')),
    last_health_check TIMESTAMPTZ,
    uptime_percentage DECIMAL(5,2),
    
    -- Traffic and analytics
    page_views INTEGER DEFAULT 0,
    unique_visitors INTEGER DEFAULT 0,
    
    -- Version and rollback
    version VARCHAR(50),
    git_commit_sha VARCHAR(40),
    rollback_deployment_id UUID REFERENCES app_deployments(id),
    
    -- Status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'archived')),
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create code generation history table
CREATE TABLE IF NOT EXISTS code_generation_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    session_id UUID REFERENCES consultation_sessions(id) ON DELETE SET NULL,
    generated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    
    -- Generation request
    generation_type VARCHAR(50) NOT NULL CHECK (generation_type IN (
        'component', 'page', 'api_endpoint', 'database_model', 'utility', 'test', 'configuration'
    )),
    prompt TEXT NOT NULL,
    context JSONB DEFAULT '{}',
    
    -- Generation process
    agent_used VARCHAR(100),
    model_used VARCHAR(100),
    generation_approach VARCHAR(50),
    
    -- Generated content
    generated_code TEXT NOT NULL,
    file_path VARCHAR(500),
    language VARCHAR(50),
    
    -- Quality metrics
    code_quality_score DECIMAL(3,2),
    complexity_score DECIMAL(3,2),
    readability_score DECIMAL(3,2),
    
    -- User feedback
    user_rating INTEGER CHECK (user_rating >= 1 AND user_rating <= 5),
    user_feedback TEXT,
    was_used BOOLEAN DEFAULT false,
    
    -- Processing metrics
    tokens_used INTEGER,
    generation_time_ms INTEGER,
    
    -- Relationships
    based_on_template_id UUID REFERENCES app_templates(id),
    related_generations UUID[],
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX idx_app_templates_tenant_id ON app_templates(tenant_id);
CREATE INDEX idx_app_templates_category ON app_templates(category, subcategory);
CREATE INDEX idx_app_templates_template_type ON app_templates(template_type);
CREATE INDEX idx_app_templates_status ON app_templates(status);
CREATE INDEX idx_app_templates_is_public ON app_templates(is_public);
CREATE INDEX idx_app_templates_is_featured ON app_templates(is_featured);
CREATE INDEX idx_app_templates_usage_count ON app_templates(usage_count DESC);
CREATE INDEX idx_app_templates_rating ON app_templates(rating_average DESC);

CREATE INDEX idx_generated_schemas_tenant_id ON generated_schemas(tenant_id);
CREATE INDEX idx_generated_schemas_project_id ON generated_schemas(project_id);
CREATE INDEX idx_generated_schemas_schema_type ON generated_schemas(schema_type);
CREATE INDEX idx_generated_schemas_status ON generated_schemas(status);
CREATE INDEX idx_generated_schemas_name_version ON generated_schemas(project_id, schema_name, version);

CREATE INDEX idx_visual_editor_components_tenant_id ON visual_editor_components(tenant_id);
CREATE INDEX idx_visual_editor_components_project_id ON visual_editor_components(project_id);
CREATE INDEX idx_visual_editor_components_component_type ON visual_editor_components(component_type);
CREATE INDEX idx_visual_editor_components_parent ON visual_editor_components(parent_component_id);
CREATE INDEX idx_visual_editor_components_status ON visual_editor_components(status);

CREATE INDEX idx_app_deployments_tenant_id ON app_deployments(tenant_id);
CREATE INDEX idx_app_deployments_project_id ON app_deployments(project_id);
CREATE INDEX idx_app_deployments_environment ON app_deployments(environment);
CREATE INDEX idx_app_deployments_platform ON app_deployments(platform);
CREATE INDEX idx_app_deployments_status ON app_deployments(status);
CREATE INDEX idx_app_deployments_deployment_time ON app_deployments(deployment_time DESC);

CREATE INDEX idx_code_generation_history_tenant_id ON code_generation_history(tenant_id);
CREATE INDEX idx_code_generation_history_project_id ON code_generation_history(project_id);
CREATE INDEX idx_code_generation_history_session_id ON code_generation_history(session_id);
CREATE INDEX idx_code_generation_history_generation_type ON code_generation_history(generation_type);
CREATE INDEX idx_code_generation_history_created_at ON code_generation_history(created_at DESC);

-- Add updated_at triggers
CREATE TRIGGER set_timestamp_app_templates
    BEFORE UPDATE ON app_templates
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_generated_schemas
    BEFORE UPDATE ON generated_schemas
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_visual_editor_components
    BEFORE UPDATE ON visual_editor_components
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_app_deployments
    BEFORE UPDATE ON app_deployments
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Enable Row Level Security
ALTER TABLE app_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE generated_schemas ENABLE ROW LEVEL SECURITY;
ALTER TABLE visual_editor_components ENABLE ROW LEVEL SECURITY;
ALTER TABLE app_deployments ENABLE ROW LEVEL SECURITY;
ALTER TABLE code_generation_history ENABLE ROW LEVEL SECURITY;

-- Create RLS policies for multi-tenancy
CREATE POLICY app_templates_tenant_isolation ON app_templates
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY generated_schemas_tenant_isolation ON generated_schemas
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY visual_editor_components_tenant_isolation ON visual_editor_components
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY app_deployments_tenant_isolation ON app_deployments
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY code_generation_history_tenant_isolation ON code_generation_history
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

-- Add table comments
COMMENT ON TABLE app_templates IS 'Reusable application templates with file structures and configurations';
COMMENT ON TABLE generated_schemas IS 'AI-generated schemas for databases, APIs, and components';
COMMENT ON TABLE visual_editor_components IS 'Visual editor components with drag-and-drop capabilities';
COMMENT ON TABLE app_deployments IS 'Application deployments to various hosting platforms';
COMMENT ON TABLE code_generation_history IS 'History of AI code generation with quality metrics';

-- Add column comments
COMMENT ON COLUMN app_templates.file_structure IS 'Complete file and folder structure of the template';
COMMENT ON COLUMN app_templates.source_files IS 'Template source code files and content';
COMMENT ON COLUMN generated_schemas.schema_definition IS 'Complete schema definition in JSON format';
COMMENT ON COLUMN visual_editor_components.component_config IS 'Visual component configuration and properties';
COMMENT ON COLUMN app_deployments.deployment_url IS 'Primary URL where the application is deployed';