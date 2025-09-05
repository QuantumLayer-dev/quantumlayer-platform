# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

QuantumLayer Platform V2 is an enterprise-grade AI Software Factory that transforms natural language requirements into production-ready applications. The platform uses a service mesh architecture with multi-LLM support, workflow orchestration via Temporal, and production Kubernetes infrastructure.

**Current Status**: OPERATIONAL - Workflow system complete with REST API (port 30889) and Temporal UI (port 30888).

## Core Architecture

The platform follows a microservices architecture with these key components:

### Services Structure
```
services/
├── workflow-api/         # REST API for workflow management (port 30889)
├── llm-router/          # Multi-provider LLM routing service  
├── agent-orchestrator/  # Agent coordination service
├── deployment-manager/  # Deployment automation
├── preview-service/     # Preview deployment service
└── api-docs/           # API documentation service

packages/
├── workflows/          # Temporal workflow definitions
├── llm-router/        # LLM routing logic
├── agent-**/          # Various agent implementations
├── qtest/            # Testing intelligence platform (MCP-enabled)
├── quantum-capsule/  # Code packaging system
├── sandbox-executor/ # Sandboxed code execution
├── mcp-gateway/     # MCP integration hub
└── shared/          # Shared utilities and configurations
```

### Key Technologies
- **Go 1.21+** for core services (Temporal workers, API services)
- **Temporal v2** for workflow orchestration
- **PostgreSQL 16** with user: qlayer, pass: QuantumLayer2024!
- **Kubernetes** with Istio service mesh
- **Multi-LLM Support**: Azure OpenAI, AWS Bedrock, Anthropic, OpenAI, Groq

## Development Commands

### Building & Testing
```bash
# Build all Go services
go build ./...

# Run tests
go test ./...
go test ./packages/workflows/... -v  # Test specific package

# Lint Go code
golangci-lint run ./...
go fmt ./...

# Build Docker images  
./build-and-push.sh                # Build and push to GHCR
./build-ai-components.sh           # Build AI components

# Deploy to Kubernetes
./deploy-enterprise.sh production primary  # Full deployment
kubectl apply -f infrastructure/kubernetes/  # Component deployment
```

### Service Development
```bash
# Run workflow API locally
go run services/workflow-api/main.go

# Run LLM router
go run packages/llm-router/cmd/main.go

# Run Temporal worker
go run packages/workflows/cmd/worker/main.go

# Access services
curl http://192.168.1.177:30889/api/v1/workflows/generate
curl http://192.168.1.177:30888  # Temporal UI
```

### Testing & Validation
```bash
# Test complete pipeline
./test-complete-integration.sh
./test-e2e-pipeline.sh
./test-enhanced-pipeline.sh

# Test QTest service
./test-qtest-service.sh

# Test LLM credentials
./test-llm-credentials.sh

# Validate enterprise scenarios
./validate-enterprise-scenarios.sh
```

### Infrastructure Management
```bash
# Deploy infrastructure
kubectl apply -k infrastructure/kubernetes/overlays/production

# Check status
kubectl get pods -n quantumlayer
kubectl get pods -n temporal
kubectl get svc -n quantumlayer

# Access logs
kubectl logs -n quantumlayer deployment/workflow-api
kubectl logs -n temporal deployment/temporal-frontend
```

## Workflow Architecture

The platform uses a 7-stage Temporal workflow for code generation:

1. **Parse Requirements**: NLP processing of user input
2. **Generate Architecture**: System design and component planning  
3. **Generate Code**: Parallel code generation via agents
4. **Generate Tests**: Automated test creation (QTest integration)
5. **Validate Quality**: Security, performance, coverage checks
6. **Package Solution**: Create QuantumCapsule with all artifacts
7. **Deploy Preview**: Sandbox execution and preview deployment

Workflows are dynamically routed based on complexity:
- **Simple** (<100 LOC): Single agent
- **Standard** (100-1000 LOC): Parallel agents
- **Complex** (>1000 LOC): Multi-phase orchestration
- **Enterprise**: Full service mesh with infrastructure

## Service Ports

| Service | Internal | NodePort | Description |
|---------|----------|----------|-------------|
| Workflow API | 8000 | 30889 | REST API |
| Temporal Web | 8080 | 30888 | Workflow UI |
| PostgreSQL | 5432 | 30432 | Database |
| Istio Gateway | 8080 | 31044 | HTTP ingress |

## Critical Patterns

### LLM Router Configuration
The LLM router (`packages/llm-router/`) manages multi-provider support. Configure providers via environment variables:
- `AZURE_OPENAI_KEY`, `AZURE_OPENAI_ENDPOINT`
- `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`  
- `ANTHROPIC_API_KEY`
- `OPENAI_API_KEY`

### Temporal Workflows
Workflows are defined in `packages/workflows/internal/workflows/`. Activities are in `packages/workflows/internal/activities/`. The workflow API service (`services/workflow-api/`) provides REST endpoints for triggering and monitoring.

### MCP Integration
The MCP Gateway (`packages/mcp-gateway/`) provides universal integration. QTest v2.0 (`packages/qtest/`) includes MCP server capabilities for testing intelligence.

### Quality Gates
Every code generation passes through validation:
- Syntax and type checking
- Import validation
- Security scanning  
- Test coverage (>80% required)
- Performance benchmarks

## Important Files

- `infrastructure/kubernetes/` - K8s manifests for all services
- `packages/workflows/internal/workflows/extended_generation.go` - Main workflow logic
- `services/workflow-api/main.go` - REST API server
- `packages/llm-router/router.go` - LLM routing logic
- `docs/architecture/SYSTEM_ARCHITECTURE.md` - Detailed architecture
- `docs/planning/MASTER_IMPLEMENTATION_PLAN.md` - Implementation roadmap

## Security & Credentials

- Database credentials in `infrastructure/kubernetes/postgres-deployment.yaml`
- LLM credentials via Kubernetes secrets (`llm-credentials` secret)
- Istio provides mTLS between services
- Zero-trust networking with authorization policies

## Common Tasks

### Trigger a workflow
```bash
curl -X POST http://192.168.1.177:30889/api/v1/workflows/generate \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Create a Python REST API", "language": "python", "type": "api"}'
```

### Check workflow status
```bash
curl http://192.168.1.177:30889/api/v1/workflows/{workflow_id}/status
```

### Access Temporal UI
Navigate to http://192.168.1.177:30888 to view workflow executions, activities, and debugging information.

### Deploy a new service
1. Create service in appropriate directory (`services/` or `packages/`)
2. Add Kubernetes manifests to `infrastructure/kubernetes/`
3. Update deployment scripts if needed
4. Deploy: `kubectl apply -f infrastructure/kubernetes/[service].yaml`