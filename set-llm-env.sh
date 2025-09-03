#!/bin/bash

# Secure script to set LLM credentials in current session
# This uses read -s to avoid exposing keys in terminal history

echo "==================================================="
echo "🔐 Secure LLM Credentials Setup"
echo "==================================================="
echo ""
echo "This will set environment variables in the CURRENT session."
echo "Your input will be hidden for security."
echo ""

# AWS Credentials
echo "----------------------------------------"
echo "AWS Bedrock Setup"
echo "----------------------------------------"
read -p "Do you have AWS credentials? (y/n): " HAS_AWS
if [ "$HAS_AWS" = "y" ]; then
    read -p "AWS Access Key ID: " AWS_ACCESS_KEY_ID
    read -s -p "AWS Secret Access Key: " AWS_SECRET_ACCESS_KEY
    echo ""
    read -p "AWS Region (default: us-east-1): " AWS_REGION
    AWS_REGION=${AWS_REGION:-us-east-1}
    
    export AWS_ACCESS_KEY_ID
    export AWS_SECRET_ACCESS_KEY
    export AWS_BEDROCK_REGION="$AWS_REGION"
    echo "✅ AWS credentials set"
fi

# Azure OpenAI
echo ""
echo "----------------------------------------"
echo "Azure OpenAI Setup"
echo "----------------------------------------"
read -p "Do you have Azure OpenAI credentials? (y/n): " HAS_AZURE
if [ "$HAS_AZURE" = "y" ]; then
    echo "Endpoint: https://myazurellm.openai.azure.com/"
    export AZURE_OPENAI_ENDPOINT="https://myazurellm.openai.azure.com/"
    
    read -s -p "Azure OpenAI API Key: " AZURE_OPENAI_KEY
    echo ""
    read -p "Deployment name (e.g., gpt-4, gpt-35-turbo): " AZURE_OPENAI_DEPLOYMENT
    
    export AZURE_OPENAI_KEY
    export AZURE_OPENAI_DEPLOYMENT
    echo "✅ Azure OpenAI credentials set"
fi

# OpenAI
echo ""
echo "----------------------------------------"
echo "OpenAI Setup (Optional)"
echo "----------------------------------------"
read -p "Do you have an OpenAI API key? (y/n): " HAS_OPENAI
if [ "$HAS_OPENAI" = "y" ]; then
    read -s -p "OpenAI API Key (sk-...): " OPENAI_API_KEY
    echo ""
    export OPENAI_API_KEY
    echo "✅ OpenAI credentials set"
fi

# Anthropic
echo ""
echo "----------------------------------------"
echo "Anthropic Setup (Optional)"
echo "----------------------------------------"
read -p "Do you have an Anthropic API key? (y/n): " HAS_ANTHROPIC
if [ "$HAS_ANTHROPIC" = "y" ]; then
    read -s -p "Anthropic API Key (sk-ant-...): " ANTHROPIC_API_KEY
    echo ""
    export ANTHROPIC_API_KEY
    echo "✅ Anthropic credentials set"
fi

echo ""
echo "==================================================="
echo "Environment variables are now set in this session!"
echo "==================================================="
echo ""
echo "Next steps:"
echo "1. Test credentials: ./test-llm-credentials.sh"
echo "2. If tests pass, create K8s secrets: ./setup-llm-credentials-secure.sh"
echo ""
echo "Note: These variables are only set in THIS terminal session."