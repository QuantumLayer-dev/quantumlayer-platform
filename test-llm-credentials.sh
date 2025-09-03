#!/bin/bash

# Script to test LLM provider credentials before adding to Kubernetes

echo "==================================================="
echo "🧪 LLM Provider Credentials Testing"
echo "==================================================="
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test AWS Bedrock
test_aws_bedrock() {
    echo "----------------------------------------"
    echo "Testing AWS Bedrock..."
    echo "----------------------------------------"
    
    if [ -z "$AWS_ACCESS_KEY_ID" ] || [ -z "$AWS_SECRET_ACCESS_KEY" ]; then
        echo -e "${YELLOW}⚠️  AWS credentials not set in environment${NC}"
        return 1
    fi
    
    # Test with AWS CLI if available
    if command -v aws &> /dev/null; then
        echo "Testing AWS credentials with CLI..."
        aws sts get-caller-identity &>/dev/null
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}✅ AWS credentials are valid${NC}"
            
            # Test Bedrock access
            echo "Testing Bedrock access..."
            aws bedrock list-foundation-models --region ${AWS_BEDROCK_REGION:-us-east-1} --max-results 1 &>/dev/null
            if [ $? -eq 0 ]; then
                echo -e "${GREEN}✅ AWS Bedrock access confirmed${NC}"
                
                # Test Claude model specifically
                echo "Testing Claude model access..."
                cat > /tmp/bedrock-test.json << 'EOF'
{
    "prompt": "\n\nHuman: Hello\n\nAssistant:",
    "max_tokens_to_sample": 10,
    "temperature": 0
}
EOF
                
                aws bedrock-runtime invoke-model \
                    --model-id "anthropic.claude-instant-v1" \
                    --content-type "application/json" \
                    --accept "application/json" \
                    --body file:///tmp/bedrock-test.json \
                    --region ${AWS_BEDROCK_REGION:-us-east-1} \
                    /tmp/bedrock-response.json 2>/dev/null
                
                if [ $? -eq 0 ]; then
                    echo -e "${GREEN}✅ Claude model on Bedrock is accessible${NC}"
                    echo "Response: $(cat /tmp/bedrock-response.json | jq -r .completion 2>/dev/null | head -c 50)..."
                    rm -f /tmp/bedrock-response.json /tmp/bedrock-test.json
                else
                    echo -e "${YELLOW}⚠️  Cannot invoke Claude model. Check if you have access to this model in your region${NC}"
                fi
            else
                echo -e "${RED}❌ Cannot access AWS Bedrock. Check your permissions${NC}"
            fi
        else
            echo -e "${RED}❌ Invalid AWS credentials${NC}"
        fi
    else
        echo "AWS CLI not found. Testing with curl..."
        # Basic AWS v4 signature test would be complex here
        echo -e "${YELLOW}⚠️  Install AWS CLI for complete testing: pip install awscli${NC}"
    fi
}

# Test Azure OpenAI
test_azure_openai() {
    echo ""
    echo "----------------------------------------"
    echo "Testing Azure OpenAI..."
    echo "----------------------------------------"
    
    if [ -z "$AZURE_OPENAI_KEY" ] || [ -z "$AZURE_OPENAI_ENDPOINT" ]; then
        echo -e "${YELLOW}⚠️  Azure OpenAI credentials not set in environment${NC}"
        return 1
    fi
    
    echo "Testing Azure OpenAI API..."
    echo "Endpoint: $AZURE_OPENAI_ENDPOINT"
    
    # Test listing deployments
    RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" \
        "${AZURE_OPENAI_ENDPOINT}/openai/deployments?api-version=2023-05-15" \
        -H "api-key: $AZURE_OPENAI_KEY")
    
    if [ "$RESPONSE" = "200" ]; then
        echo -e "${GREEN}✅ Azure OpenAI credentials are valid${NC}"
        
        # List available deployments
        echo "Available deployments:"
        curl -s "${AZURE_OPENAI_ENDPOINT}/openai/deployments?api-version=2023-05-15" \
            -H "api-key: $AZURE_OPENAI_KEY" | jq -r '.data[].id' 2>/dev/null | head -5 | sed 's/^/  - /'
        
        # Test chat completion if deployment is set
        if [ ! -z "$AZURE_OPENAI_DEPLOYMENT" ]; then
            echo "Testing deployment: $AZURE_OPENAI_DEPLOYMENT"
            
            CHAT_RESPONSE=$(curl -s -X POST \
                "${AZURE_OPENAI_ENDPOINT}/openai/deployments/${AZURE_OPENAI_DEPLOYMENT}/chat/completions?api-version=2023-05-15" \
                -H "Content-Type: application/json" \
                -H "api-key: $AZURE_OPENAI_KEY" \
                -d '{
                    "messages": [{"role": "user", "content": "Say hello"}],
                    "max_tokens": 10
                }')
            
            if echo "$CHAT_RESPONSE" | grep -q "choices"; then
                echo -e "${GREEN}✅ Azure OpenAI deployment '$AZURE_OPENAI_DEPLOYMENT' is working${NC}"
                echo "Response: $(echo "$CHAT_RESPONSE" | jq -r '.choices[0].message.content' 2>/dev/null | head -c 50)..."
            else
                echo -e "${RED}❌ Deployment '$AZURE_OPENAI_DEPLOYMENT' not accessible${NC}"
                echo "Error: $(echo "$CHAT_RESPONSE" | jq -r '.error.message' 2>/dev/null)"
            fi
        fi
    elif [ "$RESPONSE" = "401" ]; then
        echo -e "${RED}❌ Invalid Azure OpenAI API key${NC}"
    elif [ "$RESPONSE" = "404" ]; then
        echo -e "${RED}❌ Invalid Azure OpenAI endpoint URL${NC}"
    else
        echo -e "${RED}❌ Azure OpenAI test failed (HTTP $RESPONSE)${NC}"
    fi
}

# Test OpenAI
test_openai() {
    echo ""
    echo "----------------------------------------"
    echo "Testing OpenAI..."
    echo "----------------------------------------"
    
    if [ -z "$OPENAI_API_KEY" ] || [ "$OPENAI_API_KEY" = "not-configured" ]; then
        echo -e "${YELLOW}⚠️  OpenAI API key not set${NC}"
        return 1
    fi
    
    echo "Testing OpenAI API..."
    RESPONSE=$(curl -s https://api.openai.com/v1/models \
        -H "Authorization: Bearer $OPENAI_API_KEY")
    
    if echo "$RESPONSE" | grep -q "\"object\": \"list\""; then
        echo -e "${GREEN}✅ OpenAI API key is valid${NC}"
        echo "Available models:"
        echo "$RESPONSE" | jq -r '.data[].id' 2>/dev/null | grep -E "gpt|dall-e|whisper" | head -5 | sed 's/^/  - /'
    else
        ERROR=$(echo "$RESPONSE" | jq -r '.error.message' 2>/dev/null)
        echo -e "${RED}❌ Invalid OpenAI API key${NC}"
        [ ! -z "$ERROR" ] && echo "Error: $ERROR"
    fi
}

# Test Anthropic
test_anthropic() {
    echo ""
    echo "----------------------------------------"
    echo "Testing Anthropic..."
    echo "----------------------------------------"
    
    if [ -z "$ANTHROPIC_API_KEY" ] || [ "$ANTHROPIC_API_KEY" = "not-configured" ]; then
        echo -e "${YELLOW}⚠️  Anthropic API key not set${NC}"
        return 1
    fi
    
    echo "Testing Anthropic API..."
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
        echo "Response: $(echo "$RESPONSE" | jq -r '.content[0].text' 2>/dev/null | head -c 50)..."
    else
        ERROR=$(echo "$RESPONSE" | jq -r '.error.message' 2>/dev/null)
        echo -e "${RED}❌ Invalid Anthropic API key${NC}"
        [ ! -z "$ERROR" ] && echo "Error: $ERROR"
    fi
}

# Main execution
echo "Make sure you've exported your credentials as environment variables:"
echo "  export AWS_ACCESS_KEY_ID='your-key'"
echo "  export AWS_SECRET_ACCESS_KEY='your-secret'"
echo "  export AZURE_OPENAI_KEY='your-key'"
echo "  export AZURE_OPENAI_ENDPOINT='https://myazurellm.openai.azure.com/'"
echo "  export AZURE_OPENAI_DEPLOYMENT='your-deployment-name'"
echo "  export OPENAI_API_KEY='sk-...'"
echo "  export ANTHROPIC_API_KEY='sk-ant-...'"
echo ""

# Run tests
test_aws_bedrock
test_azure_openai
test_openai
test_anthropic

echo ""
echo "==================================================="
echo "Test Summary"
echo "==================================================="
echo ""

# Summary
[ ! -z "$AWS_ACCESS_KEY_ID" ] && echo -n "AWS Bedrock: " && \
    (aws sts get-caller-identity &>/dev/null && echo -e "${GREEN}✅ Ready${NC}" || echo -e "${RED}❌ Failed${NC}")

[ ! -z "$AZURE_OPENAI_KEY" ] && echo -n "Azure OpenAI: " && \
    (curl -s -o /dev/null -w "%{http_code}" "${AZURE_OPENAI_ENDPOINT}/openai/deployments?api-version=2023-05-15" -H "api-key: $AZURE_OPENAI_KEY" | grep -q "200" && \
    echo -e "${GREEN}✅ Ready${NC}" || echo -e "${RED}❌ Failed${NC}")

[ ! -z "$OPENAI_API_KEY" ] && [ "$OPENAI_API_KEY" != "not-configured" ] && echo -n "OpenAI: " && \
    (curl -s https://api.openai.com/v1/models -H "Authorization: Bearer $OPENAI_API_KEY" | grep -q "object" && \
    echo -e "${GREEN}✅ Ready${NC}" || echo -e "${RED}❌ Failed${NC}")

[ ! -z "$ANTHROPIC_API_KEY" ] && [ "$ANTHROPIC_API_KEY" != "not-configured" ] && echo -n "Anthropic: " && \
    echo -e "${YELLOW}⚠️  Test individually${NC}"

echo ""
echo "Once all tests pass, you can create the Kubernetes secrets with:"
echo "  ./setup-llm-credentials-secure.sh"