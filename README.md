# QuantumLayer Platform

[![GitHub](https://img.shields.io/github/license/QuantumLayer-dev/quantumlayer-platform)](LICENSE)
[![Kubernetes](https://img.shields.io/badge/kubernetes-ready-blue)](infrastructure/kubernetes/)
[![Multi-LLM](https://img.shields.io/badge/LLM-Multi--Provider-green)](docs/architecture/SYSTEM_ARCHITECTURE.md)

Enterprise-grade AI Software Factory Platform with multi-LLM support, multi-tenancy, and Kubernetes-native deployment.

## 🚀 Quick Start

```bash
# Clone the repository
git clone git@github.com:QuantumLayer-dev/quantumlayer-platform.git
cd quantumlayer-platform

# Setup environment
cp .env.k8s .env
make setup

# Deploy to Kubernetes
kubectl apply -f infrastructure/kubernetes/

# Access services (replace with your cluster IP)
# API: http://192.168.7.235:30800
# Web: http://192.168.7.235:30300
# Grafana: http://192.168.7.235:30301
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

## 🏗️ Architecture

QuantumLayer is built with a microservices architecture designed for scale:

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