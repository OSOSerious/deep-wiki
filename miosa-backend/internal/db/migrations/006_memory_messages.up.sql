-- Migration 006: Conversation Memory with Vectors and Partitioned Messages
-- This migration creates the memory system with vector embeddings and message relationships

-- Create conversation memory table
CREATE TABLE IF NOT EXISTS conversation_memory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    session_id UUID NOT NULL REFERENCES consultation_sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Memory classification
    memory_type VARCHAR(50) NOT NULL CHECK (memory_type IN (
        'user_preference', 'project_context', 'conversation_history', 
        'code_pattern', 'decision_rationale', 'learned_behavior'
    )),
    category VARCHAR(100),
    
    -- Memory content
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    summary TEXT,
    
    -- Context and metadata
    context JSONB DEFAULT '{}',
    entities JSONB DEFAULT '[]',
    topics JSONB DEFAULT '[]',
    
    -- Vector embeddings for semantic search
    content_embedding vector(1536),
    summary_embedding vector(1536),
    
    -- Importance and relevance
    importance_score DECIMAL(3,2) DEFAULT 0.5 CHECK (importance_score >= 0 AND importance_score <= 1),
    access_count INTEGER DEFAULT 0,
    last_accessed_at TIMESTAMPTZ,
    
    -- Relationships
    related_memories UUID[],
    source_message_ids UUID[],
    
    -- Lifecycle
    expires_at TIMESTAMPTZ,
    is_archived BOOLEAN DEFAULT false,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create partitioned messages table for high-volume message storage
CREATE TABLE IF NOT EXISTS messages (
    id UUID DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    session_id UUID NOT NULL,
    conversation_id UUID, -- For grouping related conversations
    thread_id UUID, -- For message threading
    
    -- Message source and target
    sender_type VARCHAR(20) NOT NULL CHECK (sender_type IN ('user', 'agent', 'system')),
    sender_id VARCHAR(255),
    sender_name VARCHAR(255),
    recipient_type VARCHAR(20) CHECK (recipient_type IN ('user', 'agent', 'system', 'broadcast')),
    recipient_id VARCHAR(255),
    
    -- Message content
    message_type VARCHAR(50) DEFAULT 'text' CHECK (message_type IN (
        'text', 'code', 'file', 'image', 'command', 'result', 
        'error', 'suggestion', 'question', 'answer'
    )),
    content TEXT NOT NULL,
    raw_content TEXT, -- Original unprocessed content
    formatted_content JSONB, -- Rich formatted content
    
    -- Message context
    context JSONB DEFAULT '{}',
    intent VARCHAR(100), -- Detected intent
    sentiment VARCHAR(20), -- positive, negative, neutral
    
    -- Attachments and references
    attachments JSONB DEFAULT '[]',
    file_references TEXT[],
    code_snippets JSONB DEFAULT '[]',
    
    -- AI processing
    embedding vector(1536),
    tokens_count INTEGER,
    processing_time_ms INTEGER,
    model_used VARCHAR(100),
    
    -- Message relationships
    parent_message_id UUID,
    reply_to_message_id UUID,
    related_message_ids UUID[],
    
    -- Message status and lifecycle
    status VARCHAR(20) DEFAULT 'sent' CHECK (status IN ('draft', 'sending', 'sent', 'delivered', 'read', 'error')),
    is_internal BOOLEAN DEFAULT false,
    is_system_generated BOOLEAN DEFAULT false,
    requires_response BOOLEAN DEFAULT false,
    
    -- User interaction
    reactions JSONB DEFAULT '{}',
    bookmarked BOOLEAN DEFAULT false,
    user_rating INTEGER CHECK (user_rating >= 1 AND user_rating <= 5),
    
    -- Sequence and ordering
    sequence_number SERIAL,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create monthly partitions for messages (example for current year)
CREATE TABLE messages_2024_01 PARTITION OF messages
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE messages_2024_02 PARTITION OF messages
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
CREATE TABLE messages_2024_03 PARTITION OF messages
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');
CREATE TABLE messages_2024_04 PARTITION OF messages
    FOR VALUES FROM ('2024-04-01') TO ('2024-05-01');
CREATE TABLE messages_2024_05 PARTITION OF messages
    FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');
CREATE TABLE messages_2024_06 PARTITION OF messages
    FOR VALUES FROM ('2024-06-01') TO ('2024-07-01');
CREATE TABLE messages_2024_07 PARTITION OF messages
    FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');
CREATE TABLE messages_2024_08 PARTITION OF messages
    FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');
CREATE TABLE messages_2024_09 PARTITION OF messages
    FOR VALUES FROM ('2024-09-01') TO ('2024-10-01');
CREATE TABLE messages_2024_10 PARTITION OF messages
    FOR VALUES FROM ('2024-10-01') TO ('2024-11-01');
CREATE TABLE messages_2024_11 PARTITION OF messages
    FOR VALUES FROM ('2024-11-01') TO ('2024-12-01');
CREATE TABLE messages_2024_12 PARTITION OF messages
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

-- Create message relationships table for complex message connections
CREATE TABLE IF NOT EXISTS message_relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    -- Relationship details
    source_message_id UUID NOT NULL,
    target_message_id UUID NOT NULL,
    relationship_type VARCHAR(50) NOT NULL CHECK (relationship_type IN (
        'reply', 'reference', 'follow_up', 'clarification', 
        'continuation', 'correction', 'elaboration'
    )),
    
    -- Relationship strength and context
    strength DECIMAL(3,2) DEFAULT 1.0 CHECK (strength >= 0 AND strength <= 1),
    context JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(source_message_id, target_message_id, relationship_type)
);

-- Create conversation summaries table for long conversation contexts
CREATE TABLE IF NOT EXISTS conversation_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    session_id UUID NOT NULL REFERENCES consultation_sessions(id) ON DELETE CASCADE,
    conversation_id UUID,
    
    -- Summary details
    summary_type VARCHAR(50) NOT NULL CHECK (summary_type IN (
        'periodic', 'topic_change', 'phase_completion', 'session_end'
    )),
    title VARCHAR(255),
    content TEXT NOT NULL,
    
    -- Context and scope
    message_count INTEGER NOT NULL,
    start_message_id UUID,
    end_message_id UUID,
    time_range_start TIMESTAMPTZ NOT NULL,
    time_range_end TIMESTAMPTZ NOT NULL,
    
    -- Key information
    key_topics JSONB DEFAULT '[]',
    important_decisions JSONB DEFAULT '[]',
    action_items JSONB DEFAULT '[]',
    
    -- AI generation details
    generated_by VARCHAR(100),
    embedding vector(1536),
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create knowledge graph entities table for semantic relationships
CREATE TABLE IF NOT EXISTS knowledge_entities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    -- Entity details
    entity_type VARCHAR(50) NOT NULL CHECK (entity_type IN (
        'person', 'project', 'technology', 'concept', 'file', 'function', 'class'
    )),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Entity properties
    properties JSONB DEFAULT '{}',
    aliases TEXT[],
    
    -- Vector representation
    embedding vector(1536),
    
    -- Usage tracking
    mention_count INTEGER DEFAULT 1,
    first_mentioned_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    last_mentioned_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    -- Relationships
    related_entities UUID[],
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(tenant_id, entity_type, name)
);

-- Create indexes for performance
CREATE INDEX idx_conversation_memory_tenant_id ON conversation_memory(tenant_id);
CREATE INDEX idx_conversation_memory_session_id ON conversation_memory(session_id);
CREATE INDEX idx_conversation_memory_user_id ON conversation_memory(user_id);
CREATE INDEX idx_conversation_memory_memory_type ON conversation_memory(memory_type);
CREATE INDEX idx_conversation_memory_importance ON conversation_memory(importance_score DESC);
CREATE INDEX idx_conversation_memory_created_at ON conversation_memory(created_at DESC);

-- Indexes for partitioned messages table
CREATE INDEX idx_messages_tenant_id ON messages(tenant_id, created_at);
CREATE INDEX idx_messages_session_id ON messages(session_id, created_at);
CREATE INDEX idx_messages_sender_type ON messages(sender_type, created_at);
CREATE INDEX idx_messages_message_type ON messages(message_type, created_at);
CREATE INDEX idx_messages_sequence ON messages(session_id, sequence_number);

CREATE INDEX idx_message_relationships_tenant_id ON message_relationships(tenant_id);
CREATE INDEX idx_message_relationships_source ON message_relationships(source_message_id);
CREATE INDEX idx_message_relationships_target ON message_relationships(target_message_id);
CREATE INDEX idx_message_relationships_type ON message_relationships(relationship_type);

CREATE INDEX idx_conversation_summaries_tenant_id ON conversation_summaries(tenant_id);
CREATE INDEX idx_conversation_summaries_session_id ON conversation_summaries(session_id);
CREATE INDEX idx_conversation_summaries_conversation_id ON conversation_summaries(conversation_id);
CREATE INDEX idx_conversation_summaries_time_range ON conversation_summaries(time_range_start, time_range_end);

CREATE INDEX idx_knowledge_entities_tenant_id ON knowledge_entities(tenant_id);
CREATE INDEX idx_knowledge_entities_entity_type ON knowledge_entities(entity_type);
CREATE INDEX idx_knowledge_entities_name ON knowledge_entities(name);
CREATE INDEX idx_knowledge_entities_mention_count ON knowledge_entities(mention_count DESC);

-- Vector similarity search indexes using HNSW
CREATE INDEX idx_conversation_memory_content_embedding ON conversation_memory
    USING hnsw (content_embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

CREATE INDEX idx_conversation_memory_summary_embedding ON conversation_memory
    USING hnsw (summary_embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

CREATE INDEX idx_messages_embedding ON messages
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

CREATE INDEX idx_conversation_summaries_embedding ON conversation_summaries
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

CREATE INDEX idx_knowledge_entities_embedding ON knowledge_entities
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- Add updated_at triggers
CREATE TRIGGER set_timestamp_conversation_memory
    BEFORE UPDATE ON conversation_memory
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_conversation_summaries
    BEFORE UPDATE ON conversation_summaries
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_knowledge_entities
    BEFORE UPDATE ON knowledge_entities
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Enable Row Level Security
ALTER TABLE conversation_memory ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE message_relationships ENABLE ROW LEVEL SECURITY;
ALTER TABLE conversation_summaries ENABLE ROW LEVEL SECURITY;
ALTER TABLE knowledge_entities ENABLE ROW LEVEL SECURITY;

-- Create RLS policies for multi-tenancy
CREATE POLICY conversation_memory_tenant_isolation ON conversation_memory
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY messages_tenant_isolation ON messages
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY message_relationships_tenant_isolation ON message_relationships
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY conversation_summaries_tenant_isolation ON conversation_summaries
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY knowledge_entities_tenant_isolation ON knowledge_entities
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

-- Add table comments
COMMENT ON TABLE conversation_memory IS 'Persistent conversation memory with vector embeddings';
COMMENT ON TABLE messages IS 'Partitioned high-volume message storage with rich metadata';
COMMENT ON TABLE message_relationships IS 'Complex relationships between messages';
COMMENT ON TABLE conversation_summaries IS 'Periodic summaries of conversation segments';
COMMENT ON TABLE knowledge_entities IS 'Knowledge graph entities extracted from conversations';

-- Add column comments
COMMENT ON COLUMN conversation_memory.content_embedding IS 'Vector embedding of memory content (1536 dimensions)';
COMMENT ON COLUMN conversation_memory.importance_score IS 'Memory importance for retention and retrieval';
COMMENT ON COLUMN messages.embedding IS 'Message embedding for semantic search (1536 dimensions)';
COMMENT ON COLUMN messages.formatted_content IS 'Rich formatted content with markdown, code highlighting, etc.';
COMMENT ON COLUMN knowledge_entities.embedding IS 'Entity embedding for semantic matching (1536 dimensions)';