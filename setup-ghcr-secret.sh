#!/bin/bash

# Script to create GitHub Container Registry secret for Kubernetes
# You need to provide your GitHub username and a Personal Access Token (PAT)

echo "==================================================="
echo "GitHub Container Registry Secret Setup"
echo "==================================================="
echo ""
echo "To pull images from ghcr.io/quantumlayer-dev/, you need:"
echo "1. A GitHub account with access to the QuantumLayer-dev organization"
echo "2. A Personal Access Token (PAT) with 'read:packages' scope"
echo ""
echo "To create a PAT:"
echo "1. Go to https://github.com/settings/tokens/new"
echo "2. Select 'read:packages' scope"
echo "3. Generate token and copy it"
echo ""

read -p "Enter your GitHub username: " GITHUB_USER
read -s -p "Enter your GitHub Personal Access Token: " GITHUB_TOKEN
echo ""

# Create the secret
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username="$GITHUB_USER" \
  --docker-password="$GITHUB_TOKEN" \
  --docker-email="${GITHUB_USER}@users.noreply.github.com" \
  -n quantumlayer

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Secret 'ghcr-secret' created successfully in namespace 'quantumlayer'"
    echo ""
    echo "The deployments are already configured to use this secret via:"
    echo "  imagePullSecrets:"
    echo "  - name: ghcr-secret"
else
    echo ""
    echo "❌ Failed to create secret. Please check your credentials and try again."
fi