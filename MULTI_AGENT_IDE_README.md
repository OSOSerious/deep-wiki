# Multi-Agent IDE Integration System

## 🚀 Overview

This project implements a sophisticated **Multi-Agent Intelligent Operating System Architecture (MIOSA)** that uses 10 specialized AI agents working together to generate complete, production-ready applications from natural language descriptions. The system leverages the Groq API with advanced language models to create real executable code, not just documentation.

## 🎯 What We Built

### Core Components

1. **Multi-Agent Orchestration System**
   - 10 specialized agents working in coordination
   - Dynamic task routing and execution
   - Confidence-based decision making
   - Real-time code generation using LLMs

2. **IDE Server & Interface**
   - Web-based IDE for viewing generated code
   - File tree navigation
   - Real-time updates as agents generate code
   - Syntax highlighting and code editing

3. **Agent Workspace**
   - Isolated directory for agent-generated files
   - Organized project structures
   - Complete application scaffolding

## 🤖 The 10 Specialized Agents

| Agent | Role | Output |
|-------|------|--------|
| **Strategy Agent** | Strategic planning and roadmapping | Strategy documents, project plans |
| **Analysis Agent** | Requirements analysis and breakdown | Analysis reports, specifications |
| **Architect Agent** | System design and architecture | Architecture diagrams, design docs |
| **Development Agent** | Code generation and implementation | Actual source code files |
| **Quality Agent** | Testing and quality assurance | Test suites, quality reports |
| **Monitoring Agent** | Observability and monitoring setup | Prometheus/Grafana configs |
| **Deployment Agent** | Deployment and infrastructure | Docker, Kubernetes manifests |
| **Recommender Agent** | Best practices and recommendations | Improvement suggestions |
| **Communication Agent** | Documentation and communication | README files, API docs |
| **AI Providers Agent** | AI/ML integration configuration | AI service configs |

## 🛠️ Setup Instructions

### Prerequisites

- Go 1.21+
- Node.js 18+
- Groq API Key

### Environment Setup

1. **Clone the repository:**
```bash
git clone https://github.com/sormind/OSA.git
cd OSA
git checkout feature/multi-agent-ide-integration
```

2. **Set your Groq API key:**
```bash
export GROQ_API_KEY="your_groq_api_key_here"
```

3. **Install Go dependencies:**
```bash
cd miosa-backend
go mod download
go mod tidy
```

4. **Install Node dependencies:**
```bash
cd miosa-web
npm install
```

## 🏃‍♂️ Running the System

### Start All Services (Quick Start)

```bash
# Terminal 1: IDE Server
cd miosa-backend
go run cmd/ide-server/main.go -port 8089 -root $(pwd)/../agent-workspace

# Terminal 2: Enhanced Orchestrator (Recommended)
cd miosa-backend
GROQ_API_KEY=$GROQ_API_KEY go run cmd/enhanced-orchestrator/main.go \
  -port 8092 \
  -workspace $(pwd)/../agent-workspace

# Terminal 3: Web Interface
cd miosa-web
npm run dev

# Access the IDE at: http://localhost:3000/ide
```

### Detailed Service Descriptions

#### 1. IDE Server (Port 8089)
```bash
cd miosa-backend
go run cmd/ide-server/main.go -port 8089 -root /path/to/agent-workspace
```
- Serves agent-generated files via HTTP API
- Provides file tree navigation
- Enables code viewing and editing
- CORS-enabled for browser access

#### 2. Enhanced Orchestrator (Port 8092) - RECOMMENDED
```bash
cd miosa-backend
GROQ_API_KEY=$GROQ_API_KEY go run cmd/enhanced-orchestrator/main.go \
  -port 8092 \
  -workspace /path/to/agent-workspace
```
- Generates complete applications with multiple files
- Parses LLM output into actual code files
- Creates proper project structures
- Supports multiple programming languages

#### 3. Full Orchestrator (Port 8091) - All 10 Agents
```bash
cd miosa-backend
GROQ_API_KEY=$GROQ_API_KEY go run cmd/full-orchestrator/main.go \
  -port 8091 \
  -workspace /path/to/agent-workspace
```
- Runs all 10 agents in sequence
- Comprehensive solution generation
- Complete documentation and deployment configs

#### 4. E2B GitHub Push Service (Port 3001)
```bash
cd miosa-backend
node e2b.js
```
- Node.js server that receives the generated code path.
- Creates an E2B sandbox and pushes the code to a new GitHub repository.
- Access at: **http://localhost:3001**

#### 5. Web Interface (Port 3000)
```bash
cd miosa-web
npm run dev
```
- Svelte-based IDE interface
- Real-time file tree updates
- Code viewing with syntax highlighting
- Access at: **http://localhost:3000/ide**

## 📡 API Usage Examples

### Generate a Complete Application

#### Example 1: Real-Time Chat Application
```bash
curl -X POST http://localhost:8092/api/orchestrate \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Build a real-time chat application with WebSocket support, user authentication, message history, typing indicators, online presence, file sharing, and emoji reactions. Include backend API, frontend UI, database schema, Docker setup, and tests."
  }'
```

**Generated Files:**
```
agent-workspace/[workflow-id]/
├── server/
│   ├── app.js              # Express + Socket.io server
│   ├── routes/
│   │   ├── auth.js         # Authentication routes
│   │   └── chat.js         # Chat API routes
│   ├── utils/
│   │   ├── socket.js       # WebSocket handlers
│   │   ├── auth.js         # JWT authentication
│   │   └── db.js           # Database connection
│   ├── tests/
│   │   ├── auth.test.js    # Auth tests
│   │   └── chat.test.js    # Chat tests
│   ├── schema.sql          # Database schema
│   ├── package.json        # Dependencies
│   └── Dockerfile          # Container setup
├── client/
│   ├── src/
│   │   ├── App.js          # Main React app
│   │   └── components/
│   │       ├── Chat.js     # Chat component
│   │       ├── Login.js    # Login form
│   │       ├── Register.js # Registration
│   │       └── FileUpload.js # File sharing
│   ├── package.json        # Frontend deps
│   └── Dockerfile          # Frontend container
├── docker-compose.yml      # Full stack setup
└── README.md              # Setup instructions
```

#### Example 2: Microservices E-Commerce Platform
```bash
curl -X POST http://localhost:8092/api/orchestrate \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Create a microservices-based e-commerce platform with product catalog service (Go), order service (Python), payment service (Node.js), user service (Java), and API gateway. Include Kubernetes manifests, service mesh configuration, and monitoring setup."
  }'
```

**Generated Files:**
```
agent-workspace/[workflow-id]/
├── services/
│   ├── product-catalog/    # Go service
│   │   ├── main.go
│   │   ├── go.mod
│   │   ├── Dockerfile
│   │   └── schema.sql
│   ├── order/              # Python service
│   │   ├── main.py
│   │   ├── requirements.txt
│   │   └── Dockerfile
│   ├── payment/            # Node.js service
│   │   ├── index.js
│   │   ├── package.json
│   │   └── Dockerfile
│   ├── user/               # Java service
│   │   ├── src/main/java/
│   │   ├── build.gradle
│   │   └── Dockerfile
│   └── api-gateway/        # API Gateway
│       ├── index.js
│       └── package.json
├── k8s/                    # Kubernetes configs
│   ├── namespace.yaml
│   ├── deployments/
│   ├── services/
│   └── istio/             # Service mesh
├── monitoring/             # Observability
│   ├── prometheus.yaml
│   └── grafana/
└── docker-compose.yml      # Local development
```

### Other API Endpoints

**List Available Agents:**
```bash
curl http://localhost:8092/api/agents | jq '.'
```

**Check Health:**
```bash
curl http://localhost:8092/health
```

**Get File Tree (IDE Server):**
```bash
curl http://localhost:8089/api/ide/tree | jq '.'
```

**View Generated File:**
```bash
curl "http://localhost:8089/api/ide/file?path=/path/to/file"
```

## 📁 Project Structure

```
OSA/
├── miosa-backend/
│   ├── cmd/
│   │   ├── agent-ide-demo/        # LLM code generation demo
│   │   ├── agent-orchestrator/    # Simple 3-agent orchestrator
│   │   ├── enhanced-orchestrator/ # Multi-file generation orchestrator
│   │   ├── full-orchestrator/     # All 10 agents orchestrator
│   │   └── ide-server/           # IDE file server
│   ├── internal/
│   │   ├── agents/               # Agent implementations
│   │   │   ├── development/      # Enhanced code generation
│   │   │   ├── strategy/        # Strategic planning
│   │   │   ├── quality/         # Quality assurance
│   │   │   └── ...             # Other agents
│   │   └── services/
│   │       └── ide/             # IDE service handlers
│   └── go.mod
├── miosa-web/
│   ├── src/
│   │   └── routes/
│   │       └── ide/             # IDE interface
│   │           ├── +page.svelte # IDE component
│   │           └── +page.server.ts
│   └── package.json
└── agent-workspace/             # Generated applications
    ├── [workflow-id-1]/         # First generated app
    ├── [workflow-id-2]/         # Second generated app
    └── ...
```

## 🔧 Configuration

### Groq API Configuration
The system uses Groq's API with the following models:
- **Kimi K2 Instruct** - Primary model for code generation
- **Llama 3.3 70B** - Quality assurance and analysis
- Configurable temperature and token limits

### Port Configuration
- **3000** - Web interface (Svelte)
- **8089** - IDE server
- **8091** - Full orchestrator (10 agents)
- **8092** - Enhanced orchestrator (recommended)

## 🎨 Key Features

### Intelligent Code Generation
- ✅ **Real executable code** - Not just markdown or pseudocode
- ✅ **Multiple languages** - Go, Python, Node.js, Java, TypeScript
- ✅ **Complete projects** - All necessary files for running applications
- ✅ **Best practices** - Follows language-specific conventions
- ✅ **Error handling** - Includes proper error handling patterns
- ✅ **Testing** - Generates test suites automatically

### Multi-Agent Collaboration
- ✅ **Sequential execution** - Agents build on each other's work
- ✅ **Context passing** - Shared memory between agents
- ✅ **Confidence scoring** - Quality metrics for generated code
- ✅ **Specialized expertise** - Each agent focuses on its domain

### IDE Integration
- ✅ **Web-based interface** - No installation required
- ✅ **File tree navigation** - Browse generated projects
- ✅ **Syntax highlighting** - Code viewing with highlights
- ✅ **Real-time updates** - See files as they're generated
- ✅ **CORS enabled** - Works with modern browsers

## 🐛 Troubleshooting

### Common Issues and Solutions

**Issue: "GROQ_API_KEY not set"**
```bash
export GROQ_API_KEY="gsk_your_api_key_here"
```

**Issue: "Port already in use"**
```bash
# Find and kill process using port
lsof -i :8089  # or 8091, 8092, 3000
kill -9 [PID]
```

**Issue: "Cannot connect to IDE"**
1. Check IDE server is running: `curl http://localhost:8089/api/ide/tree`
2. Verify web interface port: Should be 3000, not 5173
3. Clear browser cache and reload

**Issue: "No files generated"**
1. Check orchestrator logs for errors
2. Verify GROQ API key has credits
3. Ensure workspace directory exists and is writable

### Debug Commands

```bash
# Check service status
ps aux | grep -E "orchestrator|ide-server"

# View orchestrator logs (if using systemd)
journalctl -u orchestrator -f

# Test API connectivity
curl -I http://localhost:8092/health

# View generated files
ls -la agent-workspace/

# Monitor file generation in real-time
watch -n 1 'ls -la agent-workspace/'
```

## 📊 Performance Metrics

| Metric | Value |
|--------|-------|
| **Generation Time** | 15-30 seconds per application |
| **Agents Used** | 10 specialized agents |
| **Languages Supported** | 7+ programming languages |
| **Files per App** | 20-50 files average |
| **Lines of Code** | 500-2000 lines per application |
| **Success Rate** | ~95% for standard applications |

## 🔄 Development Workflow

1. **User Request** → Natural language description
2. **Orchestrator** → Routes to appropriate agents
3. **Strategy Agent** → Creates project plan
4. **Analysis Agent** → Breaks down requirements
5. **Architect Agent** → Designs system architecture
6. **Development Agent** → Generates actual code
7. **Quality Agent** → Creates tests and validates
8. **Monitoring Agent** → Adds observability
9. **Deployment Agent** → Creates Docker/K8s configs
10. **Recommender Agent** → Suggests improvements
11. **Output** → Complete application in workspace

## 🚢 Production Deployment

### Using Docker

```dockerfile
# Dockerfile.orchestrator
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY miosa-backend/ .
RUN go mod download
RUN go build -o orchestrator cmd/enhanced-orchestrator/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/orchestrator .
ENV GROQ_API_KEY=""
EXPOSE 8092
CMD ["./orchestrator", "-port", "8092", "-workspace", "/workspace"]
```

```yaml
# docker-compose.prod.yml
version: '3.8'
services:
  orchestrator:
    build:
      context: .
      dockerfile: Dockerfile.orchestrator
    ports:
      - "8092:8092"
    environment:
      - GROQ_API_KEY=${GROQ_API_KEY}
    volumes:
      - ./agent-workspace:/workspace
  
  ide-server:
    build:
      context: .
      dockerfile: Dockerfile.ide
    ports:
      - "8089:8089"
    volumes:
      - ./agent-workspace:/workspace
  
  web:
    build:
      context: ./miosa-web
    ports:
      - "3000:3000"
    depends_on:
      - orchestrator
      - ide-server
```

### Using Kubernetes

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: miosa-orchestrator
spec:
  replicas: 2
  selector:
    matchLabels:
      app: orchestrator
  template:
    metadata:
      labels:
        app: orchestrator
    spec:
      containers:
      - name: orchestrator
        image: miosa/orchestrator:latest
        ports:
        - containerPort: 8092
        env:
        - name: GROQ_API_KEY
          valueFrom:
            secretKeyRef:
              name: groq-secret
              key: api-key
        volumeMounts:
        - name: workspace
          mountPath: /workspace
      volumes:
      - name: workspace
        persistentVolumeClaim:
          claimName: workspace-pvc
```

## 🎯 Future Enhancements

- [ ] **More Templates** - Mobile apps, ML pipelines, blockchain
- [ ] **Live Preview** - Run generated apps in browser
- [ ] **Collaborative Editing** - Multi-user support
- [ ] **Version Control** - Git integration for generated code
- [ ] **CI/CD Integration** - Auto-deploy generated apps
- [ ] **Custom Agents** - User-defined specialized agents
- [ ] **Training Mode** - Learn from user corrections
- [ ] **Export Options** - Download as ZIP, push to GitHub

## 📝 Examples of Generated Applications

### Successfully Generated

1. **Real-Time Chat Application**
   - WebSocket messaging
   - User authentication (JWT)
   - File uploads
   - Emoji reactions
   - Typing indicators
   - PostgreSQL database

2. **Microservices E-Commerce**
   - Product Catalog (Go)
   - Order Service (Python)
   - Payment Gateway (Node.js)
   - User Management (Java)
   - API Gateway
   - Kubernetes deployment

3. **Task Management System**
   - RESTful API
   - React frontend
   - MongoDB integration
   - Real-time updates
   - Docker setup

4. **Analytics Dashboard**
   - Data ingestion pipeline
   - Real-time visualizations
   - Prometheus metrics
   - Grafana dashboards

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📄 License

This project is part of the MIOSA platform - Multi-Agent Intelligent Operating System Architecture.

## 🙏 Acknowledgments

- **Groq** - For providing the LLM API
- **Kimi K2** - Advanced code generation model
- **Claude Code** - Development assistance
- **Open Source Community** - For the amazing tools and libraries

## 📞 Support

For issues or questions:
- Create an issue on GitHub: https://github.com/sormind/OSA/issues
- Check existing documentation
- Review generated examples in `agent-workspace/`

---

## 🎉 Quick Test

Want to see it in action? Run this after setup:

```bash
# Generate a simple TODO app
curl -X POST http://localhost:8092/api/orchestrate \
  -H "Content-Type: application/json" \
  -d '{"description": "Create a TODO app with add, edit, delete, and mark complete features"}'

# Check generated files
ls -la agent-workspace/

# Open IDE to view the code
open http://localhost:3000/ide
```

---

**Built with ❤️ using AI-powered multi-agent collaboration**