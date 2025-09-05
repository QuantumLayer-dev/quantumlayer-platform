#!/bin/bash
set -euo pipefail

# Build and Deploy Script for QuantumLayer Platform
# This script builds Docker images and deploys all services

echo "╔═══════════════════════════════════════════════════════════════════════════════╗"
echo "║              QuantumLayer Platform - Build & Deploy Script                     ║"
echo "║                     Building and Deploying All Services                        ║"
echo "╚═══════════════════════════════════════════════════════════════════════════════╝"
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
RED='\033[0;31m'
NC='\033[0m'

# Configuration
REGISTRY="localhost:5000"  # Use local registry or change to your registry
NAMESPACE_TEMPORAL="temporal"
NAMESPACE_QUANTUM="quantumlayer"
IMAGE_TAG="latest-$(date +%Y%m%d-%H%M%S)"

print_stage() {
    echo -e "\n${BLUE}═══ $1 ═══${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_info() {
    echo -e "${CYAN}ℹ️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if local registry is running
check_registry() {
    print_stage "Checking Docker Registry"
    if curl -s http://localhost:5000/v2/ > /dev/null 2>&1; then
        print_success "Local registry is running"
        USE_REGISTRY=true
    else
        print_info "No local registry found, will use local images"
        USE_REGISTRY=false
    fi
}

# Build workflow service
build_workflow_service() {
    print_stage "Building Workflow Service"
    
    cd packages/workflows
    
    # Build Go binary
    print_info "Building Go binary..."
    go mod tidy
    go build -o worker ./cmd/worker/main.go
    
    # Build Docker image
    print_info "Building Docker image..."
    docker build -t workflow-worker:${IMAGE_TAG} .
    docker tag workflow-worker:${IMAGE_TAG} workflow-worker:latest
    
    if [ "$USE_REGISTRY" = true ]; then
        docker tag workflow-worker:${IMAGE_TAG} ${REGISTRY}/workflow-worker:${IMAGE_TAG}
        docker push ${REGISTRY}/workflow-worker:${IMAGE_TAG}
        print_success "Pushed workflow-worker:${IMAGE_TAG} to registry"
    fi
    
    print_success "Workflow service built successfully"
    cd ../..
}

# Build parser service
build_parser_service() {
    print_stage "Building Parser Service"
    
    if [ -f packages/parser/Dockerfile ]; then
        cd packages/parser
        
        print_info "Building Parser service Docker image..."
        docker build -t parser-service:${IMAGE_TAG} .
        docker tag parser-service:${IMAGE_TAG} parser-service:latest
        
        if [ "$USE_REGISTRY" = true ]; then
            docker tag parser-service:${IMAGE_TAG} ${REGISTRY}/parser-service:${IMAGE_TAG}
            docker push ${REGISTRY}/parser-service:${IMAGE_TAG}
            print_success "Pushed parser-service:${IMAGE_TAG} to registry"
        fi
        
        print_success "Parser service built successfully"
        cd ../..
    else
        print_info "Parser service Dockerfile not found, skipping"
    fi
}

# Deploy to Kubernetes
deploy_services() {
    print_stage "Deploying Services to Kubernetes"
    
    # Deploy workflow worker
    print_info "Deploying workflow worker..."
    
    # Check if deployment exists
    if kubectl get deployment workflow-worker -n ${NAMESPACE_TEMPORAL} > /dev/null 2>&1; then
        if [ "$USE_REGISTRY" = true ]; then
            kubectl set image deployment/workflow-worker worker=${REGISTRY}/workflow-worker:${IMAGE_TAG} -n ${NAMESPACE_TEMPORAL}
        else
            # For local images, we need to use imagePullPolicy: Never
            kubectl patch deployment workflow-worker -n ${NAMESPACE_TEMPORAL} \
                -p '{"spec":{"template":{"spec":{"containers":[{"name":"worker","image":"workflow-worker:latest","imagePullPolicy":"Never"}]}}}}'
        fi
        
        # Force rollout
        kubectl rollout restart deployment/workflow-worker -n ${NAMESPACE_TEMPORAL}
        print_info "Waiting for rollout to complete..."
        kubectl rollout status deployment/workflow-worker -n ${NAMESPACE_TEMPORAL} --timeout=120s || true
        print_success "Workflow worker deployed"
    else
        print_error "Workflow worker deployment not found"
    fi
    
    # Deploy parser service if it exists
    if kubectl get deployment parser-service -n ${NAMESPACE_QUANTUM} > /dev/null 2>&1; then
        print_info "Deploying parser service..."
        if [ "$USE_REGISTRY" = true ]; then
            kubectl set image deployment/parser-service parser=${REGISTRY}/parser-service:${IMAGE_TAG} -n ${NAMESPACE_QUANTUM}
        else
            kubectl patch deployment parser-service -n ${NAMESPACE_QUANTUM} \
                -p '{"spec":{"template":{"spec":{"containers":[{"name":"parser","image":"parser-service:latest","imagePullPolicy":"Never"}]}}}}'
        fi
        kubectl rollout restart deployment/parser-service -n ${NAMESPACE_QUANTUM}
        print_success "Parser service deployed"
    fi
}

# Load images to kind cluster (if using kind)
load_to_kind() {
    print_stage "Loading Images to Kind Cluster"
    
    if command -v kind &> /dev/null && kind get clusters 2>/dev/null | grep -q .; then
        CLUSTER_NAME=$(kind get clusters | head -1)
        print_info "Loading images to kind cluster: $CLUSTER_NAME"
        
        kind load docker-image workflow-worker:latest --name $CLUSTER_NAME
        kind load docker-image parser-service:latest --name $CLUSTER_NAME 2>/dev/null || true
        
        print_success "Images loaded to kind cluster"
    else
        print_info "Kind not detected, skipping image loading"
    fi
}

# Verify deployments
verify_deployments() {
    print_stage "Verifying Deployments"
    
    echo -e "\n${CYAN}Checking pod status:${NC}"
    kubectl get pods -n ${NAMESPACE_TEMPORAL} | grep workflow-worker || true
    kubectl get pods -n ${NAMESPACE_QUANTUM} | grep parser || true
    
    echo -e "\n${CYAN}Checking recent events:${NC}"
    kubectl get events -n ${NAMESPACE_TEMPORAL} --field-selector involvedObject.kind=Pod | tail -5 || true
}

# Main execution
main() {
    print_info "Starting build and deploy process..."
    
    # Check prerequisites
    check_registry
    
    # Build services
    build_workflow_service
    build_parser_service
    
    # Load to kind if available
    load_to_kind
    
    # Deploy to Kubernetes
    deploy_services
    
    # Verify deployments
    verify_deployments
    
    print_stage "Build and Deploy Complete"
    print_success "All services have been built and deployed!"
    
    echo -e "\n${CYAN}Next steps:${NC}"
    echo "1. Check pod status: kubectl get pods -n temporal"
    echo "2. Check logs: kubectl logs -n temporal deployment/workflow-worker"
    echo "3. Run tests: ./test-enhanced-pipeline.sh"
}

# Run main function
main "$@"