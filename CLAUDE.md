# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

MIOSA (Multi-Agent Intelligent Operating System Architecture) is a full-stack application with:
- **Backend**: Go-based microservices architecture
- **Frontend**: SvelteKit web application
- **Database**: PostgreSQL with Redis caching
- **AI Integration**: Multi-agent system using Groq/Kimi models

## Development Commands

### Frontend (miosa-web)
```bash
cd miosa-web
npm install          # Install dependencies
npm run dev          # Start development server
npm run build        # Build for production
npm run preview      # Preview production build
```

### Backend (miosa-backend)
```bash
cd miosa-backend
go mod download      # Install Go dependencies
go run cmd/api-gateway/main.go  # Run API gateway service
go run cmd/[service-name]/main.go  # Run specific service
go test ./...        # Run all tests
```

## Architecture

### Multi-Agent System
The backend implements a sophisticated multi-agent architecture located in `miosa-backend/internal/agents/`:
- **Orchestrator**: Routes tasks to specialized agents
- **Specialized Agents**: Analysis, Architecture, Development, Quality, Deployment, etc.
- **Tools Registry**: Shared tools for agents in `internal/tools/`

### Service Architecture
Each microservice has its own entry point in `miosa-backend/cmd/`:
- `api-gateway`: Main HTTP API gateway
- `agent-orchestrator`: Manages agent coordination
- `analytics-service`: Metrics and analytics
- `auth-service`: Authentication and authorization
- `billing-service`: Stripe integration
- `workspace-service`: Workspace management
- Additional services for code generation, collaboration, consultation, deployment, execution, IDE, and monitoring

### Database Structure
- PostgreSQL migrations in `miosa-backend/internal/db/migrations/`
- Migration files follow pattern: `XXX_description.up.sql` and `XXX_description.down.sql`
- Core tables: users, workspaces, projects, consultations, agents, billing
- Vector search capabilities for intelligent retrieval

### Frontend Routes
SvelteKit application with phase-based workflow:
- `/` - Landing page
- `/onboarding` - User onboarding
- `/analyze` - Analysis phase
- `/chat` - Consultation phase
- `/build` - Building phase
- `/expand` - Expansion phase
- `/optimize` - Optimization phase
- `/deploy` - Deployment phase
- `/workspace` - Workspace management
- `/ide` - Integrated development environment

## Key Patterns

### Agent Communication
- Agents use structured request/response patterns
- Tool registration system for shared capabilities
- Context passing between agents for workflow continuity

### API Proxy Pattern
Frontend proxies backend API calls through `/api/proxy/[...path]` route

### State Management
Frontend uses Svelte stores (`src/lib/stores/`) for:
- User authentication state
- Workspace context
- Agent interactions
- Theme preferences
- Error handling

### Service Integration
- Groq/Kimi for LLM capabilities
- E2B for execution environments
- Render for deployment
- Stripe for billing

## Environment Variables

Backend expects these in `.env`:
- `GROQ_API_KEY`
- `DATABASE_URL` (PostgreSQL)
- `REDIS_URL`
- `JWT_SECRET`
- `E2B_API_KEY`
- `RENDER_API_KEY`

## Testing

Backend tests are in `miosa-backend/tests/`:
- `agents_test.go` - Agent functionality
- `db_test.go` - Database operations
- `e2b_test.go` - E2B integration
- `mcp_test.go` - MCP protocol
- `services_test.go` - Service layer

Frontend tests in `miosa-web/tests/`:
- Component tests in `components/`
- Route tests in `routes/`
- Utility tests in `utils/`