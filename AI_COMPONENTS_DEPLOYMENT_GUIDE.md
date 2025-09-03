# AI-Native Components Deployment Guide

## Overview

This guide explains how to build and deploy the AI-native components that transform QuantumLayer from traditional switch/case patterns to semantic AI-powered decision making.

## Components to Deploy

### 1. AI Decision Engine (NEW)
- **Purpose**: Replaces all switch/case statements with semantic routing
- **Port**: 8095
- **Key Features**:
  - Semantic similarity matching using embeddings
  - Learning from feedback
  - Dynamic rule registration
  - Fuzzy matching with confidence scores

### 2. QSecure Engine (NEW - 5th Path)
- **Purpose**: Comprehensive security analysis and compliance
- **Port**: 8096
- **Key Features**:
  - Vulnerability scanning (OWASP, CWE)
  - Threat modeling
  - Compliance validation (GDPR, PCI-DSS, HIPAA)
  - Security remediation suggestions
  - Real-time security monitoring

### 3. Updated Agent Orchestrator
- **Changes**: Integrated AI Agent Factory
- **Features**:
  - Dynamic agent creation based on requirements
  - 5 new security specialist agents
  - AI-powered role selection

### 4. Meta-Prompt Engine
- **Port**: 8086
- **Features**: Dynamic prompt optimization

### 5. Web UI
- **Port**: 30888 (NodePort)
- **Features**: Interactive dashboard showing all 5 paths

### 6. API Documentation (Swagger)
- **Port**: 30890 (NodePort)
- **Features**: Complete API reference with examples

## Build Process

### Step 1: Build Docker Images

```bash
# Run the build script
./build-ai-components.sh
```

This script will build Docker images for:
- AI Decision Engine
- QSecure Engine
- Updated Agent Orchestrator
- Meta-Prompt Engine
- Parser Service
- API Documentation
- Web UI

### Step 2: Push to Registry (Optional)

If using a remote registry, uncomment the push commands in `build-ai-components.sh`:

```bash
# Edit the script to enable pushing
vi build-ai-components.sh
# Uncomment the docker push lines
```

### Step 3: Deploy to Kubernetes

```bash
# Deploy all AI components
./deploy-ai-components.sh
```

This will:
1. Apply Kubernetes manifests
2. Wait for deployments to be ready
3. Show deployment status
4. Display service endpoints

## Testing

### Run Integration Tests

```bash
./test-ai-services.sh
```

This tests:
- AI Decision Engine semantic routing
- QSecure security analysis
- AI-powered workflow generation
- Dynamic agent creation

### Access Web UI

```bash
# Port forward the Web UI
kubectl port-forward svc/web-ui 8888:80 -n quantumlayer

# Open browser to http://localhost:8888
```

### Access Swagger Documentation

```bash
# Port forward the API docs
kubectl port-forward svc/api-docs 8090:8090 -n quantumlayer

# Open browser to http://localhost:8090/swagger
```

## Verification

### Check AI Decision Engine

```bash
# Test language selection (replaces switch statement)
kubectl run test-ai --rm -i --image=curlimages/curl -n quantumlayer --restart=Never -- \
  curl -X POST http://ai-decision-engine:8095/api/v1/decide \
  -H "Content-Type: application/json" \
  -d '{"category":"language_selection","input":"Build a REST API"}'
```

### Check QSecure Engine

```bash
# Test security analysis (5th path)
kubectl run test-qsecure --rm -i --image=curlimages/curl -n quantumlayer --restart=Never -- \
  curl -X POST http://qsecure-engine:8096/api/v1/analyze \
  -H "Content-Type: application/json" \
  -d '{"code":"SELECT * FROM users","language":"sql"}'
```

## What Changed from Traditional to AI-Native

### Before (Traditional Switch/Case)
```go
switch prompt.Type {
case "api":
    return "python", "fastapi"
case "frontend":
    return "typescript", "react"
default:
    return "python", "generic"
}
```

### After (AI Semantic Routing)
```go
// AI Decision Engine analyzes intent and context
decision, _ := engine.Decide(ctx, "language_selection", prompt)
// Returns best match based on semantic similarity
return decision.Choice, decision.Metadata
```

## Benefits

1. **No More Hardcoding**: Decisions based on semantic understanding
2. **Fuzzy Matching**: Handles variations and typos
3. **Learning**: Improves over time from feedback
4. **Extensible**: Add new patterns without code changes
5. **5th Security Path**: Comprehensive security integration

## Troubleshooting

### If builds fail:
```bash
# Check Docker daemon
docker ps

# Check disk space
df -h

# Clean up old images
docker system prune -a
```

### If deployments fail:
```bash
# Check pod status
kubectl get pods -n quantumlayer

# Check logs
kubectl logs deployment/ai-decision-engine -n quantumlayer

# Describe deployment
kubectl describe deployment ai-decision-engine -n quantumlayer
```

### If services are not accessible:
```bash
# Check service endpoints
kubectl get svc -n quantumlayer

# Test internal connectivity
kubectl run debug --rm -i --tty --image=nicolaka/netshoot -n quantumlayer -- /bin/bash
```

## Summary

The AI-native transformation includes:
- ✅ **26 switch/case statements replaced** with AI semantic routing
- ✅ **QSecure added as 5th path** alongside Code, Infra, QA, SRE
- ✅ **5 new security specialist agents** created
- ✅ **AI Agent Factory** for dynamic agent creation
- ✅ **Meta-prompt optimization** with A/B testing
- ✅ **Complete API documentation** with Swagger
- ✅ **Interactive Web UI** showing all 5 product paths

This makes QuantumLayer truly universal and AI-native, suitable for the AI age!