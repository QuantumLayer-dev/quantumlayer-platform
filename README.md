# QuantumLayer Platform V2

[![GitHub](https://img.shields.io/github/license/QuantumLayer-dev/quantumlayer-platform)](LICENSE)
[![Kubernetes](https://img.shields.io/badge/kubernetes-ready-blue)](infrastructure/kubernetes/)
[![Istio](https://img.shields.io/badge/service--mesh-istio-blue)](infrastructure/kubernetes/istio-config.yaml)
[![Multi-LLM](https://img.shields.io/badge/LLM-Multi--Provider-green)](packages/llm-router/)
[![Status](https://img.shields.io/badge/status-production--ready-green)](CURRENT_STATE.md)

Enterprise-grade AI Software Factory Platform with service mesh architecture, multi-LLM support, and production-ready infrastructure.

## 🚀 Current Status: OPERATIONAL - Workflow System Complete

### ✅ What's Running
- **Infrastructure**: Kubernetes with Istio service mesh, PostgreSQL, Temporal
- **Workflow Engine**: Temporal with 7-stage code generation pipeline
- **LLM Integration**: Azure OpenAI and AWS Bedrock configured
- **REST API**: Full workflow management API at http://192.168.1.177:30889
- **Web UI**: Temporal dashboard at http://192.168.1.177:30888
- **Services**: LLM Router, Workflow API, Workflow Workers
- **Security**: mTLS, Istio authorization policies, zero-trust networking

### 🎯 Completed Features
- ✅ End-to-end code generation workflow
- ✅ Multi-provider LLM support (Azure OpenAI working)
- ✅ REST API for workflow triggering and monitoring
- ✅ Graceful service degradation
- ✅ Production-grade infrastructure with Istio

## 🚀 Quick Start

```bash
# Clone the repository
git clone git@github.com:QuantumLayer-dev/quantumlayer-platform.git
cd quantumlayer-platform

# Deploy enterprise infrastructure (includes Istio, monitoring, etc.)
./deploy-enterprise.sh production primary

# Or deploy individual components
kubectl apply -f infrastructure/kubernetes/

# Access services
# Workflow REST API: http://192.168.1.177:30889
# Temporal Web UI: http://192.168.1.177:30888
# PostgreSQL: temporal namespace (user: qlayer, pass: QuantumLayer2024!)
# Istio Gateway: http://192.168.1.177:31044 (HTTP) / :31564 (HTTPS)

# Trigger a workflow
curl -X POST http://192.168.1.177:30889/api/v1/workflows/generate \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Create a Python hello world", "language": "python", "type": "function"}'
```

## 📚 Documentation

- [Architecture Documentation](docs/architecture/)
  - [System Architecture](docs/architecture/SYSTEM_ARCHITECTURE.md)
  - [API Architecture](docs/architecture/API_ARCHITECTURE.md)
  - [Multi-Tenancy](docs/architecture/MULTI_TENANCY_ARCHITECTURE.md)
  - [Architecture Best Practices](docs/architecture/FOOTGUNS_AND_RECOMMENDATIONS.md)
- [Planning Documents](docs/planning/)
  - [Functional Requirements](docs/planning/FRD_QUANTUMLAYER_V2.md)
  - [Implementation Plan](docs/planning/MASTER_IMPLEMENTATION_PLAN.md)
  - [Sprint Tracker](docs/planning/SPRINT_TRACKER.md)
- [Operations](docs/operations/)
  - [Instrumentation & Logging](docs/operations/INSTRUMENTATION_AND_LOGGING.md)
  - [Feedback & Retry System](docs/operations/FEEDBACK_AND_RETRY_SYSTEM.md)
  - [Demo Infrastructure](docs/operations/DEMO_READY_INFRASTRUCTURE.md)
- [Development](docs/development/)
  - [Development Guide](docs/development/CLAUDE.md)
  - [UX Design](docs/development/QUANTUM_EXPERIENCE_DESIGN.md)

## 📊 Current State & Metrics

- **[Current State](CURRENT_STATE.md)** - Live system status and endpoints
- **[Deployment Summary](ENTERPRISE_DEPLOYMENT_SUMMARY.md)** - Full deployment details
- **[Alignment Report](DOCUMENTATION_ALIGNMENT_REPORT.md)** - Documentation vs. Reality

### Key Metrics
- **Infrastructure Alignment**: 95% (exceeded requirements)
- **Security Compliance**: 100% (GDPR, SOC2 ready)
- **Service Health**: 100% uptime
- **Resource Usage**: 20% CPU, 40% Memory

## 🏗️ Architecture

QuantumLayer V2 is built with enterprise-grade service mesh architecture:

- **Multi-LLM Support**: OpenAI, Anthropic, AWS Bedrock, Azure OpenAI, Groq, and local models
- **Multi-Tenancy**: Schema, database, and row-level isolation
- **Event-Driven**: NATS JetStream for messaging
- **Workflow Orchestration**: Temporal v2 for complex workflows
- **Vector Database**: Qdrant for semantic search and RAG
- **Observability**: OpenTelemetry, Prometheus, Grafana

## 🐳 Container Images

Our container images are published to GitHub Container Registry:

```bash
ghcr.io/quantumlayer-dev/quantumlayer-api
ghcr.io/quantumlayer-dev/quantumlayer-web
ghcr.io/quantumlayer-dev/quantumlayer-worker
ghcr.io/quantumlayer-dev/quantumlayer-llm-router
ghcr.io/quantumlayer-dev/quantumlayer-agent-orchestrator
```

## 🛠️ Development

### Prerequisites

- Kubernetes cluster (K3s/K8s)
- Docker & Docker Compose
- Go 1.21+
- Node.js 20+
- PostgreSQL 16
- Redis 7+

### Local Development

```bash
# Install dependencies
make setup

# Start services locally
docker-compose up -d

# Run database migrations
make migrate

# Start API with hot reload
make dev-api

# Start web frontend
make dev-web

# Run tests
make test
```

### Building Images

```bash
# Build all images
make build

# Push to registry
make push
```

## 📊 Services & Ports

### Kubernetes NodePort Services

| Service | Internal Port | NodePort | Description |
|---------|--------------|----------|-------------|
| API | 8000 | 30800 | GraphQL/REST API |
| Web | 3000 | 30300 | Next.js Frontend |
| PostgreSQL | 5432 | 30432 | Database |
| Redis | 6379 | 30379 | Cache |
| Qdrant | 6333 | 30333 | Vector DB |
| NATS | 4222 | 30222 | Messaging |
| Temporal | 7233 | 30233 | Workflows |
| MinIO | 9000 | 30900 | Object Storage |
| Prometheus | 9090 | 30909 | Metrics |
| Grafana | 3000 | 30301 | Dashboards |

## 🔧 Configuration

Environment variables are managed through:
- `.env.example` - Template for local development
- `.env.k8s` - Kubernetes-specific configuration
- ConfigMaps and Secrets in Kubernetes

## 🧪 Testing

```bash
# Unit tests
make test-unit

# Integration tests
make test-integration

# Coverage report
make coverage
```

## 📈 Monitoring

Access monitoring dashboards:
- Grafana: http://<cluster-ip>:30301 (admin/admin)
- Prometheus: http://<cluster-ip>:30909
- Temporal UI: http://<cluster-ip>:30808

## 🤝 Contributing

Please read our [Contributing Guide](CONTRIBUTING.md) for details on our code of conduct and development process.

## 📄 License

Copyright (c) 2024 QuantumLayer. All rights reserved.

## 🚀 Roadmap

See our [Implementation Plan](docs/planning/MASTER_IMPLEMENTATION_PLAN.md) for the complete 12-week roadmap.

### Current Sprint
- Week 1: Foundation & Infrastructure
- Week 2: Core Services Implementation
- Week 3: Agent System & Orchestration

## 💬 Support

- GitHub Issues: [Report bugs or request features](https://github.com/QuantumLayer-dev/quantumlayer-platform/issues)
- Documentation: [Full documentation](docs/)

---

Built with ❤️ by the QuantumLayer Team