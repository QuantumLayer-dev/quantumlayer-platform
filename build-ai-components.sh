#!/bin/bash

# Build and Deploy AI-Native Components for QuantumLayer Platform
# This script builds Docker images for the new AI components and deploys them to Kubernetes

set -e

# Configuration
REGISTRY="ghcr.io/quantumlayer-dev"
VERSION="ai-native-v1.0.0"
NAMESPACE="quantumlayer"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Building AI-Native Components for QuantumLayer Platform${NC}"
echo -e "${YELLOW}Version: ${VERSION}${NC}"
echo ""

# Function to build and push Docker image
build_and_push() {
    local service_name=$1
    local dockerfile_path=$2
    local context_path=$3
    
    echo -e "${YELLOW}📦 Building ${service_name}...${NC}"
    
    # Build the Docker image
    docker build -f ${dockerfile_path} -t ${REGISTRY}/${service_name}:${VERSION} ${context_path}
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ Successfully built ${service_name}${NC}"
        
        # Push to registry (uncomment when ready)
        # echo -e "${YELLOW}📤 Pushing ${service_name} to registry...${NC}"
        # docker push ${REGISTRY}/${service_name}:${VERSION}
        # echo -e "${GREEN}✅ Successfully pushed ${service_name}${NC}"
    else
        echo -e "${RED}❌ Failed to build ${service_name}${NC}"
        exit 1
    fi
    
    echo ""
}

# Build AI Decision Engine
echo -e "${GREEN}=== Building AI Decision Engine ===${NC}"
build_and_push "ai-decision-engine" "packages/ai-decision-engine/Dockerfile" "packages/ai-decision-engine"

# Build QSecure Engine
echo -e "${GREEN}=== Building QSecure Engine ===${NC}"
build_and_push "qsecure-engine" "packages/qsecure/Dockerfile" "packages/qsecure"

# Build updated Agent Orchestrator with AI Factory
echo -e "${GREEN}=== Building Agent Orchestrator with AI Factory ===${NC}"
build_and_push "agent-orchestrator" "services/agent-orchestrator/Dockerfile" "services/agent-orchestrator"

# Build Meta-Prompt Engine
echo -e "${GREEN}=== Building Meta-Prompt Engine ===${NC}"
build_and_push "meta-prompt-engine" "services/meta-prompt-engine/Dockerfile" "services/meta-prompt-engine"

# Build Parser Service
echo -e "${GREEN}=== Building Parser Service ===${NC}"
build_and_push "parser" "services/parser/Dockerfile" "services/parser"

# Build API Docs Service with Swagger
echo -e "${GREEN}=== Building API Documentation Service ===${NC}"
build_and_push "api-docs" "services/api-docs/Dockerfile" "services/api-docs"

# Build Web UI
echo -e "${GREEN}=== Building Web UI ===${NC}"
build_and_push "web-ui" "services/web-ui/Dockerfile" "services/web-ui"

echo -e "${GREEN}🎉 All AI components built successfully!${NC}"
echo ""
echo -e "${YELLOW}📋 Next Steps:${NC}"
echo "1. Push images to registry: Uncomment the docker push commands in this script"
echo "2. Deploy to Kubernetes: Run ./deploy-ai-components.sh"
echo "3. Test the services: Run ./test-ai-services.sh"