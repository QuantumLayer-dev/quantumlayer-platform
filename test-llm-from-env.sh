#!/bin/bash

# Test LLM credentials loaded from .env file

echo "==================================================="
echo "🧪 Testing LLM Credentials from .env"
echo "==================================================="
echo ""

# Check if .env file exists
if [ ! -f .env ]; then
    echo "❌ .env file not found!"
    echo "Please create one: cp .env.template .env"
    exit 1
fi

# Source the .env file
echo "Loading credentials from .env file..."
set -a
source .env
set +a

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test AWS Bedrock
if [ ! -z "$AWS_ACCESS_KEY_ID" ] && [ "$AWS_ACCESS_KEY_ID" != "your-aws-access-key-id" ]; then
    echo ""
    echo "Testing AWS Bedrock..."
    if command -v aws &> /dev/null; then
        aws sts get-caller-identity &>/dev/null
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}✅ AWS credentials are valid${NC}"
            
            # Quick Bedrock test
            aws bedrock list-foundation-models --region ${AWS_BEDROCK_REGION:-us-east-1} --max-results 1 &>/dev/null
            if [ $? -eq 0 ]; then
                echo -e "${GREEN}✅ AWS Bedrock is accessible${NC}"
            else
                echo -e "${YELLOW}⚠️  Cannot access Bedrock (check permissions)${NC}"
            fi
        else
            echo -e "${RED}❌ Invalid AWS credentials${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  AWS CLI not installed (skipping test)${NC}"
    fi
fi

# Test Azure OpenAI
if [ ! -z "$AZURE_OPENAI_KEY" ] && [ "$AZURE_OPENAI_KEY" != "your-azure-openai-key" ]; then
    echo ""
    echo "Testing Azure OpenAI..."
    echo "Endpoint: $AZURE_OPENAI_ENDPOINT"
    
    RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" \
        "${AZURE_OPENAI_ENDPOINT}/openai/deployments?api-version=2023-05-15" \
        -H "api-key: $AZURE_OPENAI_KEY")
    
    if [ "$RESPONSE" = "200" ]; then
        echo -e "${GREEN}✅ Azure OpenAI credentials are valid${NC}"
        
        # List deployments
        echo "Available deployments:"
        curl -s "${AZURE_OPENAI_ENDPOINT}/openai/deployments?api-version=2023-05-15" \
            -H "api-key: $AZURE_OPENAI_KEY" | jq -r '.data[].id' 2>/dev/null | head -3 | sed 's/^/  - /'
    else
        echo -e "${RED}❌ Azure OpenAI test failed (HTTP $RESPONSE)${NC}"
    fi
fi

# Test OpenAI
if [ ! -z "$OPENAI_API_KEY" ] && [ "$OPENAI_API_KEY" != "sk-your-openai-key" ]; then
    echo ""
    echo "Testing OpenAI..."
    RESPONSE=$(curl -s https://api.openai.com/v1/models \
        -H "Authorization: Bearer $OPENAI_API_KEY")
    
    if echo "$RESPONSE" | grep -q "\"object\": \"list\""; then
        echo -e "${GREEN}✅ OpenAI API key is valid${NC}"
    else
        echo -e "${RED}❌ Invalid OpenAI API key${NC}"
    fi
fi

# Test Anthropic
if [ ! -z "$ANTHROPIC_API_KEY" ] && [ "$ANTHROPIC_API_KEY" != "sk-ant-your-anthropic-key" ]; then
    echo ""
    echo "Testing Anthropic..."
    RESPONSE=$(curl -s -X POST https://api.anthropic.com/v1/messages \
        -H "x-api-key: $ANTHROPIC_API_KEY" \
        -H "anthropic-version: 2023-06-01" \
        -H "content-type: application/json" \
        -d '{
            "model": "claude-3-haiku-20240307",
            "max_tokens": 10,
            "messages": [{"role": "user", "content": "Hi"}]
        }')
    
    if echo "$RESPONSE" | grep -q "\"type\": \"message\""; then
        echo -e "${GREEN}✅ Anthropic API key is valid${NC}"
    else
        echo -e "${RED}❌ Invalid Anthropic API key${NC}"
    fi
fi

echo ""
echo "==================================================="
echo "Test Complete"
echo "==================================================="
echo ""
echo "If all tests passed, create Kubernetes secrets with:"
echo "  ./setup-llm-from-env.sh"