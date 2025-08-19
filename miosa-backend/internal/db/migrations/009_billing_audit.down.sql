-- Migration 009 Down: Drop Billing and Audit tables

-- Drop RLS policies
DROP POLICY IF EXISTS subscriptions_tenant_isolation ON subscriptions;
DROP POLICY IF EXISTS usage_metrics_tenant_isolation ON usage_metrics;
DROP POLICY IF EXISTS invoices_tenant_isolation ON invoices;
DROP POLICY IF EXISTS audit_logs_tenant_isolation ON audit_logs;
DROP POLICY IF EXISTS api_keys_tenant_isolation ON api_keys;
DROP POLICY IF EXISTS api_usage_logs_tenant_isolation ON api_usage_logs;

-- Drop triggers
DROP TRIGGER IF EXISTS set_timestamp_subscriptions ON subscriptions;
DROP TRIGGER IF EXISTS set_timestamp_invoices ON invoices;
DROP TRIGGER IF EXISTS set_timestamp_api_keys ON api_keys;

-- Drop API usage logs partitions first, then the main table
DROP TABLE IF EXISTS api_usage_logs_2024_01;
DROP TABLE IF EXISTS api_usage_logs_2024_02;
DROP TABLE IF EXISTS api_usage_logs_2024_03;
DROP TABLE IF EXISTS api_usage_logs_2024_04;
DROP TABLE IF EXISTS api_usage_logs_2024_05;
DROP TABLE IF EXISTS api_usage_logs_2024_06;
DROP TABLE IF EXISTS api_usage_logs_2024_07;
DROP TABLE IF EXISTS api_usage_logs_2024_08;
DROP TABLE IF EXISTS api_usage_logs_2024_09;
DROP TABLE IF EXISTS api_usage_logs_2024_10;
DROP TABLE IF EXISTS api_usage_logs_2024_11;
DROP TABLE IF EXISTS api_usage_logs_2024_12;
DROP TABLE IF EXISTS api_usage_logs;

-- Drop audit logs partitions first, then the main table
DROP TABLE IF EXISTS audit_logs_2024_01;
DROP TABLE IF EXISTS audit_logs_2024_02;
DROP TABLE IF EXISTS audit_logs_2024_03;
DROP TABLE IF EXISTS audit_logs_2024_04;
DROP TABLE IF EXISTS audit_logs_2024_05;
DROP TABLE IF EXISTS audit_logs_2024_06;
DROP TABLE IF EXISTS audit_logs_2024_07;
DROP TABLE IF EXISTS audit_logs_2024_08;
DROP TABLE IF EXISTS audit_logs_2024_09;
DROP TABLE IF EXISTS audit_logs_2024_10;
DROP TABLE IF EXISTS audit_logs_2024_11;
DROP TABLE IF EXISTS audit_logs_2024_12;
DROP TABLE IF EXISTS audit_logs;

-- Drop usage metrics partitions first, then the main table
DROP TABLE IF EXISTS usage_metrics_2024_01;
DROP TABLE IF EXISTS usage_metrics_2024_02;
DROP TABLE IF EXISTS usage_metrics_2024_03;
DROP TABLE IF EXISTS usage_metrics_2024_04;
DROP TABLE IF EXISTS usage_metrics_2024_05;
DROP TABLE IF EXISTS usage_metrics_2024_06;
DROP TABLE IF EXISTS usage_metrics_2024_07;
DROP TABLE IF EXISTS usage_metrics_2024_08;
DROP TABLE IF EXISTS usage_metrics_2024_09;
DROP TABLE IF EXISTS usage_metrics_2024_10;
DROP TABLE IF EXISTS usage_metrics_2024_11;
DROP TABLE IF EXISTS usage_metrics_2024_12;
DROP TABLE IF EXISTS usage_metrics;

-- Drop tables in reverse order (due to foreign key dependencies)
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS subscriptions;