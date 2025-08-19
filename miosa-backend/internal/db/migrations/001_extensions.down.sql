-- Migration 001 Down: Drop PostgreSQL Extensions
-- This migration removes the PostgreSQL extensions

-- Drop btree_gist extension
DROP EXTENSION IF EXISTS btree_gist;

-- Drop btree_gin extension
DROP EXTENSION IF EXISTS btree_gin;

-- Drop pg_trgm extension
DROP EXTENSION IF EXISTS pg_trgm;

-- Drop pgvector extension
DROP EXTENSION IF EXISTS vector;

-- Drop uuid-ossp extension
DROP EXTENSION IF EXISTS "uuid-ossp";