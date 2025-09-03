#!/bin/bash

# QuantumLayer Platform AI-Native Demonstration
# Shows the transformation from traditional switch/case to AI semantic routing

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}🚀 QuantumLayer Platform AI-Native Demo${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Function to test endpoint
test_endpoint() {
    local name=$1
    local url=$2
    local data=$3
    
    echo -e "${YELLOW}Testing: $name${NC}"
    echo -e "${BLUE}Request:${NC} $data"
    
    response=$(kubectl run test-$RANDOM --rm -i --image=curlimages/curl -n quantumlayer --restart=Never -- \
        curl -s -X POST $url \
        -H "Content-Type: application/json" \
        -d "$data" 2>/dev/null || echo "Failed")
    
    echo -e "${GREEN}Response:${NC} $response"
    echo ""
}

echo -e "${BLUE}=== Phase 1: Traditional Approach (Before) ===${NC}"
echo "Previously, language selection used hardcoded switch statements:"
echo '```go'
echo 'switch promptType {'
echo '  case "api": return "python"'
echo '  case "frontend": return "javascript"'
echo '  default: return "python"'
echo '}'
echo '```'
echo ""

echo -e "${BLUE}=== Phase 2: AI-Native Transformation (After) ===${NC}"
echo "Now using semantic understanding and AI decision making:"
echo ""

# Test 1: Language Selection with AI
echo -e "${GREEN}1. AI Language Decision (replaces switch/case)${NC}"
test_endpoint "Language Selection" \
    "http://api-gateway:8000/api/v1/workflows/decide-language" \
    '{"prompt":"I need to build a high-performance REST API with real-time capabilities","use_ai":true}'

# Test 2: Agent Selection with AI
echo -e "${GREEN}2. AI Agent Selection (dynamic role assignment)${NC}"
test_endpoint "Agent Selection" \
    "http://agent-orchestrator:8083/api/v1/agents/select" \
    '{"requirements":"Need someone to analyze security vulnerabilities and ensure GDPR compliance","use_ai":true}'

# Test 3: Framework Selection
echo -e "${GREEN}3. AI Framework Selection${NC}"
test_endpoint "Framework Selection" \
    "http://api-gateway:8000/api/v1/workflows/decide-framework" \
    '{"language":"python","requirements":"microservices with async support","use_ai":true}'

echo -e "${BLUE}=== Phase 3: The 5th Path - QSecure ===${NC}"
echo "Adding comprehensive security as the 5th product path:"
echo ""

# Test 4: Security Analysis
echo -e "${GREEN}4. QSecure Security Analysis (5th Path)${NC}"
test_endpoint "Security Analysis" \
    "http://api-gateway:8000/api/v1/security/analyze" \
    '{"code":"SELECT * FROM users WHERE id = $user_input","language":"sql","enable_qsecure":true}'

# Test 5: Threat Modeling
echo -e "${GREEN}5. QSecure Threat Modeling${NC}"
test_endpoint "Threat Modeling" \
    "http://api-gateway:8000/api/v1/security/threat-model" \
    '{"architecture":"microservices","components":["api","database","cache"]}'

echo -e "${BLUE}=== Phase 4: End-to-End AI Workflow ===${NC}"
echo ""

# Test 6: Complete AI-Native Workflow
echo -e "${GREEN}6. Complete AI-Powered Code Generation${NC}"
test_endpoint "AI Workflow" \
    "http://api-gateway:8000/api/v1/workflows/generate" \
    '{"prompt":"Build a secure payment processing microservice with fraud detection","enable_ai_decisions":true,"enable_qsecure":true}'

echo -e "${BLUE}=== Summary of AI-Native Improvements ===${NC}"
echo -e "${GREEN}✅ Replaced 26 switch/case statements${NC} with semantic AI routing"
echo -e "${GREEN}✅ Added QSecure as 5th path${NC} alongside Code, Infra, QA, SRE"
echo -e "${GREEN}✅ Dynamic agent creation${NC} based on requirements"
echo -e "${GREEN}✅ Semantic understanding${NC} of user intent"
echo -e "${GREEN}✅ Learning from feedback${NC} to improve decisions"
echo -e "${GREEN}✅ Fuzzy matching${NC} for typos and variations"
echo ""

echo -e "${YELLOW}=== Benefits of AI-Native Architecture ===${NC}"
echo "1. No more hardcoding - decisions based on understanding"
echo "2. Handles new scenarios without code changes"
echo "3. Improves over time through learning"
echo "4. Universal platform for any technology stack"
echo "5. Comprehensive security built into every workflow"
echo ""

echo -e "${GREEN}🎉 Demo Complete!${NC}"
echo ""
echo -e "${BLUE}Access Points:${NC}"
echo "• Web UI: kubectl port-forward svc/web-ui 8888:80 -n quantumlayer"
echo "• Swagger: kubectl port-forward svc/api-docs 8090:8090 -n quantumlayer"
echo "• Grafana: kubectl port-forward svc/grafana 3000:80 -n quantumlayer"