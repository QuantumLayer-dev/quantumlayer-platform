#!/bin/bash

# QuantumLayer GitHub Repository Setup Script
# This script creates the GitHub repo and configures GHCR

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Configuration
GITHUB_ORG="QuantumLayer-dev"
REPO_NAME="quantumlayer-platform"
REPO_DESCRIPTION="Enterprise-grade AI Software Factory Platform"

echo -e "${GREEN}QuantumLayer GitHub Repository Setup${NC}"
echo "======================================"

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo -e "${RED}GitHub CLI (gh) is not installed. Please install it first:${NC}"
    echo "https://cli.github.com/"
    exit 1
fi

# Check if authenticated
if ! gh auth status &> /dev/null; then
    echo -e "${YELLOW}Not authenticated with GitHub. Running 'gh auth login'...${NC}"
    gh auth login
fi

# Create organization if it doesn't exist (will fail if already exists, that's ok)
echo -e "${YELLOW}Checking GitHub organization...${NC}"
gh api user/orgs --jq '.[].login' | grep -q "^${GITHUB_ORG}$" || {
    echo -e "${YELLOW}Organization ${GITHUB_ORG} not found in your account${NC}"
    echo -e "${YELLOW}Please create it manually at: https://github.com/organizations/new${NC}"
    read -p "Press enter once the organization is created..."
}

# Create repository
echo -e "${GREEN}Creating repository ${GITHUB_ORG}/${REPO_NAME}...${NC}"
gh repo create ${GITHUB_ORG}/${REPO_NAME} \
    --description "${REPO_DESCRIPTION}" \
    --public \
    --clone=false \
    --add-readme=false \
    2>/dev/null || echo -e "${YELLOW}Repository already exists${NC}"

# Configure repository settings
echo -e "${GREEN}Configuring repository settings...${NC}"

# Enable GitHub Pages (for documentation)
gh api repos/${GITHUB_ORG}/${REPO_NAME}/pages \
    --method POST \
    -f source='{"branch":"main","path":"/docs"}' \
    2>/dev/null || echo -e "${YELLOW}GitHub Pages already configured${NC}"

# Create repository secrets for K8s deployment
echo -e "${GREEN}Setting up repository secrets...${NC}"
echo -e "${YELLOW}You'll need to add these secrets manually:${NC}"
echo "  1. KUBE_CONFIG - Your Kubernetes config (base64 encoded)"
echo "  2. DOCKERHUB_USERNAME - (optional) Docker Hub username"
echo "  3. DOCKERHUB_TOKEN - (optional) Docker Hub token"
echo ""
echo "To add secrets, visit:"
echo "https://github.com/${GITHUB_ORG}/${REPO_NAME}/settings/secrets/actions"

# Set up branch protection
echo -e "${GREEN}Setting up branch protection for main...${NC}"
gh api repos/${GITHUB_ORG}/${REPO_NAME}/branches/main/protection \
    --method PUT \
    -f required_status_checks='{"strict":true,"contexts":["continuous-integration"]}' \
    -f enforce_admins=false \
    -f required_pull_request_reviews='{"required_approving_review_count":1,"dismiss_stale_reviews":true}' \
    -f restrictions=null \
    -f allow_force_pushes=false \
    -f allow_deletions=false \
    2>/dev/null || echo -e "${YELLOW}Branch protection already configured${NC}"

# Configure GHCR
echo -e "${GREEN}Configuring GitHub Container Registry...${NC}"
echo -e "${YELLOW}GHCR is automatically enabled for public repositories${NC}"
echo "Container images will be published to:"
echo "  ghcr.io/${GITHUB_ORG}/quantumlayer-api"
echo "  ghcr.io/${GITHUB_ORG}/quantumlayer-web"
echo "  ghcr.io/${GITHUB_ORG}/quantumlayer-worker"
echo "  ghcr.io/${GITHUB_ORG}/quantumlayer-llm-router"

# Add topics to repository
echo -e "${GREEN}Adding repository topics...${NC}"
gh api repos/${GITHUB_ORG}/${REPO_NAME}/topics \
    --method PUT \
    -f names='["ai","llm","code-generation","kubernetes","microservices","multi-tenant","golang","typescript","nextjs"]' \
    2>/dev/null || echo -e "${YELLOW}Topics already set${NC}"

# Set up git remote
echo -e "${GREEN}Setting up git remote...${NC}"
git remote remove origin 2>/dev/null || true
git remote add origin git@github.com:${GITHUB_ORG}/${REPO_NAME}.git

# Create .gitignore if it doesn't exist
if [ ! -f .gitignore ]; then
    echo -e "${GREEN}Creating .gitignore...${NC}"
    cat > .gitignore << 'EOF'
# Dependencies
node_modules/
vendor/
*.lock
package-lock.json

# Environment files
.env
.env.local
.env.*.local
!.env.example
!.env.k8s

# Build outputs
dist/
build/
out/
.next/
*.out
*.exe
*.dll
*.so
*.dylib

# IDE
.vscode/
.idea/
*.swp
*.swo
*~
.DS_Store

# Testing
coverage/
*.test
*.cover
coverage.html
coverage.out

# Logs
logs/
*.log
npm-debug.log*
yarn-debug.log*
yarn-error.log*

# Runtime data
pids/
*.pid
*.seed
*.pid.lock

# Temporary files
tmp/
temp/
.cache/

# Docker
.dockerignore

# Kubernetes
*.kubeconfig
kubeconfig

# Terraform
*.tfstate
*.tfstate.*
.terraform/
terraform.tfvars

# Python
__pycache__/
*.py[cod]
*$py.class
.Python
venv/
.venv/

# Go
/bin/
*.test
*.out

# Database
*.sqlite
*.sqlite3
*.db

# Secrets
*.key
*.pem
*.crt
*.p12
secrets/
EOF
fi

# Create initial README if needed
if [ ! -f README.md ]; then
    echo -e "${GREEN}Creating README.md...${NC}"
    cat > README.md << 'EOF'
# QuantumLayer Platform

Enterprise-grade AI Software Factory Platform with multi-LLM support, multi-tenancy, and Kubernetes-native deployment.

## Quick Start

```bash
# Setup environment
make setup

# Start services in Kubernetes
kubectl apply -f infrastructure/kubernetes/

# Access services
# API: http://192.168.7.235:30800
# Web: http://192.168.7.235:30300
# Grafana: http://192.168.7.235:30301
```

## Documentation

See [docs/](./docs/) for complete documentation.

## Container Images

Our container images are available on GitHub Container Registry:
- `ghcr.io/quantumlayer-dev/quantumlayer-api`
- `ghcr.io/quantumlayer-dev/quantumlayer-web`
- `ghcr.io/quantumlayer-dev/quantumlayer-worker`
- `ghcr.io/quantumlayer-dev/quantumlayer-llm-router`

## License

Copyright (c) 2024 QuantumLayer
EOF
fi

echo -e "${GREEN}Repository setup complete!${NC}"
echo ""
echo "Next steps:"
echo "1. Add Kubernetes config secret:"
echo "   cat ~/.kube/config | base64 | gh secret set KUBE_CONFIG -R ${GITHUB_ORG}/${REPO_NAME}"
echo ""
echo "2. Push your code:"
echo "   git add ."
echo "   git commit -m 'Initial commit: Complete platform architecture'"
echo "   git push -u origin main"
echo ""
echo "3. Access your repository:"
echo "   https://github.com/${GITHUB_ORG}/${REPO_NAME}"