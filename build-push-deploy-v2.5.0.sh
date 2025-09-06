#!/bin/bash
set -e

# Version and registry configuration
VERSION="2.5.0"
REGISTRY="ghcr.io/quantumlayer-dev"
DATE=$(date +%Y%m%d-%H%M%S)
GITHUB_TOKEN="${GITHUB_TOKEN:-}"
GITHUB_USER="${GITHUB_USER:-quantumlayer-dev}"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   QUANTUMLAYER - BUILD, PUSH & DEPLOY v${VERSION}           ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if we're logged in to GHCR
echo -e "${BLUE}=== Step 1: GitHub Container Registry Login ===${NC}"
echo ""

if [ -z "$GITHUB_TOKEN" ]; then
    echo -e "${YELLOW}GITHUB_TOKEN not set. Checking if already logged in...${NC}"
    if ! docker pull ghcr.io/quantumlayer-dev/workflow-api:latest 2>/dev/null; then
        echo -e "${RED}❌ Not logged in to GHCR and no token provided${NC}"
        echo ""
        echo "Please set GITHUB_TOKEN environment variable:"
        echo "  export GITHUB_TOKEN=your_github_personal_access_token"
        echo ""
        echo "Or login manually:"
        echo "  echo \$GITHUB_TOKEN | docker login ghcr.io -u $GITHUB_USER --password-stdin"
        exit 1
    else
        echo -e "${GREEN}✅ Already logged in to GHCR${NC}"
    fi
else
    echo "Logging in to GitHub Container Registry..."
    echo "$GITHUB_TOKEN" | docker login ghcr.io -u "$GITHUB_USER" --password-stdin
    echo -e "${GREEN}✅ Logged in to GHCR${NC}"
fi

echo ""
echo -e "${BLUE}=== Step 2: Build, Tag, and Push Images ===${NC}"
echo ""

# Track results
BUILD_SUCCESS=()
BUILD_FAILED=()
PUSH_SUCCESS=()
PUSH_FAILED=()

# Function to build, tag and push image
build_and_push() {
    local name=$1
    local path=$2
    local full_name="${REGISTRY}/${name}:${VERSION}"
    local latest_name="${REGISTRY}/${name}:latest"
    
    echo -e "${YELLOW}Processing ${name}...${NC}"
    
    # Check if Dockerfile exists
    if [ ! -f "$path/Dockerfile" ]; then
        echo -e "${YELLOW}  ⚠️  No Dockerfile found in $path, skipping${NC}"
        return
    fi
    
    # Build
    echo "  Building..."
    if docker build -t "$full_name" -t "$latest_name" "$path" 2>/dev/null; then
        echo -e "${GREEN}  ✅ Built${NC}"
        BUILD_SUCCESS+=("$name")
        
        # Push versioned tag
        echo "  Pushing version ${VERSION}..."
        if docker push "$full_name" 2>/dev/null; then
            echo -e "${GREEN}  ✅ Pushed version tag${NC}"
            
            # Push latest tag
            echo "  Pushing latest tag..."
            if docker push "$latest_name" 2>/dev/null; then
                echo -e "${GREEN}  ✅ Pushed latest tag${NC}"
                PUSH_SUCCESS+=("$name")
            else
                echo -e "${RED}  ❌ Failed to push latest tag${NC}"
                PUSH_FAILED+=("$name")
            fi
        else
            echo -e "${RED}  ❌ Failed to push version tag${NC}"
            PUSH_FAILED+=("$name")
        fi
    else
        echo -e "${RED}  ❌ Build failed${NC}"
        BUILD_FAILED+=("$name")
    fi
    echo ""
}

# Core services that must be built
CORE_SERVICES=(
    "workflow-api:./services/workflow-api"
    "llm-router:./packages/llm-router"
    "agent-orchestrator:./packages/agent-orchestrator"
    "parser:./packages/parser"
    "meta-prompt-engine:./packages/meta-prompt-engine"
    "sandbox-executor:./packages/sandbox-executor"
    "capsule-builder:./packages/capsule-builder"
    "deployment-manager:./services/deployment-manager"
    "preview-service:./services/preview-service"
    "image-registry:./services/image-registry"
    "cve-tracker:./services/cve-tracker"
    "qinfra-ai:./services/qinfra-ai"
    "workflows:./packages/workflows"
)

# Additional services
ADDITIONAL_SERVICES=(
    "api-gateway:./packages/api-gateway"
    "quantum-capsule:./packages/quantum-capsule"
    "quantum-drops:./packages/quantum-drops"
    "mcp-gateway:./packages/mcp-gateway"
    "qtest:./packages/qtest"
    "qinfra:./packages/qinfra"
    "trivy-scanner:./services/trivy-scanner"
    "cosign-signer:./services/cosign-signer"
    "packer-builder:./services/packer-builder"
)

# Process core services
echo -e "${BLUE}Building Core Services:${NC}"
for service_path in "${CORE_SERVICES[@]}"; do
    IFS=':' read -r service path <<< "$service_path"
    build_and_push "$service" "$path"
done

# Ask about additional services
echo -e "${YELLOW}Build additional services? (y/n)${NC}"
read -r BUILD_ADDITIONAL
if [[ "$BUILD_ADDITIONAL" == "y" ]]; then
    echo -e "${BLUE}Building Additional Services:${NC}"
    for service_path in "${ADDITIONAL_SERVICES[@]}"; do
        IFS=':' read -r service path <<< "$service_path"
        build_and_push "$service" "$path"
    done
fi

# Summary
echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                    BUILD & PUSH SUMMARY                       ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

echo -e "${GREEN}Successfully Built: ${#BUILD_SUCCESS[@]} images${NC}"
for img in "${BUILD_SUCCESS[@]}"; do
    echo "  ✅ $img"
done

if [ ${#BUILD_FAILED[@]} -gt 0 ]; then
    echo ""
    echo -e "${RED}Failed to Build: ${#BUILD_FAILED[@]} images${NC}"
    for img in "${BUILD_FAILED[@]}"; do
        echo "  ❌ $img"
    done
fi

echo ""
echo -e "${GREEN}Successfully Pushed: ${#PUSH_SUCCESS[@]} images${NC}"
for img in "${PUSH_SUCCESS[@]}"; do
    echo "  ✅ ghcr.io/quantumlayer-dev/$img:${VERSION}"
done

if [ ${#PUSH_FAILED[@]} -gt 0 ]; then
    echo ""
    echo -e "${RED}Failed to Push: ${#PUSH_FAILED[@]} images${NC}"
    for img in "${PUSH_FAILED[@]}"; do
        echo "  ❌ $img"
    done
fi

# Only proceed to deployment if we have successful pushes
if [ ${#PUSH_SUCCESS[@]} -eq 0 ]; then
    echo ""
    echo -e "${RED}No images were successfully pushed. Cannot proceed with deployment.${NC}"
    exit 1
fi

echo ""
echo -e "${BLUE}=== Step 3: Update Kubernetes Deployments ===${NC}"
echo ""

# Update deployments with new image versions
echo "Updating Kubernetes deployments with version ${VERSION}..."

# Function to update deployment
update_deployment() {
    local namespace=$1
    local deployment=$2
    local image=$3
    
    if kubectl get deployment "$deployment" -n "$namespace" &>/dev/null; then
        echo -n "  Updating $namespace/$deployment... "
        if kubectl set image deployment/"$deployment" \
            "*=${REGISTRY}/${image}:${VERSION}" \
            -n "$namespace" --record &>/dev/null; then
            echo -e "${GREEN}✅${NC}"
        else
            echo -e "${RED}❌${NC}"
        fi
    fi
}

# Update deployments
for img in "${PUSH_SUCCESS[@]}"; do
    case "$img" in
        workflow-api)
            update_deployment "quantumlayer" "workflow-api" "$img"
            ;;
        llm-router)
            update_deployment "quantumlayer" "llm-router" "$img"
            ;;
        agent-orchestrator)
            update_deployment "quantumlayer" "agent-orchestrator" "$img"
            ;;
        image-registry)
            update_deployment "quantumlayer" "image-registry" "$img"
            ;;
        cve-tracker)
            update_deployment "security-services" "cve-tracker" "$img"
            ;;
        qinfra-ai)
            update_deployment "quantumlayer" "qinfra-ai" "$img"
            ;;
        workflows)
            update_deployment "temporal" "temporal-worker" "$img"
            update_deployment "temporal" "infra-workflow-worker" "$img"
            ;;
    esac
done

echo ""
echo -e "${BLUE}=== Step 4: Verify Deployments ===${NC}"
echo ""

# Wait for rollout
echo "Waiting for deployments to roll out..."
sleep 5

# Check pod status
echo "Current pod status:"
kubectl get pods --all-namespaces | grep -E "quantumlayer|temporal|security-services" | grep -v Completed

echo ""
echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                    DEPLOYMENT COMPLETE                        ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

echo -e "${GREEN}✅ Version ${VERSION} deployed successfully!${NC}"
echo ""
echo "Next steps:"
echo "1. Run ./test-baseline-v2.5.0.sh to test all services"
echo "2. Check pod logs if any services are failing:"
echo "   kubectl logs -n <namespace> <pod-name>"
echo "3. Run ./demo-all-services.sh for full functionality test"
echo ""
echo "Monitor deployment status:"
echo "  kubectl get pods --all-namespaces -w"