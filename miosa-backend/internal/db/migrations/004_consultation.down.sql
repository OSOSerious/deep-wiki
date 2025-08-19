-- Migration 004 Down: Drop Consultation system tables

-- Drop RLS policies
DROP POLICY IF EXISTS consultation_sessions_tenant_isolation ON consultation_sessions;
DROP POLICY IF EXISTS consultation_messages_tenant_isolation ON consultation_messages;
DROP POLICY IF EXISTS consultation_insights_tenant_isolation ON consultation_insights;
DROP POLICY IF EXISTS consultation_analytics_tenant_isolation ON consultation_analytics;

-- Drop triggers
DROP TRIGGER IF EXISTS set_timestamp_consultation_sessions ON consultation_sessions;
DROP TRIGGER IF EXISTS set_timestamp_consultation_insights ON consultation_insights;

-- Drop tables in reverse order (due to foreign key dependencies)
DROP TABLE IF EXISTS consultation_analytics;
DROP TABLE IF EXISTS consultation_insights;
DROP TABLE IF EXISTS consultation_messages;
DROP TABLE IF EXISTS consultation_sessions;