-- Migration 006 Down: Drop Memory and Messages tables

-- Drop RLS policies
DROP POLICY IF EXISTS conversation_memory_tenant_isolation ON conversation_memory;
DROP POLICY IF EXISTS messages_tenant_isolation ON messages;
DROP POLICY IF EXISTS message_relationships_tenant_isolation ON message_relationships;
DROP POLICY IF EXISTS conversation_summaries_tenant_isolation ON conversation_summaries;
DROP POLICY IF EXISTS knowledge_entities_tenant_isolation ON knowledge_entities;

-- Drop triggers
DROP TRIGGER IF EXISTS set_timestamp_conversation_memory ON conversation_memory;
DROP TRIGGER IF EXISTS set_timestamp_conversation_summaries ON conversation_summaries;
DROP TRIGGER IF EXISTS set_timestamp_knowledge_entities ON knowledge_entities;

-- Drop tables in reverse order (due to foreign key dependencies)
DROP TABLE IF EXISTS knowledge_entities;
DROP TABLE IF EXISTS conversation_summaries;
DROP TABLE IF EXISTS message_relationships;

-- Drop message partitions first, then the main table
DROP TABLE IF EXISTS messages_2024_01;
DROP TABLE IF EXISTS messages_2024_02;
DROP TABLE IF EXISTS messages_2024_03;
DROP TABLE IF EXISTS messages_2024_04;
DROP TABLE IF EXISTS messages_2024_05;
DROP TABLE IF EXISTS messages_2024_06;
DROP TABLE IF EXISTS messages_2024_07;
DROP TABLE IF EXISTS messages_2024_08;
DROP TABLE IF EXISTS messages_2024_09;
DROP TABLE IF EXISTS messages_2024_10;
DROP TABLE IF EXISTS messages_2024_11;
DROP TABLE IF EXISTS messages_2024_12;
DROP TABLE IF EXISTS messages;

DROP TABLE IF EXISTS conversation_memory;