#!/bin/bash
set -e

# Version and registry configuration
VERSION="2.5.0"
REGISTRY="ghcr.io/quantumlayer-dev"
DATE=$(date +%Y%m%d-%H%M%S)

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     QUANTUMLAYER PLATFORM - BUILD ALL v${VERSION}           ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Track build results
BUILD_SUCCESS=()
BUILD_FAILED=()

# Function to build and push image
build_image() {
    local name=$1
    local path=$2
    local full_name="${REGISTRY}/${name}:${VERSION}"
    local latest_name="${REGISTRY}/${name}:latest"
    
    echo -e "${YELLOW}Building ${name}...${NC}"
    
    if docker build -t "$full_name" -t "$latest_name" "$path" 2>/dev/null; then
        echo -e "${GREEN}✅ Built ${name}${NC}"
        BUILD_SUCCESS+=("$name")
        
        # Push to registry if logged in
        if docker push "$full_name" 2>/dev/null && docker push "$latest_name" 2>/dev/null; then
            echo -e "${GREEN}✅ Pushed ${name}${NC}"
        else
            echo -e "${YELLOW}⚠️  Could not push ${name} (registry access needed)${NC}"
        fi
    else
        echo -e "${RED}❌ Failed to build ${name}${NC}"
        BUILD_FAILED+=("$name")
    fi
    echo ""
}

# Packages to build
echo -e "${BLUE}=== Building Package Services ===${NC}"
echo ""

# Core packages
build_image "api-gateway" "./packages/api-gateway"
build_image "llm-router" "./packages/llm-router"
build_image "agent-orchestrator" "./packages/agent-orchestrator"
build_image "parser" "./packages/parser"
build_image "meta-prompt-engine" "./packages/meta-prompt-engine"
build_image "sandbox-executor" "./packages/sandbox-executor"
build_image "capsule-builder" "./packages/capsule-builder"
build_image "quantum-capsule" "./packages/quantum-capsule"
build_image "quantum-drops" "./packages/quantum-drops"
build_image "mcp-gateway" "./packages/mcp-gateway"
build_image "workflows" "./packages/workflows"

# QInfra packages
build_image "qinfra" "./packages/qinfra"
build_image "qsecure" "./packages/qsecure"
build_image "qtest" "./packages/qtest"
build_image "ai-decision-engine" "./packages/ai-decision-engine"

# Services to build
echo -e "${BLUE}=== Building Services ===${NC}"
echo ""

build_image "workflow-api" "./services/workflow-api"
build_image "deployment-manager" "./services/deployment-manager"
build_image "preview-service" "./services/preview-service"
build_image "image-registry" "./services/image-registry"
build_image "qinfra-dashboard" "./services/qinfra-dashboard"
build_image "qinfra-ai" "./services/qinfra-ai"
build_image "cve-tracker" "./services/cve-tracker"
build_image "trivy-scanner" "./services/trivy-scanner"
build_image "cosign-signer" "./services/cosign-signer"
build_image "packer-builder" "./services/packer-builder"
build_image "web-ui" "./services/web-ui"

# Additional infrastructure services
echo -e "${BLUE}=== Building Infrastructure Services ===${NC}"
echo ""

# Check if we have infrastructure services
if [ -d "./services/nats" ]; then
    build_image "nats-server" "./services/nats"
fi

# Build summary
echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                    BUILD SUMMARY                              ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

echo -e "${GREEN}Successfully Built: ${#BUILD_SUCCESS[@]} images${NC}"
for img in "${BUILD_SUCCESS[@]}"; do
    echo "  ✅ $img"
done

if [ ${#BUILD_FAILED[@]} -gt 0 ]; then
    echo ""
    echo -e "${RED}Failed Builds: ${#BUILD_FAILED[@]} images${NC}"
    for img in "${BUILD_FAILED[@]}"; do
        echo "  ❌ $img"
    done
fi

# Generate deployment manifest
echo ""
echo -e "${BLUE}=== Generating Version Manifest ===${NC}"

cat > version-manifest-${VERSION}.yaml << EOF
# QuantumLayer Platform Version Manifest
# Version: ${VERSION}
# Built: ${DATE}

apiVersion: v1
kind: ConfigMap
metadata:
  name: platform-version
  namespace: quantumlayer
data:
  version: "${VERSION}"
  build_date: "${DATE}"
  images: |
EOF

for img in "${BUILD_SUCCESS[@]}"; do
    echo "    - ${REGISTRY}/${img}:${VERSION}" >> version-manifest-${VERSION}.yaml
done

echo -e "${GREEN}✅ Version manifest saved to version-manifest-${VERSION}.yaml${NC}"

# Create deployment script
cat > deploy-v${VERSION}.sh << 'DEPLOY_SCRIPT'
#!/bin/bash
set -e

VERSION="2.5.0"

echo "Deploying QuantumLayer Platform v${VERSION}..."

# Update image tags in all deployments
for ns in quantumlayer temporal security-services; do
    echo "Updating namespace: $ns"
    kubectl get deployments -n $ns -o name | while read deploy; do
        kubectl set image $deploy \*=\*:${VERSION} -n $ns --record || true
    done
done

echo "Deployment of v${VERSION} initiated. Check pod status with:"
echo "kubectl get pods --all-namespaces | grep -v Running"
DEPLOY_SCRIPT

chmod +x deploy-v${VERSION}.sh

echo -e "${GREEN}✅ Deployment script saved to deploy-v${VERSION}.sh${NC}"

# Total statistics
echo ""
echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                    FINAL STATISTICS                           ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo "Version:        ${VERSION}"
echo "Build Date:     ${DATE}"
echo "Total Images:   $((${#BUILD_SUCCESS[@]} + ${#BUILD_FAILED[@]}))"
echo "Successful:     ${#BUILD_SUCCESS[@]}"
echo "Failed:         ${#BUILD_FAILED[@]}"
echo ""

if [ ${#BUILD_FAILED[@]} -eq 0 ]; then
    echo -e "${GREEN}🎉 All images built successfully!${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Run ./deploy-v${VERSION}.sh to deploy"
    echo "2. Run ./test-complete-integration.sh to test"
    echo "3. Run ./demo-all-services.sh for full demo"
    exit 0
else
    echo -e "${YELLOW}⚠️  Some builds failed. Review and fix before deploying.${NC}"
    exit 1
fi