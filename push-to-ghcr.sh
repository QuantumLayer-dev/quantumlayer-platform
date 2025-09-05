#!/bin/bash
set -euo pipefail

# Push QuantumLayer Platform images to GitHub Container Registry
# Requires: gh auth login or docker login ghcr.io

echo "╔═══════════════════════════════════════════════════════════════════════════════╗"
echo "║              QuantumLayer Platform - Push to GHCR                              ║"
echo "║                     Pushing Docker Images to ghcr.io                           ║"
echo "╚═══════════════════════════════════════════════════════════════════════════════╝"
echo ""

# Configuration
GITHUB_USER="${GITHUB_USER:-satishgonella2024}"
GITHUB_REPO="${GITHUB_REPO:-quantumlayer-platform}"
REGISTRY="ghcr.io/${GITHUB_USER}"
TAG="${TAG:-latest}"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
RED='\033[0;31m'
NC='\033[0m'

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

# Check if logged in to ghcr.io
check_ghcr_login() {
    print_stage "Checking GHCR Authentication"
    
    if docker pull ghcr.io/github/super-linter:latest > /dev/null 2>&1; then
        print_success "Already logged in to ghcr.io"
    else
        print_info "Please login to GitHub Container Registry"
        echo "Run: echo \$GITHUB_TOKEN | docker login ghcr.io -u \$GITHUB_USER --password-stdin"
        echo "Or: gh auth token | docker login ghcr.io -u \$GITHUB_USER --password-stdin"
        exit 1
    fi
}

# Build and tag images
build_and_tag_images() {
    print_stage "Building and Tagging Images"
    
    # Workflow Worker
    print_info "Building workflow-worker..."
    cd packages/workflows
    go mod tidy
    docker build -t workflow-worker:${TAG} .
    docker tag workflow-worker:${TAG} ${REGISTRY}/workflow-worker:${TAG}
    docker tag workflow-worker:${TAG} ${REGISTRY}/workflow-worker:${TIMESTAMP}
    print_success "workflow-worker built and tagged"
    cd ../..
    
    # Parser Service (if it exists and builds)
    if [ -f packages/parser/Dockerfile ]; then
        print_info "Building parser-service..."
        cd packages/parser
        docker build -t parser-service:${TAG} . 2>/dev/null && {
            docker tag parser-service:${TAG} ${REGISTRY}/parser-service:${TAG}
            docker tag parser-service:${TAG} ${REGISTRY}/parser-service:${TIMESTAMP}
            print_success "parser-service built and tagged"
        } || print_info "Skipping parser-service (build failed)"
        cd ../..
    fi
    
    # Workflow API (if it exists)
    if [ -f packages/api/Dockerfile ]; then
        print_info "Building workflow-api..."
        cd packages/api
        docker build -t workflow-api:${TAG} .
        docker tag workflow-api:${TAG} ${REGISTRY}/workflow-api:${TAG}
        docker tag workflow-api:${TAG} ${REGISTRY}/workflow-api:${TIMESTAMP}
        print_success "workflow-api built and tagged"
        cd ../..
    fi
}

# Push images to GHCR
push_images() {
    print_stage "Pushing Images to GHCR"
    
    # Push workflow-worker
    print_info "Pushing workflow-worker to ${REGISTRY}..."
    docker push ${REGISTRY}/workflow-worker:${TAG}
    docker push ${REGISTRY}/workflow-worker:${TIMESTAMP}
    print_success "workflow-worker pushed successfully"
    
    # Push parser-service if it was built
    if docker images | grep -q "${REGISTRY}/parser-service"; then
        print_info "Pushing parser-service to ${REGISTRY}..."
        docker push ${REGISTRY}/parser-service:${TAG}
        docker push ${REGISTRY}/parser-service:${TIMESTAMP}
        print_success "parser-service pushed successfully"
    fi
    
    # Push workflow-api if it was built
    if docker images | grep -q "${REGISTRY}/workflow-api"; then
        print_info "Pushing workflow-api to ${REGISTRY}..."
        docker push ${REGISTRY}/workflow-api:${TAG}
        docker push ${REGISTRY}/workflow-api:${TIMESTAMP}
        print_success "workflow-api pushed successfully"
    fi
}

# Generate Kubernetes deployment YAML
generate_k8s_manifests() {
    print_stage "Generating Kubernetes Manifests"
    
    cat > k8s-deployment-ghcr.yaml <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: workflow-worker
  namespace: temporal
spec:
  replicas: 2
  selector:
    matchLabels:
      app: workflow-worker
  template:
    metadata:
      labels:
        app: workflow-worker
    spec:
      containers:
      - name: worker
        image: ${REGISTRY}/workflow-worker:${TAG}
        imagePullPolicy: Always
        env:
        - name: TEMPORAL_HOST
          value: "temporal-frontend:7233"
        - name: LOG_LEVEL
          value: "info"
        - name: QUANTUM_DROPS_URL
          value: "http://quantum-drops.quantumlayer.svc.cluster.local:8090"
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: parser-service
  namespace: quantumlayer
spec:
  replicas: 1
  selector:
    matchLabels:
      app: parser-service
  template:
    metadata:
      labels:
        app: parser-service
    spec:
      containers:
      - name: parser
        image: ${REGISTRY}/parser-service:${TAG}
        imagePullPolicy: Always
        ports:
        - containerPort: 8082
        env:
        - name: PORT
          value: "8082"
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
EOF
    
    print_success "Generated k8s-deployment-ghcr.yaml"
}

# Update existing deployments
update_deployments() {
    print_stage "Updating Kubernetes Deployments"
    
    print_info "Updating workflow-worker deployment..."
    kubectl set image deployment/workflow-worker worker=${REGISTRY}/workflow-worker:${TAG} -n temporal || {
        print_error "Failed to update workflow-worker"
    }
    
    print_info "Restarting workflow-worker..."
    kubectl rollout restart deployment/workflow-worker -n temporal
    
    print_info "Waiting for rollout..."
    kubectl rollout status deployment/workflow-worker -n temporal --timeout=120s || {
        print_info "Rollout is taking longer than expected"
    }
    
    print_success "Deployments updated"
}

# Show summary
show_summary() {
    print_stage "Summary"
    
    echo -e "\n${GREEN}Images pushed to GHCR:${NC}"
    echo "  • ${REGISTRY}/workflow-worker:${TAG}"
    echo "  • ${REGISTRY}/workflow-worker:${TIMESTAMP}"
    
    if docker images | grep -q "${REGISTRY}/parser-service"; then
        echo "  • ${REGISTRY}/parser-service:${TAG}"
        echo "  • ${REGISTRY}/parser-service:${TIMESTAMP}"
    fi
    
    if docker images | grep -q "${REGISTRY}/workflow-api"; then
        echo "  • ${REGISTRY}/workflow-api:${TAG}"
        echo "  • ${REGISTRY}/workflow-api:${TIMESTAMP}"
    fi
    
    echo -e "\n${CYAN}To deploy to Kubernetes:${NC}"
    echo "  kubectl apply -f k8s-deployment-ghcr.yaml"
    
    echo -e "\n${CYAN}To pull images:${NC}"
    echo "  docker pull ${REGISTRY}/workflow-worker:${TAG}"
    
    echo -e "\n${CYAN}Verify deployment:${NC}"
    echo "  kubectl get pods -n temporal"
    echo "  kubectl logs -n temporal deployment/workflow-worker"
}

# Main execution
main() {
    print_info "Starting push to GitHub Container Registry..."
    
    # Check authentication
    check_ghcr_login
    
    # Build and tag
    build_and_tag_images
    
    # Push to registry
    push_images
    
    # Generate manifests
    generate_k8s_manifests
    
    # Update deployments
    read -p "Do you want to update Kubernetes deployments now? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        update_deployments
    fi
    
    # Show summary
    show_summary
    
    print_success "All operations completed successfully!"
}

# Run main function
main "$@"