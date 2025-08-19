-- Migration 003 Down: Drop Workspaces and Projects tables

-- Drop RLS policies
DROP POLICY IF EXISTS workspaces_tenant_isolation ON workspaces;
DROP POLICY IF EXISTS workspace_members_tenant_isolation ON workspace_members;
DROP POLICY IF EXISTS projects_tenant_isolation ON projects;
DROP POLICY IF EXISTS project_builds_tenant_isolation ON project_builds;
DROP POLICY IF EXISTS project_templates_tenant_isolation ON project_templates;

-- Drop triggers
DROP TRIGGER IF EXISTS set_timestamp_workspaces ON workspaces;
DROP TRIGGER IF EXISTS set_timestamp_workspace_members ON workspace_members;
DROP TRIGGER IF EXISTS set_timestamp_projects ON projects;
DROP TRIGGER IF EXISTS set_timestamp_project_templates ON project_templates;

-- Drop tables in reverse order (due to foreign key dependencies)
DROP TABLE IF EXISTS project_templates;
DROP TABLE IF EXISTS project_builds;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS workspace_members;
DROP TABLE IF EXISTS workspaces;