-- Migration 002 Down: Drop Core Users and Authentication tables

-- Drop RLS policies
DROP POLICY IF EXISTS users_tenant_isolation ON users;
DROP POLICY IF EXISTS user_sessions_tenant_isolation ON user_sessions;
DROP POLICY IF EXISTS password_reset_tokens_tenant_isolation ON password_reset_tokens;
DROP POLICY IF EXISTS email_verification_tokens_tenant_isolation ON email_verification_tokens;

-- Drop triggers
DROP TRIGGER IF EXISTS set_timestamp_users ON users;

-- Drop the timestamp function
DROP FUNCTION IF EXISTS trigger_set_timestamp();

-- Drop tables in reverse order (due to foreign key dependencies)
DROP TABLE IF EXISTS email_verification_tokens;
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS users;