-- Migration 007 Down: Drop Intelligence and Analytics tables

-- Drop RLS policies
DROP POLICY IF EXISTS business_insights_tenant_isolation ON business_insights;
DROP POLICY IF EXISTS predictive_analytics_tenant_isolation ON predictive_analytics;
DROP POLICY IF EXISTS email_intelligence_tenant_isolation ON email_intelligence;
DROP POLICY IF EXISTS meeting_intelligence_tenant_isolation ON meeting_intelligence;
DROP POLICY IF EXISTS analytics_dashboards_tenant_isolation ON analytics_dashboards;
DROP POLICY IF EXISTS usage_analytics_tenant_isolation ON usage_analytics;

-- Drop triggers
DROP TRIGGER IF EXISTS set_timestamp_business_insights ON business_insights;
DROP TRIGGER IF EXISTS set_timestamp_predictive_analytics ON predictive_analytics;
DROP TRIGGER IF EXISTS set_timestamp_email_intelligence ON email_intelligence;
DROP TRIGGER IF EXISTS set_timestamp_meeting_intelligence ON meeting_intelligence;
DROP TRIGGER IF EXISTS set_timestamp_analytics_dashboards ON analytics_dashboards;

-- Drop usage analytics partitions first, then the main table
DROP TABLE IF EXISTS usage_analytics_2024_01;
DROP TABLE IF EXISTS usage_analytics_2024_02;
DROP TABLE IF EXISTS usage_analytics_2024_03;
DROP TABLE IF EXISTS usage_analytics_2024_04;
DROP TABLE IF EXISTS usage_analytics_2024_05;
DROP TABLE IF EXISTS usage_analytics_2024_06;
DROP TABLE IF EXISTS usage_analytics_2024_07;
DROP TABLE IF EXISTS usage_analytics_2024_08;
DROP TABLE IF EXISTS usage_analytics_2024_09;
DROP TABLE IF EXISTS usage_analytics_2024_10;
DROP TABLE IF EXISTS usage_analytics_2024_11;
DROP TABLE IF EXISTS usage_analytics_2024_12;
DROP TABLE IF EXISTS usage_analytics;

-- Drop tables in reverse order (due to foreign key dependencies)
DROP TABLE IF EXISTS analytics_dashboards;
DROP TABLE IF EXISTS meeting_intelligence;
DROP TABLE IF EXISTS email_intelligence;
DROP TABLE IF EXISTS predictive_analytics;
DROP TABLE IF EXISTS business_insights;