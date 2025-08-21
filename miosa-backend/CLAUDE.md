# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

MIOSA (Multi-Agent Intelligent Operating System Architecture) is a production-ready backend platform implementing a self-improving multi-agent system with intelligent LLM routing.

## Development Commands

### Building and Running
```bash
# Install dependencies
go mod download
go mod tidy

# Run specific service
go run cmd/api-gateway/main.go
go run cmd/agent-orchestrator/main.go
go run cmd/[service-name]/main.go

# Run all tests
go test ./...

# Run specific test file
go test ./tests -v -run TestNodeBasedRouter
go test ./tests -v -run TestOrchestrationFlow

# Run tests with coverage
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Database Operations
```bash
# Run migrations (uses golang-migrate)
migrate -database "${DATABASE_URL}" -path internal/db/migrations up
migrate -database "${DATABASE_URL}" -path internal/db/migrations down 1

# Generate SQLC code from queries
sqlc generate -f internal/db/sqlc.yaml
```

### Code Quality
```bash
# Format code
go fmt ./...
gofmt -w -s .

# Lint (install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
golangci-lint run ./...

# Vet code
go vet ./...
```

## Architecture

### Multi-Agent System Design
The platform implements an 11-agent architecture with Kimi K2 as the primary orchestrator:

1. **Agent Registry** (`internal/agents/registry.go`) - Central registration and confidence tracking
2. **Orchestrator** (`internal/agents/orchestrator.go`) - Kimi K2 powered, handles workflow routing
3. **Communication Flow**: Agents communicate via Task/Result interfaces with confidence scoring (0-10 scale)

### LLM Router Architecture
The node-based router (`internal/llm/router.go`) implements self-improvement:
- **Scoring Nodes**: TaskFit, Quality, Speed, Cost, History, Confidence
- **Self-Improvement**: Models with >80% success rate get 10% bonus
- **Priority Modes**: Speed, Quality, Cost, Balance affect weight distribution
- **Performance**: 2.1μs per routing decision

### Database Schema Pattern
Migrations follow numbered pattern in `internal/db/migrations/`:
- `XXX_name.up.sql` / `XXX_name.down.sql`
- Multi-tenancy via `tenant_id` with Row Level Security
- Vector search using pgvector with HNSW indexes
- Materialized views for performance-critical queries

### Service Communication
Services in `cmd/` directory communicate via:
1. **Synchronous**: HTTP/gRPC for real-time operations
2. **Asynchronous**: Event-driven via agent executions table
3. **Caching**: Redis for session and temporary data

## Key Implementation Patterns

### Agent Implementation
When creating new agents:
1. Implement the `Agent` interface in `internal/agents/interfaces.go`
2. Register in `internal/agents/registry.go` 
3. Add to `AgentType` enum
4. Implement confidence scoring in Execute method

### LLM Provider Integration
New providers must:
1. Implement `Provider` interface in `internal/llm/`
2. Add to router's catalog with task fit scores
3. Implement retry logic and error handling
4. Support both streaming and non-streaming modes

### Database Migrations
1. Create numbered migration files in `internal/db/migrations/`
2. Include tenant_id for multi-tenancy tables
3. Add RLS policies for security
4. Create indexes for foreign keys and frequent queries

## Environment Configuration

Required environment variables:
```bash
# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/miosa?sslmode=disable
REDIS_URL=redis://localhost:6379

# LLM Providers
GROQ_API_KEY=your_groq_api_key
KIMI_API_KEY=optional_for_direct_moonshot_api

# Services
JWT_SECRET=your_jwt_secret
E2B_API_KEY=for_code_execution
RENDER_API_KEY=for_deployments
STRIPE_SECRET_KEY=for_billing
```

## Testing Strategy

### Test Organization
- `tests/integration_test.go` - End-to-end orchestration flows
- `tests/router_test.go` - LLM router and self-improvement
- `tests/simple_test.go` - API connectivity checks
- Service-specific tests in `internal/services/*/`

### Running Tests
Tests use environment variables for credentials. If `GROQ_API_KEY` is not set, tests will skip gracefully.

## Critical Code Paths

### Orchestration Flow
1. Request enters via `cmd/api-gateway/main.go`
2. Routed to `internal/agents/orchestrator.go`
3. Orchestrator uses `internal/llm/router.go` to select model
4. Task executed by specific agent from registry
5. Result stored with confidence score and pattern analysis
6. Low scores (<7.0) trigger improvement analysis

### Self-Improvement Loop
1. Router tracks success rate in `Stats` struct
2. After 100 requests, evaluates performance
3. Models with >85% success rate marked as "improved"
4. Future selections apply 10% bonus to proven performers

## Performance Considerations

### Database
- Use read replicas for analytics queries
- Materialized views refresh on schedule (not real-time)
- HNSW indexes for vector search (build time vs query speed tradeoff)

### LLM Routing
- Cache model selection for similar tasks
- Use Groq for latency-sensitive subagent tasks
- Reserve Kimi K2 for complex orchestration

## Migration Guide for Major Changes

### Adding New Agent Type
1. Update `AgentType` enum in `interfaces.go`
2. Add to database migration for agents table
3. Update orchestrator's routing logic
4. Add default prompts in `config/defaults.go`

### Changing LLM Provider
1. Update catalog in `router.go`
2. Adjust task fit scores based on benchmarks
3. Update provider initialization in orchestrator
4. Test confidence scoring with new model