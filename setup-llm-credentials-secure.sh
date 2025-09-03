#!/bin/bash

# Secure script to configure LLM provider credentials for QuantumLayer
# This script uses environment variables to avoid exposing secrets

echo "==================================================="
echo "🔒 Secure LLM Provider Credentials Setup"
echo "==================================================="
echo ""
echo "IMPORTANT: This script reads credentials from environment variables"
echo "to avoid exposing them in command history or logs."
echo ""

# Function to check if secret exists
check_secret() {
    kubectl get secret $1 -n quantumlayer &>/dev/null
    return $?
}

# Option 1: Use environment variables
echo "----------------------------------------"
echo "Option 1: Using Environment Variables"
echo "----------------------------------------"
echo ""
echo "Set your credentials as environment variables first:"
echo ""
echo "export AWS_ACCESS_KEY_ID='your-key-id'"
echo "export AWS_SECRET_ACCESS_KEY='your-secret-key'"
echo "export AZURE_OPENAI_KEY='your-azure-key'"
echo "export OPENAI_API_KEY='sk-...'"
echo "export ANTHROPIC_API_KEY='sk-ant-...'"
echo ""

read -p "Have you set the environment variables? (y/n): " ENV_SET

if [ "$ENV_SET" = "y" ]; then
    # Create secrets from environment variables
    echo "Creating Kubernetes secrets from environment variables..."
    
    # AWS Bedrock
    if [ ! -z "$AWS_ACCESS_KEY_ID" ] && [ ! -z "$AWS_SECRET_ACCESS_KEY" ]; then
        kubectl create secret generic aws-bedrock-secrets \
            --from-literal=AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_ID" \
            --from-literal=AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY" \
            --from-literal=AWS_BEDROCK_REGION="${AWS_BEDROCK_REGION:-us-east-1}" \
            --from-literal=AWS_BEDROCK_MODEL="${AWS_BEDROCK_MODEL:-anthropic.claude-3-sonnet-20240229-v1:0}" \
            -n quantumlayer --dry-run=client -o yaml | kubectl apply -f -
        echo "✅ AWS Bedrock secrets configured"
    fi
    
    # Azure OpenAI
    if [ ! -z "$AZURE_OPENAI_KEY" ] && [ ! -z "$AZURE_OPENAI_ENDPOINT" ]; then
        kubectl create secret generic azure-openai-secrets \
            --from-literal=AZURE_OPENAI_ENDPOINT="$AZURE_OPENAI_ENDPOINT" \
            --from-literal=AZURE_OPENAI_KEY="$AZURE_OPENAI_KEY" \
            --from-literal=AZURE_OPENAI_DEPLOYMENT="${AZURE_OPENAI_DEPLOYMENT:-gpt-4}" \
            -n quantumlayer --dry-run=client -o yaml | kubectl apply -f -
        echo "✅ Azure OpenAI secrets configured"
    fi
    
    # OpenAI and Anthropic
    kubectl create secret generic llm-secrets \
        --from-literal=OPENAI_API_KEY="${OPENAI_API_KEY:-not-configured}" \
        --from-literal=ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY:-not-configured}" \
        --from-literal=GROQ_API_KEY="${GROQ_API_KEY:-not-configured}" \
        -n quantumlayer --dry-run=client -o yaml | kubectl apply -f -
    echo "✅ LLM secrets configured"
fi

echo ""
echo "----------------------------------------"
echo "Option 2: Using Files (More Secure)"
echo "----------------------------------------"
echo ""
echo "Create files with your secrets (one value per file):"
echo ""
echo "mkdir -p ~/.quantumlayer-secrets"
echo "echo 'your-access-key-id' > ~/.quantumlayer-secrets/aws-access-key-id"
echo "echo 'your-secret-key' > ~/.quantumlayer-secrets/aws-secret-access-key"
echo "echo 'your-openai-key' > ~/.quantumlayer-secrets/openai-api-key"
echo "echo 'your-anthropic-key' > ~/.quantumlayer-secrets/anthropic-api-key"
echo ""

read -p "Do you want to use file-based secrets? (y/n): " USE_FILES

if [ "$USE_FILES" = "y" ]; then
    SECRET_DIR="$HOME/.quantumlayer-secrets"
    
    if [ -d "$SECRET_DIR" ]; then
        echo "Creating secrets from files in $SECRET_DIR..."
        
        # AWS Bedrock
        if [ -f "$SECRET_DIR/aws-access-key-id" ] && [ -f "$SECRET_DIR/aws-secret-access-key" ]; then
            kubectl create secret generic aws-bedrock-secrets \
                --from-file=AWS_ACCESS_KEY_ID="$SECRET_DIR/aws-access-key-id" \
                --from-file=AWS_SECRET_ACCESS_KEY="$SECRET_DIR/aws-secret-access-key" \
                -n quantumlayer --dry-run=client -o yaml | kubectl apply -f -
            echo "✅ AWS Bedrock secrets created from files"
        fi
        
        # OpenAI and Anthropic
        if [ -f "$SECRET_DIR/openai-api-key" ] || [ -f "$SECRET_DIR/anthropic-api-key" ]; then
            ARGS=""
            [ -f "$SECRET_DIR/openai-api-key" ] && ARGS="$ARGS --from-file=OPENAI_API_KEY=$SECRET_DIR/openai-api-key"
            [ -f "$SECRET_DIR/anthropic-api-key" ] && ARGS="$ARGS --from-file=ANTHROPIC_API_KEY=$SECRET_DIR/anthropic-api-key"
            
            kubectl create secret generic llm-secrets $ARGS \
                -n quantumlayer --dry-run=client -o yaml | kubectl apply -f -
            echo "✅ LLM secrets created from files"
        fi
    else
        echo "❌ Directory $SECRET_DIR not found. Please create it and add your secret files."
    fi
fi

echo ""
echo "----------------------------------------"
echo "Option 3: Manual Secret Creation (Most Secure)"
echo "----------------------------------------"
echo ""
echo "For maximum security, create a YAML file locally and apply it:"
echo ""
cat << 'EOF'
# Create a file called llm-secrets.yaml (DO NOT COMMIT THIS!)
apiVersion: v1
kind: Secret
metadata:
  name: llm-secrets
  namespace: quantumlayer
type: Opaque
stringData:
  OPENAI_API_KEY: "sk-..."
  ANTHROPIC_API_KEY: "sk-ant-..."
---
apiVersion: v1
kind: Secret
metadata:
  name: aws-bedrock-secrets
  namespace: quantumlayer
type: Opaque
stringData:
  AWS_ACCESS_KEY_ID: "your-key"
  AWS_SECRET_ACCESS_KEY: "your-secret"
  AWS_BEDROCK_REGION: "us-east-1"
---
apiVersion: v1
kind: Secret
metadata:
  name: azure-openai-secrets
  namespace: quantumlayer
type: Opaque
stringData:
  AZURE_OPENAI_ENDPOINT: "https://your-resource.openai.azure.com"
  AZURE_OPENAI_KEY: "your-key"
  AZURE_OPENAI_DEPLOYMENT: "gpt-4"
EOF

echo ""
echo "Then apply with: kubectl apply -f llm-secrets.yaml"
echo "And delete the file: rm llm-secrets.yaml"
echo ""

echo "----------------------------------------"
echo "Verification"
echo "----------------------------------------"
echo ""
echo "To verify secrets are created (without exposing values):"
echo "  kubectl get secrets -n quantumlayer"
echo ""
echo "To check if a specific secret exists:"
echo "  kubectl describe secret llm-secrets -n quantumlayer"
echo ""
echo "NEVER run: kubectl get secret -o yaml (this would expose the values!)"
echo ""

# Check what secrets exist
echo "Current secrets in quantumlayer namespace:"
kubectl get secrets -n quantumlayer --no-headers 2>/dev/null | awk '{print "  - "$1}' || echo "  No secrets found"

echo ""
echo "🔒 Security Best Practices:"
echo "  1. Never type secrets directly in terminal (appears in history)"
echo "  2. Use environment variables or files"
echo "  3. Delete any files containing secrets after use"
echo "  4. Use a secret management tool like HashiCorp Vault in production"
echo "  5. Rotate secrets regularly"
echo "  6. Use IAM roles when possible (for AWS)"