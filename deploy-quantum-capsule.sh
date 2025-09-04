#!/bin/bash
set -euo pipefail

# Deployment script for QuantumCapsule ecosystem
# This deploys the new Sandbox Executor and Capsule Builder services

NAMESPACE="quantumlayer"
REGISTRY="${REGISTRY:-localhost:5000}"

echo "════════════════════════════════════════════════════════════════"
echo "   QuantumCapsule Ecosystem Deployment"
echo "   Registry: $REGISTRY"
echo "   Namespace: $NAMESPACE"
echo "════════════════════════════════════════════════════════════════"

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl not found"
        exit 1
    fi
    
    if ! command -v docker &> /dev/null; then
        log_error "docker not found"
        exit 1
    fi
    
    if ! kubectl get namespace $NAMESPACE &> /dev/null; then
        log_info "Creating namespace $NAMESPACE"
        kubectl create namespace $NAMESPACE
    fi
    
    log_info "Prerequisites check passed ✓"
}

# Build and push Docker images
build_and_push_images() {
    log_info "Building Docker images..."
    
    # Build Sandbox Executor
    log_info "Building Sandbox Executor..."
    cd packages/sandbox-executor
    docker build -t $REGISTRY/sandbox-executor:latest .
    docker push $REGISTRY/sandbox-executor:latest
    cd ../..
    
    # Build Capsule Builder
    log_info "Building Capsule Builder..."
    cd packages/capsule-builder
    docker build -t $REGISTRY/capsule-builder:latest .
    docker push $REGISTRY/capsule-builder:latest
    cd ../..
    
    log_info "Docker images built and pushed ✓"
}

# Deploy Sandbox Executor
deploy_sandbox_executor() {
    log_info "Deploying Sandbox Executor..."
    
    # Update image in deployment
    sed -i "s|quantumlayer/sandbox-executor:latest|$REGISTRY/sandbox-executor:latest|g" \
        packages/sandbox-executor/k8s-deployment.yaml
    
    kubectl apply -f packages/sandbox-executor/k8s-deployment.yaml
    
    # Wait for deployment
    kubectl rollout status deployment/sandbox-executor -n $NAMESPACE --timeout=300s
    
    log_info "Sandbox Executor deployed ✓"
}

# Deploy Capsule Builder
deploy_capsule_builder() {
    log_info "Deploying Capsule Builder..."
    
    # Create deployment manifest
    cat > /tmp/capsule-builder-deployment.yaml <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: capsule-builder
  namespace: $NAMESPACE
  labels:
    app: capsule-builder
    component: builder
spec:
  replicas: 2
  selector:
    matchLabels:
      app: capsule-builder
  template:
    metadata:
      labels:
        app: capsule-builder
        component: builder
    spec:
      containers:
      - name: capsule-builder
        image: $REGISTRY/capsule-builder:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 8092
          name: http
        env:
        - name: PORT
          value: "8092"
        - name: QUANTUM_DROPS_URL
          value: "http://quantum-drops.quantumlayer.svc.cluster.local:8090"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8092
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8092
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: capsule-builder
  namespace: $NAMESPACE
  labels:
    app: capsule-builder
spec:
  selector:
    app: capsule-builder
  ports:
  - port: 8092
    targetPort: 8092
    name: http
  type: ClusterIP
EOF
    
    kubectl apply -f /tmp/capsule-builder-deployment.yaml
    
    # Wait for deployment
    kubectl rollout status deployment/capsule-builder -n $NAMESPACE --timeout=300s
    
    log_info "Capsule Builder deployed ✓"
}

# Update workflow activities to use new services
update_workflow_integration() {
    log_info "Updating workflow integration..."
    
    # Add environment variables to workflow-worker
    kubectl set env deployment/workflow-worker -n temporal \
        SANDBOX_EXECUTOR_URL=http://sandbox-executor.$NAMESPACE.svc.cluster.local:8091 \
        CAPSULE_BUILDER_URL=http://capsule-builder.$NAMESPACE.svc.cluster.local:8092
    
    log_info "Workflow integration updated ✓"
}

# Create test job
create_test_job() {
    log_info "Creating test job..."
    
    cat > /tmp/test-quantum-capsule.yaml <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: test-quantum-capsule-$(date +%s)
  namespace: $NAMESPACE
spec:
  template:
    spec:
      containers:
      - name: test
        image: curlimages/curl:latest
        command: ["/bin/sh", "-c"]
        args:
          - |
            echo "Testing Sandbox Executor..."
            curl -X POST http://sandbox-executor:8091/api/v1/execute \
              -H "Content-Type: application/json" \
              -d '{
                "language": "python",
                "code": "print(\"Hello from QuantumCapsule!\")",
                "timeout": 10
              }'
            
            echo ""
            echo "Testing Capsule Builder..."
            curl -X POST http://capsule-builder:8092/api/v1/build \
              -H "Content-Type: application/json" \
              -d '{
                "workflow_id": "test-123",
                "language": "python",
                "framework": "fastapi",
                "type": "api",
                "name": "test-api",
                "description": "Test API",
                "code": "from fastapi import FastAPI\napp = FastAPI()\n@app.get(\"/\")\ndef read_root():\n    return {\"Hello\": \"World\"}",
                "dependencies": ["fastapi", "uvicorn"]
              }'
      restartPolicy: Never
  backoffLimit: 1
EOF
    
    kubectl apply -f /tmp/test-quantum-capsule.yaml
    
    log_info "Test job created ✓"
}

# Main deployment flow
main() {
    check_prerequisites
    build_and_push_images
    deploy_sandbox_executor
    deploy_capsule_builder
    update_workflow_integration
    create_test_job
    
    echo ""
    echo "════════════════════════════════════════════════════════════════"
    echo "   QuantumCapsule Ecosystem Deployment Complete!"
    echo "════════════════════════════════════════════════════════════════"
    echo ""
    echo "Services deployed:"
    echo "  • Sandbox Executor: http://sandbox-executor.$NAMESPACE.svc.cluster.local:8091"
    echo "  • Capsule Builder: http://capsule-builder.$NAMESPACE.svc.cluster.local:8092"
    echo ""
    echo "Check status:"
    echo "  kubectl get pods -n $NAMESPACE"
    echo ""
    echo "View logs:"
    echo "  kubectl logs -f deployment/sandbox-executor -n $NAMESPACE"
    echo "  kubectl logs -f deployment/capsule-builder -n $NAMESPACE"
    echo ""
    echo "Test endpoints:"
    echo "  kubectl port-forward svc/sandbox-executor 8091:8091 -n $NAMESPACE"
    echo "  kubectl port-forward svc/capsule-builder 8092:8092 -n $NAMESPACE"
}

# Run main function
main "$@"