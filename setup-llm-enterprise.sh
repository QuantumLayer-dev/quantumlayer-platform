#!/bin/bash

# Enterprise LLM Router Setup Script
# This script configures and deploys the production-ready LLM router

set -e

echo "==========================================="
echo "🚀 Enterprise LLM Router Setup"
echo "==========================================="

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration file
CONFIG_FILE=".env.llm"

# Function to print colored output
print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Check if running in Kubernetes cluster
check_kubernetes() {
    if ! kubectl cluster-info &> /dev/null; then
        print_error "Kubernetes cluster not accessible"
        exit 1
    fi
    print_status "Kubernetes cluster accessible"
}

# Create configuration file if it doesn't exist
create_config_file() {
    if [ ! -f "$CONFIG_FILE" ]; then
        echo "Creating configuration file: $CONFIG_FILE"
        cat > "$CONFIG_FILE" << 'EOF'
# Azure OpenAI Configuration
AZURE_OPENAI_KEY=
AZURE_OPENAI_ENDPOINT=
AZURE_OPENAI_DEPLOYMENT=
AZURE_OPENAI_EMBEDDING_DEPLOYMENT=

# AWS Bedrock Configuration
AWS_ACCESS_KEY_ID=
AWS_SECRET_ACCESS_KEY=
AWS_BEDROCK_REGION=us-east-1
AWS_BEDROCK_MODEL=anthropic.claude-3-haiku-20240307

# OpenAI Configuration
OPENAI_API_KEY=

# Anthropic Configuration
ANTHROPIC_API_KEY=

# Groq Configuration
GROQ_API_KEY=

# Optional: Clerk Authentication
CLERK_PUBLISHABLE_KEY=
CLERK_SECRET_KEY=
EOF
        print_warning "Please edit $CONFIG_FILE with your actual API keys"
        exit 1
    fi
}

# Load configuration
load_config() {
    if [ -f "$CONFIG_FILE" ]; then
        export $(cat "$CONFIG_FILE" | grep -v '^#' | xargs)
        print_status "Configuration loaded from $CONFIG_FILE"
    else
        print_error "Configuration file not found: $CONFIG_FILE"
        exit 1
    fi
}

# Validate Azure configuration
validate_azure() {
    echo ""
    echo "Validating Azure OpenAI configuration..."
    
    if [ -z "$AZURE_OPENAI_KEY" ] || [ -z "$AZURE_OPENAI_ENDPOINT" ] || [ -z "$AZURE_OPENAI_DEPLOYMENT" ]; then
        print_warning "Azure OpenAI not configured (optional)"
        return 1
    fi
    
    # Fix endpoint format
    if [[ ! "$AZURE_OPENAI_ENDPOINT" =~ ^https:// ]]; then
        AZURE_OPENAI_ENDPOINT="https://$AZURE_OPENAI_ENDPOINT"
    fi
    
    # Remove trailing slash
    AZURE_OPENAI_ENDPOINT="${AZURE_OPENAI_ENDPOINT%/}"
    
    # Validate endpoint format
    if [[ ! "$AZURE_OPENAI_ENDPOINT" =~ \.openai\.azure\.com$ ]]; then
        print_warning "Azure endpoint doesn't match expected format: $AZURE_OPENAI_ENDPOINT"
    fi
    
    # Test Azure connection
    echo "Testing Azure OpenAI connection..."
    AZURE_URL="${AZURE_OPENAI_ENDPOINT}/openai/deployments/${AZURE_OPENAI_DEPLOYMENT}/chat/completions?api-version=2024-02-01"
    
    RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$AZURE_URL" \
        -H "api-key: $AZURE_OPENAI_KEY" \
        -H "Content-Type: application/json" \
        -d '{
            "messages": [{"role": "user", "content": "test"}],
            "max_tokens": 5
        }' --max-time 10 2>/dev/null || echo "000")
    
    if [ "$RESPONSE" = "200" ]; then
        print_status "Azure OpenAI connection successful"
        return 0
    elif [ "$RESPONSE" = "401" ]; then
        print_error "Azure OpenAI authentication failed - check API key"
        return 1
    elif [ "$RESPONSE" = "404" ]; then
        print_error "Azure OpenAI deployment not found: $AZURE_OPENAI_DEPLOYMENT"
        return 1
    else
        print_warning "Azure OpenAI connection test returned: $RESPONSE"
        return 1
    fi
}

# Validate other providers
validate_providers() {
    echo ""
    echo "Validating other LLM providers..."
    
    # OpenAI
    if [ -n "$OPENAI_API_KEY" ]; then
        RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" \
            -H "Authorization: Bearer $OPENAI_API_KEY" \
            https://api.openai.com/v1/models --max-time 5 2>/dev/null || echo "000")
        
        if [ "$RESPONSE" = "200" ]; then
            print_status "OpenAI API key valid"
        else
            print_warning "OpenAI API key validation failed: $RESPONSE"
        fi
    else
        print_warning "OpenAI not configured (optional)"
    fi
    
    # Anthropic
    if [ -n "$ANTHROPIC_API_KEY" ]; then
        print_status "Anthropic API key configured"
    else
        print_warning "Anthropic not configured (optional)"
    fi
    
    # Groq
    if [ -n "$GROQ_API_KEY" ]; then
        print_status "Groq API key configured"
    else
        print_warning "Groq not configured (optional)"
    fi
    
    # AWS Bedrock
    if [ -n "$AWS_ACCESS_KEY_ID" ] && [ -n "$AWS_SECRET_ACCESS_KEY" ]; then
        print_status "AWS Bedrock credentials configured"
    else
        print_warning "AWS Bedrock not configured (optional)"
    fi
}

# Create Kubernetes secret
create_kubernetes_secret() {
    echo ""
    echo "Creating Kubernetes secret..."
    
    # Delete existing secret if it exists
    kubectl delete secret llm-credentials -n quantumlayer 2>/dev/null || true
    
    # Create new secret using envsubst
    envsubst < infrastructure/kubernetes/llm-secrets-enterprise.yaml | kubectl apply -f -
    
    print_status "Kubernetes secret created/updated"
}

# Build and deploy LLM router
deploy_llm_router() {
    echo ""
    echo "Building and deploying LLM router..."
    
    # Build Docker image
    echo "Building Docker image..."
    docker build -t ghcr.io/quantumlayer-dev/llm-router:v2.5.1 \
        -f packages/llm-router/Dockerfile \
        --build-arg VERSION=v2.5.1 \
        .
    
    # Push to registry
    echo "Pushing to registry..."
    docker push ghcr.io/quantumlayer-dev/llm-router:v2.5.1
    
    # Update deployment
    echo "Updating Kubernetes deployment..."
    kubectl set image deployment/llm-router \
        llm-router=ghcr.io/quantumlayer-dev/llm-router:v2.5.1 \
        -n quantumlayer
    
    # Wait for rollout
    kubectl rollout status deployment/llm-router -n quantumlayer
    
    print_status "LLM router deployed successfully"
}

# Test the deployment
test_deployment() {
    echo ""
    echo "Testing LLM router deployment..."
    
    # Get service endpoint
    NODE_IP="192.168.1.177"
    NODE_PORT="30881"
    
    # Wait for service to be ready
    echo "Waiting for service to be ready..."
    sleep 10
    
    # Test health endpoint
    echo "Testing health endpoint..."
    HEALTH=$(curl -s http://$NODE_IP:$NODE_PORT/health 2>/dev/null || echo "{}")
    if echo "$HEALTH" | grep -q "healthy"; then
        print_status "Health check passed"
    else
        print_error "Health check failed"
    fi
    
    # Test code generation
    echo "Testing code generation..."
    RESPONSE=$(curl -s -X POST http://$NODE_IP:$NODE_PORT/generate \
        -H "Content-Type: application/json" \
        -d '{
            "messages": [
                {"role": "system", "content": "Generate code only, no explanations"},
                {"role": "user", "content": "Write a Python function to calculate factorial"}
            ],
            "language": "python",
            "max_tokens": 200
        }' --max-time 15 2>/dev/null || echo "{}")
    
    if echo "$RESPONSE" | grep -q "def"; then
        print_status "Code generation test passed"
        echo "Sample response:"
        echo "$RESPONSE" | jq -r '.content' 2>/dev/null | head -5
    else
        print_warning "Code generation test returned unexpected response"
        echo "$RESPONSE"
    fi
}

# Update workflow activities
update_workflow_activities() {
    echo ""
    echo "Updating workflow activities..."
    
    # Check if file exists
    if [ -f "packages/workflows/internal/activities/activities.go" ]; then
        # Create backup
        cp packages/workflows/internal/activities/activities.go \
           packages/workflows/internal/activities/activities.go.bak
        
        print_status "Backup created: activities.go.bak"
        
        # Note: Manual update needed for Go files
        print_warning "Please manually update the workflow activities to use proper system prompts"
        print_warning "Example system prompt: 'Generate code only, no explanations or markdown'"
    fi
}

# Main execution
main() {
    echo "Starting enterprise LLM router setup..."
    echo ""
    
    # Check prerequisites
    check_kubernetes
    
    # Create config file if needed
    create_config_file
    
    # Load configuration
    load_config
    
    # Validate providers
    AZURE_VALID=0
    if validate_azure; then
        AZURE_VALID=1
    fi
    
    validate_providers
    
    # Check if at least one provider is configured
    if [ -z "$AZURE_OPENAI_KEY" ] && [ -z "$OPENAI_API_KEY" ] && [ -z "$ANTHROPIC_API_KEY" ] && [ -z "$GROQ_API_KEY" ]; then
        print_error "No LLM providers configured. Please configure at least one provider in $CONFIG_FILE"
        exit 1
    fi
    
    # Create Kubernetes resources
    create_kubernetes_secret
    
    # Deploy LLM router
    read -p "Do you want to build and deploy the new LLM router? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        deploy_llm_router
        test_deployment
    fi
    
    # Update workflow activities
    update_workflow_activities
    
    echo ""
    echo "==========================================="
    echo "✅ Enterprise LLM Router Setup Complete"
    echo "==========================================="
    echo ""
    echo "Next steps:"
    echo "1. Verify the deployment: kubectl get pods -n quantumlayer | grep llm-router"
    echo "2. Check logs: kubectl logs -n quantumlayer deployment/llm-router"
    echo "3. Test endpoint: curl http://$NODE_IP:$NODE_PORT/health"
    echo "4. Monitor metrics: http://$NODE_IP:30890/metrics"
    echo ""
    
    if [ $AZURE_VALID -eq 0 ]; then
        echo "⚠️  Azure OpenAI needs configuration:"
        echo "   - Set valid API key, endpoint, and deployment name in $CONFIG_FILE"
        echo "   - Endpoint format: https://YOUR-RESOURCE.openai.azure.com"
        echo ""
    fi
}

# Run main function
main "$@"