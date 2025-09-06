# Enterprise LLM Router Deployment Guide

## Overview
This guide provides step-by-step instructions to deploy the production-ready, enterprise-grade LLM router with proper Azure OpenAI configuration, fallback mechanisms, and monitoring.

## Architecture Improvements

### ✅ What We've Fixed
1. **Proper Azure OpenAI Integration**
   - Enterprise-grade provider implementation
   - Circuit breaker pattern for resilience
   - Rate limiting and retry logic
   - Proper error handling and logging

2. **Intelligent Validation**
   - Reduced minimum code length from 100 to 30 characters
   - More lenient pattern matching
   - Accepts structured data and various code formats
   - No longer rejects valid code with comments

3. **Enhanced System Prompts**
   - Clear instructions for code-only generation
   - No markdown or conversational responses
   - Language-specific best practices

4. **Production Features**
   - Prometheus metrics for monitoring
   - Health checks for all providers
   - Connection pooling for performance
   - Non-root container execution
   - Graceful shutdown handling

## Prerequisites

1. **Kubernetes Cluster Access**
   ```bash
   kubectl cluster-info
   ```

2. **Docker Registry Access**
   ```bash
   docker login ghcr.io
   ```

3. **At Least One LLM Provider API Key**
   - Azure OpenAI (recommended)
   - OpenAI
   - Anthropic
   - Groq
   - AWS Bedrock

## Step 1: Configure API Keys

### Create Configuration File
```bash
# Create .env.llm file
cat > .env.llm << 'EOF'
# Azure OpenAI Configuration (RECOMMENDED)
AZURE_OPENAI_KEY=your-azure-key-here
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com
AZURE_OPENAI_DEPLOYMENT=gpt-4
AZURE_OPENAI_EMBEDDING_DEPLOYMENT=text-embedding-3-small

# OpenAI Configuration (Optional)
OPENAI_API_KEY=sk-...

# Anthropic Configuration (Optional)
ANTHROPIC_API_KEY=sk-ant-...

# Groq Configuration (Optional)
GROQ_API_KEY=gsk_...

# AWS Bedrock Configuration (Optional)
AWS_ACCESS_KEY_ID=AKIA...
AWS_SECRET_ACCESS_KEY=...
AWS_BEDROCK_REGION=us-east-1
EOF
```

### Azure OpenAI Setup
1. **Create Azure OpenAI Resource**
   - Go to Azure Portal
   - Create "Azure OpenAI" resource
   - Deploy models (e.g., gpt-4, gpt-35-turbo)

2. **Get Credentials**
   ```bash
   # Endpoint format: https://<resource-name>.openai.azure.com
   # Key: Found in "Keys and Endpoint" section
   # Deployment: Name you gave when deploying the model
   ```

## Step 2: Run Enterprise Setup Script

```bash
# Run the automated setup
./setup-llm-enterprise.sh
```

This script will:
1. Validate your Kubernetes cluster connection
2. Load and validate API keys
3. Test provider connections
4. Create Kubernetes secrets
5. Deploy the improved LLM router
6. Run health checks

## Step 3: Manual Deployment (Alternative)

### A. Create Kubernetes Secret
```bash
# Load environment variables
source .env.llm

# Apply secret configuration
envsubst < infrastructure/kubernetes/llm-secrets-enterprise.yaml | kubectl apply -f -
```

### B. Build and Push Docker Image
```bash
# Build the improved router
docker build -t ghcr.io/quantumlayer-dev/llm-router:v2.5.1 \
  -f packages/llm-router/Dockerfile \
  --build-arg VERSION=v2.5.1 \
  .

# Push to registry
docker push ghcr.io/quantumlayer-dev/llm-router:v2.5.1
```

### C. Update Deployment
```bash
# Update the deployment
kubectl set image deployment/llm-router \
  llm-router=ghcr.io/quantumlayer-dev/llm-router:v2.5.1 \
  -n quantumlayer

# Wait for rollout
kubectl rollout status deployment/llm-router -n quantumlayer
```

## Step 4: Update Workflow Activities

Edit `packages/workflows/internal/activities/activities.go`:

```go
// Update the generateCodeWithLLM function around line 217
llmRequest := map[string]interface{}{
    "messages": []map[string]string{
        {
            "role": "system", 
            "content": "You are an expert code generator. Generate ONLY executable code without explanations, markdown formatting, or conversational text. Do not include code block markers (```). Return pure code only."
        },
        {"role": "user", "content": request.Prompt},
    },
    "provider": request.Provider,
    "max_tokens": request.MaxTokens,
    "temperature": 0.7,
    "language": request.Language,  // Add this
}
```

## Step 5: Verify Deployment

### Check Pod Status
```bash
kubectl get pods -n quantumlayer | grep llm-router
```

### Check Logs
```bash
kubectl logs -n quantumlayer deployment/llm-router --tail=50
```

### Test Health Endpoint
```bash
curl http://192.168.1.177:30881/health
```

Expected response:
```json
{
  "status": "healthy",
  "providers": ["azure", "groq", "openai"]
}
```

### Test Code Generation
```bash
curl -X POST http://192.168.1.177:30881/generate \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "system", "content": "Generate code only"},
      {"role": "user", "content": "Create a Python function to reverse a string"}
    ],
    "language": "python",
    "max_tokens": 500
  }'
```

## Step 6: Monitor Performance

### Prometheus Metrics
```bash
# Access metrics endpoint
curl http://192.168.1.177:30890/metrics | grep llm_router
```

Key metrics to monitor:
- `llm_router_requests_total` - Total requests by provider
- `llm_router_request_duration_seconds` - Request latency
- `llm_router_provider_errors_total` - Provider error counts
- `llm_router_token_usage_total` - Token consumption
- `llm_router_active_providers` - Provider health status

### Grafana Dashboard (Optional)
Import the dashboard from `infrastructure/monitoring/llm-router-dashboard.json`

## Troubleshooting

### Issue: "Azure API: Invalid type for 'messages'"
**Solution**: Update workflow activities to use proper message format (Step 4)

### Issue: "All providers failed"
**Check**:
1. API keys are correctly set: `kubectl get secret llm-credentials -n quantumlayer -o yaml`
2. Network connectivity: `kubectl exec -n quantumlayer deployment/llm-router -- wget -O- https://api.openai.com`
3. Provider logs: `kubectl logs -n quantumlayer deployment/llm-router | grep ERROR`

### Issue: "Code validation failed"
**Solution**: The new validation logic should fix this, but ensure:
- System prompt explicitly requests code only
- Response doesn't start with greetings

### Issue: Azure timeout errors
**Check**:
1. Endpoint format is correct (https://resource.openai.azure.com)
2. Deployment name exists in Azure
3. API version is supported (2024-02-01)

## Configuration Options

### Environment Variables
```bash
# Router configuration
PRIMARY_PROVIDER=azure        # Primary provider to use
FALLBACK_PROVIDERS=groq,openai # Comma-separated fallback list
REQUEST_TIMEOUT=30s           # Maximum request timeout
MAX_RETRIES=3                 # Retry attempts per provider

# Provider-specific
AZURE_OPENAI_API_VERSION=2024-02-01  # Azure API version
OPENAI_MODEL=gpt-4-turbo-preview     # OpenAI model
ANTHROPIC_MODEL=claude-3-opus        # Anthropic model
```

### Advanced Configuration
Edit `infrastructure/kubernetes/llm-router-config.yaml` for:
- Rate limiting per provider
- Circuit breaker settings
- Retry policies
- Cache configuration
- Custom system prompts

## Production Checklist

- [ ] At least 2 LLM providers configured
- [ ] Kubernetes secrets properly set
- [ ] Health checks passing
- [ ] Metrics endpoint accessible
- [ ] Workflow activities updated
- [ ] Test code generation working
- [ ] Monitoring configured
- [ ] Alerts set up for failures
- [ ] Rate limits configured
- [ ] Circuit breakers tested

## Support

### Logs
```bash
# Router logs
kubectl logs -n quantumlayer deployment/llm-router -f

# Workflow worker logs
kubectl logs -n temporal deployment/workflow-worker --tail=100
```

### Debug Mode
```bash
# Enable debug logging
kubectl set env deployment/llm-router LOG_LEVEL=debug -n quantumlayer
```

### Reset Deployment
```bash
# Rollback to previous version
kubectl rollout undo deployment/llm-router -n quantumlayer

# Force restart
kubectl rollout restart deployment/llm-router -n quantumlayer
```

## Summary

The enterprise LLM router now provides:
1. **Reliable code generation** with intelligent validation
2. **Multi-provider support** with automatic fallback
3. **Production monitoring** via Prometheus metrics
4. **Enterprise security** with non-root containers and secrets management
5. **High availability** with circuit breakers and retry logic

The system is designed to handle provider failures gracefully and ensure consistent code generation output for your workflow system.