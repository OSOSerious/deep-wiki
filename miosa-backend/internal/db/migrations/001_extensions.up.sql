-- Migration 001: PostgreSQL Extensions
-- This migration sets up the required PostgreSQL extensions for MIOSA

-- Enable UUID generation extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable pgvector extension for vector similarity search
-- Used for embedding storage and semantic search
CREATE EXTENSION IF NOT EXISTS vector;

-- Enable pg_trgm extension for full-text search and fuzzy matching
-- Used for code search and text similarity
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Enable btree_gin extension for better indexing performance
CREATE EXTENSION IF NOT EXISTS btree_gin;

-- Enable btree_gist extension for exclusion constraints
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Comment on the schema
COMMENT ON SCHEMA public IS 'MIOSA main schema with vector search and UUID support';