-- Migration 008 Down: Drop Generated Apps and Visual Editor tables

-- Drop RLS policies
DROP POLICY IF EXISTS app_templates_tenant_isolation ON app_templates;
DROP POLICY IF EXISTS generated_schemas_tenant_isolation ON generated_schemas;
DROP POLICY IF EXISTS visual_editor_components_tenant_isolation ON visual_editor_components;
DROP POLICY IF EXISTS app_deployments_tenant_isolation ON app_deployments;
DROP POLICY IF EXISTS code_generation_history_tenant_isolation ON code_generation_history;

-- Drop triggers
DROP TRIGGER IF EXISTS set_timestamp_app_templates ON app_templates;
DROP TRIGGER IF EXISTS set_timestamp_generated_schemas ON generated_schemas;
DROP TRIGGER IF EXISTS set_timestamp_visual_editor_components ON visual_editor_components;
DROP TRIGGER IF EXISTS set_timestamp_app_deployments ON app_deployments;

-- Drop tables in reverse order (due to foreign key dependencies)
DROP TABLE IF EXISTS code_generation_history;
DROP TABLE IF EXISTS app_deployments;
DROP TABLE IF EXISTS visual_editor_components;
DROP TABLE IF EXISTS generated_schemas;
DROP TABLE IF EXISTS app_templates;