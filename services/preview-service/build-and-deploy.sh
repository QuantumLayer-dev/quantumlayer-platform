#!/bin/bash
set -e

echo "Building Preview Service..."

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Build Docker image
print_status "Building Docker image..."
docker build -t ghcr.io/quantumlayer-dev/preview-service:latest .

# Push to registry
print_status "Pushing to GitHub Container Registry..."
docker push ghcr.io/quantumlayer-dev/preview-service:latest

# Deploy to Kubernetes
print_status "Deploying to Kubernetes..."
kubectl apply -f k8s-deployment.yaml

# Wait for deployment
print_status "Waiting for deployment to be ready..."
kubectl rollout status deployment/preview-service -n quantumlayer --timeout=300s

# Get pod status
print_status "Checking pod status..."
kubectl get pods -n quantumlayer -l app=preview-service

# Display access URL
NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')
print_status "Preview Service deployed successfully!"
echo ""
echo "Access the service at: http://${NODE_IP}:30900"
echo "Or use port-forward: kubectl port-forward -n quantumlayer svc/preview-service 3000:3000"
echo ""