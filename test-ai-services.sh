#!/bin/bash

# Test AI-Native Services
# This script tests the deployed AI components

set -e

# Configuration
NAMESPACE="quantumlayer"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${GREEN}🧪 Testing AI-Native Components${NC}"
echo ""

# Function to test a service endpoint
test_endpoint() {
    local service_name=$1
    local endpoint=$2
    local data=$3
    local method=${4:-GET}
    
    echo -e "${YELLOW}🔬 Testing ${service_name}...${NC}"
    
    if [ "$method" = "POST" ]; then
        response=$(kubectl run test-${service_name}-$RANDOM --rm -i --image=curlimages/curl -n ${NAMESPACE} --restart=Never -- \
            curl -s -X POST ${endpoint} \
            -H "Content-Type: application/json" \
            -d "${data}" 2>/dev/null || echo "Failed")
    else
        response=$(kubectl run test-${service_name}-$RANDOM --rm -i --image=curlimages/curl -n ${NAMESPACE} --restart=Never -- \
            curl -s ${endpoint} 2>/dev/null || echo "Failed")
    fi
    
    if [[ "$response" != *"Failed"* ]] && [[ "$response" != *"error"* ]]; then
        echo -e "${GREEN}✅ ${service_name}: SUCCESS${NC}"
        echo "Response: ${response:0:100}..."
    else
        echo -e "${RED}❌ ${service_name}: FAILED${NC}"
        echo "Response: $response"
    fi
    echo ""
}

echo -e "${BLUE}=== Testing Core Services ===${NC}"
echo ""

# Test AI Decision Engine
test_endpoint "AI Decision Engine Health" \
    "http://ai-decision-engine:8095/health"

test_endpoint "AI Language Selection" \
    "http://ai-decision-engine:8095/api/v1/decide" \
    '{"category":"language_selection","input":"Build a REST API with user authentication"}' \
    "POST"

test_endpoint "AI Agent Selection" \
    "http://ai-decision-engine:8095/api/v1/decide" \
    '{"category":"agent_selection","input":"Need to analyze security vulnerabilities"}' \
    "POST"

# Test QSecure Engine
test_endpoint "QSecure Engine Health" \
    "http://qsecure-engine:8096/health"

test_endpoint "QSecure Security Analysis" \
    "http://qsecure-engine:8096/api/v1/analyze" \
    '{"code":"SELECT * FROM users WHERE id = " + user_input","language":"python"}' \
    "POST"

# Test existing services with AI integration
test_endpoint "Workflow with AI Decision" \
    "http://api-gateway:8000/api/v1/workflows/generate" \
    '{"prompt":"Create a microservice for order processing","use_ai_decision":true}' \
    "POST"

test_endpoint "Agent with AI Factory" \
    "http://agent-orchestrator:8083/api/v1/agents/spawn" \
    '{"requirements":"Need a specialist for database optimization","use_ai_factory":true}' \
    "POST"

echo -e "${BLUE}=== Testing Integration Points ===${NC}"
echo ""

# Test AI-powered workflow
test_endpoint "End-to-End AI Workflow" \
    "http://api-gateway:8000/api/v1/workflows/ai-native" \
    '{"requirements":"Build a secure payment processing system","enable_qsecure":true}' \
    "POST"

echo ""
echo -e "${GREEN}🎉 Testing Complete!${NC}"
echo ""
echo -e "${YELLOW}📋 Summary:${NC}"
echo "- AI Decision Engine: Semantic routing instead of switch statements"
echo "- QSecure Engine: 5th product path for security"
echo "- AI Agent Factory: Dynamic agent creation"
echo "- Meta-Prompt Enhancement: Optimized prompt engineering"
echo ""
echo -e "${BLUE}🌐 Access Points:${NC}"
echo "Web UI: kubectl port-forward svc/web-ui 8888:80 -n ${NAMESPACE}"
echo "Swagger API: kubectl port-forward svc/api-docs 8090:8090 -n ${NAMESPACE}"