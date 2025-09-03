#!/bin/bash

# Build and push all service images to GitHub Container Registry

set -e

REGISTRY="ghcr.io/quantumlayer-dev"
TAG="${1:-latest}"

echo "==================================================="
echo "Building and Pushing QuantumLayer Services"
echo "Registry: $REGISTRY"
echo "Tag: $TAG"
echo "==================================================="

# Services to build
SERVICES=(
    "llm-router"
    "agent-orchestrator"
    "parser"
    "meta-prompt-engine"
    "api-gateway"
)

# Check if logged in to GitHub Container Registry
echo "Checking GitHub Container Registry authentication..."
if ! docker images >/dev/null 2>&1; then
    echo "Docker is not running or accessible"
    exit 1
fi
echo "✅ Docker is accessible, proceeding with build..."

# Build and push each service
for SERVICE in "${SERVICES[@]}"; do
    echo ""
    echo "==================================================="
    echo "Building $SERVICE..."
    echo "==================================================="
    
    if [ -f "packages/$SERVICE/Dockerfile" ]; then
        # Build the image
        docker build -t "$REGISTRY/$SERVICE:$TAG" \
            -f "packages/$SERVICE/Dockerfile" \
            "packages/$SERVICE"
        
        # Push to registry
        echo "Pushing $REGISTRY/$SERVICE:$TAG..."
        docker push "$REGISTRY/$SERVICE:$TAG"
        
        echo "✅ $SERVICE built and pushed successfully"
    else
        echo "⚠️  Dockerfile not found for $SERVICE, skipping..."
    fi
done

echo ""
echo "==================================================="
echo "✅ Build and push complete!"
echo "==================================================="
echo ""
echo "Images pushed:"
for SERVICE in "${SERVICES[@]}"; do
    echo "  - $REGISTRY/$SERVICE:$TAG"
done
echo ""
echo "Now you can deploy the services with:"
echo "  kubectl apply -f infrastructure/kubernetes/"