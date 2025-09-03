#!/bin/bash

# Deploy AI-Native Components to Kubernetes
# This script deploys the AI components to the QuantumLayer platform

set -e

# Configuration
NAMESPACE="quantumlayer"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Deploying AI-Native Components to Kubernetes${NC}"
echo ""

# Function to deploy a service
deploy_service() {
    local service_name=$1
    local manifest_file=$2
    
    echo -e "${YELLOW}📦 Deploying ${service_name}...${NC}"
    
    kubectl apply -f ${manifest_file}
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ Successfully deployed ${service_name}${NC}"
    else
        echo -e "${RED}❌ Failed to deploy ${service_name}${NC}"
        exit 1
    fi
    
    echo ""
}

# Function to wait for deployment to be ready
wait_for_deployment() {
    local deployment_name=$1
    local timeout=${2:-120}
    
    echo -e "${BLUE}⏳ Waiting for ${deployment_name} to be ready...${NC}"
    
    kubectl wait --for=condition=available --timeout=${timeout}s \
        deployment/${deployment_name} -n ${NAMESPACE}
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ ${deployment_name} is ready${NC}"
    else
        echo -e "${RED}❌ ${deployment_name} failed to become ready${NC}"
        kubectl describe deployment/${deployment_name} -n ${NAMESPACE}
        return 1
    fi
}

# Deploy AI Decision Engine
echo -e "${GREEN}=== Deploying AI Decision Engine ===${NC}"
deploy_service "AI Decision Engine" "infrastructure/kubernetes/ai-decision-engine.yaml"
wait_for_deployment "ai-decision-engine"

# Deploy QSecure Engine
echo -e "${GREEN}=== Deploying QSecure Engine ===${NC}"
deploy_service "QSecure Engine" "infrastructure/kubernetes/qsecure-engine.yaml"
wait_for_deployment "qsecure-engine"

# Deploy Meta-Prompt Engine (if manifest exists)
if [ -f "infrastructure/kubernetes/meta-prompt-engine.yaml" ]; then
    echo -e "${GREEN}=== Deploying Meta-Prompt Engine ===${NC}"
    deploy_service "Meta-Prompt Engine" "infrastructure/kubernetes/meta-prompt-engine.yaml"
    wait_for_deployment "meta-prompt-engine"
else
    echo -e "${YELLOW}⚠️  Meta-Prompt Engine manifest not found, using existing deployment${NC}"
fi

# Deploy Web UI
if [ -f "infrastructure/kubernetes/web-ui.yaml" ]; then
    echo -e "${GREEN}=== Deploying Web UI ===${NC}"
    deploy_service "Web UI" "infrastructure/kubernetes/web-ui.yaml"
    wait_for_deployment "web-ui"
else
    echo -e "${YELLOW}⚠️  Web UI manifest not found${NC}"
fi

# Deploy API Documentation Service
if [ -f "infrastructure/kubernetes/api-docs.yaml" ]; then
    echo -e "${GREEN}=== Deploying API Documentation ===${NC}"
    deploy_service "API Documentation" "infrastructure/kubernetes/api-docs.yaml"
    wait_for_deployment "api-docs"
else
    echo -e "${YELLOW}⚠️  API Docs manifest not found${NC}"
fi

echo ""
echo -e "${GREEN}🎉 AI Components Deployment Complete!${NC}"
echo ""
echo -e "${BLUE}🔍 Checking deployment status...${NC}"
kubectl get deployments -n ${NAMESPACE} | grep -E "(ai-decision|qsecure|meta-prompt|web-ui|api-docs)"

echo ""
echo -e "${BLUE}🌐 Service endpoints:${NC}"
kubectl get services -n ${NAMESPACE} | grep -E "(ai-decision|qsecure|meta-prompt|web-ui|api-docs)"

echo ""
echo -e "${YELLOW}📋 Next Steps:${NC}"
echo "1. Test AI Decision Engine: curl http://ai-decision-engine.${NAMESPACE}:8095/health"
echo "2. Test QSecure Engine: curl http://qsecure-engine.${NAMESPACE}:8096/health"
echo "3. Access Web UI: kubectl port-forward svc/web-ui 8888:80 -n ${NAMESPACE}"
echo "4. Run integration tests: ./test-ai-services.sh"