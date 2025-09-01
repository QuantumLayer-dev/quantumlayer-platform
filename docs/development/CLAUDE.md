# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

QuantumLayer Platform V2 is an enterprise-grade AI software factory that transforms natural language requirements into production-ready applications. The platform consists of four integrated products:
- **QLayer**: Code generation engine
- **QTest**: Automated testing suite
- **QInfra**: Infrastructure automation
- **QSRE**: Site reliability engineering

## Architecture

The platform follows a monorepo structure with clean architecture principles:

```
quantumlayer-platform/
├── apps/           # Application entry points
├── packages/       # Core business logic and products
├── infrastructure/ # Kubernetes, Terraform, Docker configs
├── configs/        # Environment configurations
└── tools/          # Scripts and generators
```

## Technology Stack

**Backend:**
- Go 1.22+ for core services (performance-critical paths)
- Temporal v2 for workflow orchestration
- PostgreSQL 16 (primary database) + Redis (cache)
- GraphQL (primary API), REST (compatibility), gRPC (internal)
- NATS JetStream for messaging
- Weaviate for vector database

**Frontend:**
- Next.js 14 with App Router
- Tailwind CSS + Radix UI
- Zustand + React Query for state management
- Clerk for authentication

**Infrastructure:**
- Kubernetes with Helm charts
- Docker with multi-stage builds
- GitHub Actions + ArgoCD for CI/CD
- Prometheus + Grafana for monitoring

## Development Commands

### Initial Setup
```bash
# Initialize monorepo with Turborepo
npx create-turbo@latest

# Install dependencies
npm install

# Setup Go modules
go mod init github.com/quantumlayer/platform
go mod tidy

# Initialize Temporal development server
temporal server start-dev

# Setup PostgreSQL and Redis
docker-compose up -d postgres redis
```

### Development
```bash
# Run all services in development mode
npm run dev

# Run specific service
npm run dev --filter=api
npm run dev --filter=web

# Run Go services
go run apps/api/main.go
go run apps/worker/main.go

# Run tests
npm run test
go test ./...

# Run specific test
go test ./packages/qlayer/... -v
npm run test --filter=qtest

# Linting and formatting
npm run lint
go fmt ./...
golangci-lint run

# Type checking
npm run typecheck
```

### Building
```bash
# Build all packages
npm run build

# Build Docker images
docker build -f infrastructure/docker/api.Dockerfile -t quantumlayer/api:latest .
docker build -f infrastructure/docker/web.Dockerfile -t quantumlayer/web:latest .

# Build Go binaries
go build -o bin/api apps/api/main.go
go build -o bin/worker apps/worker/main.go
```

### Infrastructure
```bash
# Deploy to local Kubernetes
kubectl apply -k infrastructure/kubernetes/overlays/development

# Deploy with Helm
helm install quantumlayer infrastructure/helm/quantumlayer

# Run database migrations
go run tools/migrate/main.go up

# Generate GraphQL schemas
go run tools/gqlgen/main.go
```

## Core Architecture Patterns

### 1. Workflow System
The platform uses Temporal v2 for orchestrating complex workflows. Workflows are dynamically routed based on complexity:
- **Simple** (<100 LOC): Single agent execution
- **Standard** (100-1000 LOC): Parallel agent coordination
- **Complex** (>1000 LOC): Multi-phase orchestration
- **Enterprise**: Full service mesh with infrastructure

### 2. QuantumCapsule Pattern
All generated code is packaged into self-contained QuantumCapsules containing:
- Source code and tests
- Docker/Kubernetes configurations
- Environment configurations
- Dependencies with lock files
- Preview deployment specs

### 3. NLP Processing Pipeline
User input flows through three stages:
1. **Parse**: Extract goals, domain, and complexity
2. **Enrich**: Add context and implicit requirements
3. **Validate**: Check feasibility and hallucinations

### 4. Quality Gates
Every generation passes through validation:
- Syntax and type checking
- Import and API validation
- Security scanning
- Performance testing
- Coverage requirements (>80%)

### 5. HITL/AITL Integration
- **HITL** (Human in the Loop): Strategic checkpoints for approval
- **AITL** (AI in the Loop): Continuous learning from execution patterns

## Performance Requirements
- API Response: <100ms (p99)
- Code Generation: <30s simple, <2m complex
- Preview Deployment: <60s
- Cache Hit Rate: >60%
- Uptime: 99.99%

## Security Considerations
- Zero-trust architecture
- All secrets in environment variables or secret management
- Automated security scanning on every build
- RBAC for multi-tenancy
- SOC2 compliance requirements

## Key Services

### QLayer Engine (packages/qlayer/)
Handles code generation from natural language:
- Requirements parsing with NLP
- Dynamic execution planning
- Parallel code generation
- Quality validation
- Deployment preparation

### QTest Engine (packages/qtest/)
Automated testing infrastructure:
- Test generation from code
- Self-healing test capabilities
- Coverage analysis
- Performance benchmarking
- Security scanning

### API Gateway (apps/api/)
Unified entry point for all services:
- GraphQL schema management
- Authentication middleware
- Rate limiting and caching
- Metrics collection
- Request routing

### Workflow Workers (apps/worker/)
Temporal workflow implementations:
- Activity definitions
- Workflow orchestration
- Error handling and retries
- Progress tracking
- Result aggregation