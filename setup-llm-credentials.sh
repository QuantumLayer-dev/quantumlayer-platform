#!/bin/bash

# Script to configure LLM provider credentials for QuantumLayer

echo "==================================================="
echo "LLM Provider Credentials Setup"
echo "==================================================="
echo ""
echo "This script will help you configure credentials for:"
echo "1. AWS Bedrock (Claude, Llama, etc.)"
echo "2. Azure OpenAI"
echo "3. OpenAI API"
echo "4. Anthropic Claude API"
echo "5. Groq (optional)"
echo ""

# AWS Bedrock Configuration
echo "----------------------------------------"
echo "1. AWS Bedrock Configuration"
echo "----------------------------------------"
read -p "Do you have AWS credentials for Bedrock? (y/n): " HAS_AWS
if [ "$HAS_AWS" = "y" ]; then
    read -p "AWS Access Key ID: " AWS_ACCESS_KEY_ID
    read -s -p "AWS Secret Access Key: " AWS_SECRET_ACCESS_KEY
    echo ""
    read -p "AWS Region (default: us-east-1): " AWS_REGION
    AWS_REGION=${AWS_REGION:-us-east-1}
    read -p "Bedrock Model (default: anthropic.claude-3-sonnet-20240229-v1:0): " AWS_BEDROCK_MODEL
    AWS_BEDROCK_MODEL=${AWS_BEDROCK_MODEL:-anthropic.claude-3-sonnet-20240229-v1:0}
else
    AWS_ACCESS_KEY_ID="not-configured"
    AWS_SECRET_ACCESS_KEY="not-configured"
    AWS_REGION="us-east-1"
    AWS_BEDROCK_MODEL="anthropic.claude-3-sonnet-20240229-v1:0"
fi

# Azure OpenAI Configuration
echo ""
echo "----------------------------------------"
echo "2. Azure OpenAI Configuration"
echo "----------------------------------------"
read -p "Do you have Azure OpenAI credentials? (y/n): " HAS_AZURE
if [ "$HAS_AZURE" = "y" ]; then
    read -p "Azure OpenAI Endpoint (e.g., https://myresource.openai.azure.com): " AZURE_OPENAI_ENDPOINT
    read -s -p "Azure OpenAI API Key: " AZURE_OPENAI_KEY
    echo ""
    read -p "Azure OpenAI Deployment Name: " AZURE_OPENAI_DEPLOYMENT
else
    AZURE_OPENAI_ENDPOINT="https://not-configured.openai.azure.com"
    AZURE_OPENAI_KEY="not-configured"
    AZURE_OPENAI_DEPLOYMENT="not-configured"
fi

# OpenAI Configuration
echo ""
echo "----------------------------------------"
echo "3. OpenAI API Configuration"
echo "----------------------------------------"
read -p "Do you have an OpenAI API key? (y/n): " HAS_OPENAI
if [ "$HAS_OPENAI" = "y" ]; then
    read -s -p "OpenAI API Key (sk-...): " OPENAI_API_KEY
    echo ""
else
    OPENAI_API_KEY="sk-not-configured"
fi

# Anthropic Configuration
echo ""
echo "----------------------------------------"
echo "4. Anthropic Claude API Configuration"
echo "----------------------------------------"
read -p "Do you have an Anthropic API key? (y/n): " HAS_ANTHROPIC
if [ "$HAS_ANTHROPIC" = "y" ]; then
    read -s -p "Anthropic API Key (sk-ant-...): " ANTHROPIC_API_KEY
    echo ""
else
    ANTHROPIC_API_KEY="sk-ant-not-configured"
fi

# Groq Configuration (Optional)
echo ""
echo "----------------------------------------"
echo "5. Groq API Configuration (Optional)"
echo "----------------------------------------"
read -p "Do you have a Groq API key? (y/n): " HAS_GROQ
if [ "$HAS_GROQ" = "y" ]; then
    read -s -p "Groq API Key: " GROQ_API_KEY
    echo ""
else
    GROQ_API_KEY="not-configured"
fi

# Create the secrets
echo ""
echo "Creating Kubernetes secrets..."

# Delete existing secrets if they exist
kubectl delete secret llm-secrets -n quantumlayer 2>/dev/null || true
kubectl delete secret aws-bedrock-secrets -n quantumlayer 2>/dev/null || true
kubectl delete secret azure-openai-secrets -n quantumlayer 2>/dev/null || true

# Create LLM secrets
kubectl create secret generic llm-secrets \
  --from-literal=OPENAI_API_KEY="$OPENAI_API_KEY" \
  --from-literal=ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY" \
  --from-literal=GROQ_API_KEY="$GROQ_API_KEY" \
  -n quantumlayer

# Create AWS Bedrock secrets
kubectl create secret generic aws-bedrock-secrets \
  --from-literal=AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_ID" \
  --from-literal=AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY" \
  --from-literal=AWS_BEDROCK_REGION="$AWS_REGION" \
  --from-literal=AWS_BEDROCK_MODEL="$AWS_BEDROCK_MODEL" \
  -n quantumlayer

# Create Azure OpenAI secrets
kubectl create secret generic azure-openai-secrets \
  --from-literal=AZURE_OPENAI_ENDPOINT="$AZURE_OPENAI_ENDPOINT" \
  --from-literal=AZURE_OPENAI_KEY="$AZURE_OPENAI_KEY" \
  --from-literal=AZURE_OPENAI_DEPLOYMENT="$AZURE_OPENAI_DEPLOYMENT" \
  -n quantumlayer

echo ""
echo "✅ Secrets created successfully!"
echo ""
echo "Configured providers:"
[ "$HAS_AWS" = "y" ] && echo "  ✅ AWS Bedrock (Region: $AWS_REGION)"
[ "$HAS_AZURE" = "y" ] && echo "  ✅ Azure OpenAI"
[ "$HAS_OPENAI" = "y" ] && echo "  ✅ OpenAI"
[ "$HAS_ANTHROPIC" = "y" ] && echo "  ✅ Anthropic Claude"
[ "$HAS_GROQ" = "y" ] && echo "  ✅ Groq"
echo ""
echo "You can update these secrets anytime by running this script again."
echo ""
echo "To test the LLM Router after deployment:"
echo "  curl http://<node-ip>:30881/api/v1/providers"
echo "  curl http://<node-ip>:30881/api/v1/chat/completions \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"messages\":[{\"role\":\"user\",\"content\":\"Hello\"}]}'"