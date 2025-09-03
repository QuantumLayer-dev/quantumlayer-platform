#!/bin/bash

# Secure script to configure LLM credentials from .env file
# This avoids exposing secrets in terminal history or command line

set -e

echo "==================================================="
echo "🔐 Secure LLM Credentials Setup from .env"
echo "==================================================="
echo ""

# Check if .env file exists
if [ ! -f .env ]; then
    echo "❌ .env file not found!"
    echo ""
    echo "Please create a .env file with your credentials:"
    echo "  cp .env.template .env"
    echo "  # Edit .env with your actual credentials"
    echo ""
    exit 1
fi

# Source the .env file
echo "Loading credentials from .env file..."
set -a  # automatically export all variables
source .env
set +a  # turn off automatic export

# Validate required credentials
MISSING_CREDS=false

echo ""
echo "Checking credentials..."
echo "----------------------------------------"

# Check AWS credentials
if [ -z "$AWS_ACCESS_KEY_ID" ] || [ "$AWS_ACCESS_KEY_ID" = "your-aws-access-key-id" ]; then
    echo "⚠️  AWS_ACCESS_KEY_ID not configured"
else
    echo "✅ AWS credentials found"
fi

# Check Azure credentials
if [ -z "$AZURE_OPENAI_KEY" ] || [ "$AZURE_OPENAI_KEY" = "your-azure-openai-key" ]; then
    echo "⚠️  AZURE_OPENAI_KEY not configured"
else
    echo "✅ Azure OpenAI credentials found"
fi

# Check optional providers
if [ ! -z "$OPENAI_API_KEY" ] && [ "$OPENAI_API_KEY" != "sk-your-openai-key" ]; then
    echo "✅ OpenAI credentials found"
fi

if [ ! -z "$ANTHROPIC_API_KEY" ] && [ "$ANTHROPIC_API_KEY" != "sk-ant-your-anthropic-key" ]; then
    echo "✅ Anthropic credentials found"
fi

echo ""
echo "----------------------------------------"
echo "Creating Kubernetes secrets..."
echo "----------------------------------------"

# Ensure namespace exists
kubectl create namespace quantumlayer --dry-run=client -o yaml | kubectl apply -f -

# Create AWS Bedrock secrets
if [ ! -z "$AWS_ACCESS_KEY_ID" ] && [ "$AWS_ACCESS_KEY_ID" != "your-aws-access-key-id" ]; then
    echo "Creating AWS Bedrock secrets..."
    kubectl create secret generic aws-bedrock-secrets \
        --from-literal=AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_ID" \
        --from-literal=AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY" \
        --from-literal=AWS_BEDROCK_REGION="${AWS_BEDROCK_REGION:-us-east-1}" \
        --from-literal=AWS_BEDROCK_MODEL="${AWS_BEDROCK_MODEL:-anthropic.claude-3-sonnet-20240229-v1:0}" \
        -n quantumlayer --dry-run=client -o yaml | kubectl apply -f -
    echo "✅ AWS Bedrock secrets created"
fi

# Create Azure OpenAI secrets
if [ ! -z "$AZURE_OPENAI_KEY" ] && [ "$AZURE_OPENAI_KEY" != "your-azure-openai-key" ]; then
    echo "Creating Azure OpenAI secrets..."
    kubectl create secret generic azure-openai-secrets \
        --from-literal=AZURE_OPENAI_ENDPOINT="${AZURE_OPENAI_ENDPOINT}" \
        --from-literal=AZURE_OPENAI_KEY="$AZURE_OPENAI_KEY" \
        --from-literal=AZURE_OPENAI_DEPLOYMENT="${AZURE_OPENAI_DEPLOYMENT:-gpt-4}" \
        -n quantumlayer --dry-run=client -o yaml | kubectl apply -f -
    echo "✅ Azure OpenAI secrets created"
fi

# Create general LLM secrets
echo "Creating general LLM secrets..."
kubectl create secret generic llm-secrets \
    --from-literal=OPENAI_API_KEY="${OPENAI_API_KEY:-not-configured}" \
    --from-literal=ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY:-not-configured}" \
    --from-literal=GROQ_API_KEY="${GROQ_API_KEY:-not-configured}" \
    -n quantumlayer --dry-run=client -o yaml | kubectl apply -f -
echo "✅ General LLM secrets created"

# Create Clerk authentication secrets if configured
if [ ! -z "$CLERK_SECRET_KEY" ] && [ "$CLERK_SECRET_KEY" != "sk_live_your-clerk-secret-key" ]; then
    echo "Creating Clerk authentication secrets..."
    kubectl create secret generic clerk-secrets \
        --from-literal=CLERK_SECRET_KEY="$CLERK_SECRET_KEY" \
        --from-literal=CLERK_PUBLISHABLE_KEY="${CLERK_PUBLISHABLE_KEY}" \
        -n quantumlayer --dry-run=client -o yaml | kubectl apply -f -
    echo "✅ Clerk secrets created"
fi

echo ""
echo "==================================================="
echo "✅ Secrets created successfully!"
echo "==================================================="
echo ""

# List created secrets (without exposing values)
echo "Created secrets in quantumlayer namespace:"
kubectl get secrets -n quantumlayer --no-headers | grep -E "(llm-secrets|aws-bedrock|azure-openai|clerk)" | awk '{print "  - "$1}'

echo ""
echo "Next steps:"
echo "1. Test credentials: ./test-llm-from-env.sh"
echo "2. Deploy services: kubectl apply -f infrastructure/kubernetes/"
echo ""
echo "Security reminders:"
echo "  ✅ .env file is in .gitignore (never commit it!)"
echo "  ✅ Secrets are stored securely in Kubernetes"
echo "  ✅ No credentials exposed in shell history"