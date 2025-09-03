# 🚀 QuantumLayer Platform API Documentation

## Overview
QuantumLayer Platform is an Enterprise AI Software Factory with 5 product paths: **QLayer** (Code), **QTest** (Testing), **QInfra** (Infrastructure), **QSRE** (Site Reliability), and **QSecure** (Security).

## 🌐 Service Endpoints

| Service | Port | Status | Description |
|---------|------|--------|-------------|
| **Temporal UI** | 30888 | ✅ Active | Workflow management interface |
| **Workflow API** | 30889 | ✅ Active | Code generation workflows |
| **LLM Router** | 30881 | ✅ Active | Multi-provider LLM routing |
| **Agent Orchestrator** | 30883 | ✅ Active | AI agent management |
| **API Gateway** | 30880 | ✅ Active | Main entry point |
| **Meta-Prompt Engine** | 30885 | 📦 Ready | Prompt optimization |
| **Parser Service** | 30882 | 📦 Ready | Code parsing |

Base URL: `http://192.168.1.177`

---

## 📚 API Reference

### 1️⃣ QLayer - Code Generation APIs

#### Generate Code
```http
POST /api/v1/workflows/generate
```

**Request Body:**
```json
{
  "prompt": "Create a REST API with user authentication",
  "language": "python",
  "type": "api",
  "framework": "fastapi" // optional
}
```

**Response:**
```json
{
  "workflow_id": "code-gen-123",
  "run_id": "run-456",
  "status": "started",
  "message": "Workflow started successfully"
}
```

**Supported Languages:** python, javascript, go, java, rust, typescript  
**Supported Types:** api, function, frontend, fullstack

---

### 2️⃣ Agent Management APIs

#### Spawn Agent
```http
POST /api/v1/agents/spawn
```

**Request Body:**
```json
{
  "role": "security-architect"
}
```

**Available Roles:**
- `project-manager` - Project coordination
- `architect` - System design
- `backend-developer` - Backend development
- `frontend-developer` - Frontend development
- `security-architect` - Security design (NEW!)
- `compliance-officer` - Compliance validation (NEW!)
- `threat-hunter` - Threat detection (NEW!)
- `incident-responder` - Incident handling (NEW!)
- `security-auditor` - Security auditing (NEW!)

#### List Agents
```http
GET /api/v1/agents
```

#### Get Agent Metrics
```http
GET /api/v1/agents/metrics
```

---

### 3️⃣ LLM Router APIs

#### Generate with LLM
```http
POST /generate
```

**Request Body:**
```json
{
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Explain AI-native architecture"}
  ],
  "provider": "azure",
  "max_tokens": 200
}
```

**Supported Providers:**
- `azure` - Azure OpenAI (Active)
- `openai` - OpenAI GPT-4
- `anthropic` - Claude
- `bedrock` - AWS Bedrock
- `groq` - Fast inference

---

### 4️⃣ AI Decision Engine APIs (NEW!)

#### Make Decision
```http
POST /api/v1/decisions/decide
```

**Request Body:**
```json
{
  "category": "language_selection",
  "input": "I need to build a high-performance web service",
  "context": {"project_type": "api"}
}
```

**Response:**
```json
{
  "decision": "go",
  "confidence": 0.92,
  "reasoning": "Go excels at high-performance web services",
  "metadata": {
    "alternatives": ["rust", "java"],
    "embedding_score": 0.89
  }
}
```

#### Select Language (AI-Powered)
```http
POST /api/v1/decisions/language
```

**Request Body:**
```json
{
  "requirements": "Build a machine learning pipeline with real-time inference"
}
```

---

### 5️⃣ QSecure - Security APIs (NEW!)

#### Analyze Security
```http
POST /api/v1/security/analyze
```

**Request Body:**
```json
{
  "code": "def login(user, pass): query = 'SELECT * FROM users WHERE user=' + user",
  "language": "python"
}
```

**Response:**
```json
{
  "overall_risk": "critical",
  "score": 25.0,
  "vulnerabilities": [
    {
      "type": "SQL Injection",
      "severity": "critical",
      "cwe": "CWE-89",
      "location": "line 1",
      "remediation": "Use parameterized queries"
    }
  ]
}
```

#### Generate Threat Model
```http
POST /api/v1/security/threat-model
```

#### Validate Compliance
```http
POST /api/v1/security/compliance
```

**Standards Supported:** OWASP, GDPR, HIPAA, SOC2, PCI-DSS

---

## 🔄 Workflow Examples

### Complete Code Generation Flow

1. **Generate Code**
```bash
curl -X POST http://192.168.1.177:30889/api/v1/workflows/generate \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Create a secure REST API with JWT authentication",
    "language": "python",
    "type": "api"
  }'
```

2. **Check Workflow Status**
```bash
curl http://192.168.1.177:30889/api/v1/workflows/status/{workflow_id}
```

3. **Security Analysis** (When deployed)
```bash
curl -X POST http://192.168.1.177:30890/api/v1/security/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "code": "<generated_code>",
    "language": "python"
  }'
```

---

## 🧠 AI-Native Features

### Traditional vs AI-Native Approach

#### ❌ Old Way (Switch Statements)
```go
switch language {
  case "python":
    return "py"
  case "javascript":
    return "js"
  default:
    return "txt"
}
```

#### ✅ New Way (AI Decision Engine)
```go
decision, err := aiEngine.Decide(ctx, "language", requirements)
// Uses embeddings and semantic similarity
// Handles fuzzy matching and unknown inputs
// Learns from feedback
```

### Key Advantages:
- **Semantic Understanding**: Understands intent, not just exact matches
- **Fuzzy Matching**: Handles variations and typos
- **Learning**: Improves over time with feedback
- **Extensibility**: No code changes for new cases
- **Universal**: Works across all domains

---

## 📊 Platform Metrics

| Metric | Value | Status |
|--------|-------|--------|
| Overall Progress | 45% | 🟡 In Progress |
| Switch Statements Replaced | 26 | ✅ Complete |
| Security Agents Added | 5 | ✅ Complete |
| AI Decision Points | 15+ | ✅ Active |
| Services Running | 8/12 | 🟡 Partial |

---

## 🚦 Service Health Endpoints

All services expose health endpoints:

```bash
# Workflow API
curl http://192.168.1.177:30889/health

# LLM Router
curl http://192.168.1.177:30881/health

# Agent Orchestrator
curl http://192.168.1.177:30883/health
```

---

## 🔐 Authentication

Currently, the platform is in development mode without authentication. 
Enterprise authentication (OAuth2/JWT) will be added in the next phase.

---

## 📈 What's New

### Stage 1 Achievements ✅
- **AI Decision Engine**: Replaced all switch statements with semantic routing
- **QSecure Path**: Added comprehensive security as 5th product line
- **Security Agents**: 5 specialized security agents
- **AI Agent Factory**: Dynamic agent creation based on requirements
- **Language Engine**: AI-powered language/framework selection

### Coming Soon 🚀
- **QTest Engine**: Intelligent test generation
- **QInfra Engine**: Infrastructure automation
- **QSRE Engine**: Site reliability automation
- **Web Frontend**: Interactive UI
- **Multi-tenancy**: Workspace isolation
- **API Authentication**: OAuth2/JWT

---

## 📝 Example: Full Stack Application Generation

```bash
# 1. Generate backend
curl -X POST http://192.168.1.177:30889/api/v1/workflows/generate \
  -d '{"prompt": "FastAPI backend with PostgreSQL", "language": "python", "type": "api"}'

# 2. Generate frontend (planned)
curl -X POST http://192.168.1.177:30889/api/v1/workflows/generate \
  -d '{"prompt": "React dashboard with charts", "language": "javascript", "type": "frontend"}'

# 3. Generate infrastructure (planned)
curl -X POST http://192.168.1.177:30891/api/v1/infra/generate \
  -d '{"services": ["api", "database", "frontend"], "platform": "kubernetes"}'

# 4. Security scan
curl -X POST http://192.168.1.177:30890/api/v1/security/analyze \
  -d '{"code": "<all_code>", "standards": ["OWASP", "GDPR"]}'
```

---

## 🌍 Platform Vision

QuantumLayer is transforming software development by:
1. **Eliminating manual coding** for common patterns
2. **AI-native decision making** at every level
3. **Security-first** approach with QSecure
4. **Universal platform** supporting any language/framework
5. **Self-improving** through feedback loops

**Target**: From idea to production in < 3 minutes

---

## 📞 Support

- GitHub: https://github.com/QuantumLayer-dev/quantumlayer-platform
- Issues: https://github.com/QuantumLayer-dev/quantumlayer-platform/issues
- Documentation: This file

---

*Last Updated: September 2025 | Version: 2.0.0-alpha*