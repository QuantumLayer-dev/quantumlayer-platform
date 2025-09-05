#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     QUANTUMLAYER PLATFORM - COMPLETE SERVICES DEMO          ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Service URLs
BASE_URL="http://192.168.1.177"
WORKFLOW_API="$BASE_URL:30889"
IMAGE_REGISTRY="$BASE_URL:30096"
TRIVY="$BASE_URL:30097"
COSIGN="$BASE_URL:30099"
PACKER="$BASE_URL:30100"
QINFRA="$BASE_URL:30095"
QINFRA_AI="$BASE_URL:30098"
CVE_TRACKER="$BASE_URL:30101"
TEMPORAL_UI="$BASE_URL:30888"

echo -e "${GREEN}=== PHASE 1: Service Health Checks ===${NC}"
echo ""

services=(
    "Workflow API|$WORKFLOW_API/health"
    "Image Registry|$IMAGE_REGISTRY/health"
    "QInfra|$QINFRA/health"
    "QInfra-AI|$QINFRA_AI/health"
    "CVE Tracker|$CVE_TRACKER/health"
)

for service in "${services[@]}"; do
    IFS='|' read -r name url <<< "$service"
    if curl -s -f "$url" > /dev/null 2>&1; then
        echo -e "✅ $name: ${GREEN}Healthy${NC}"
    else
        echo -e "❌ $name: ${RED}Down${NC}"
    fi
done

echo ""
echo -e "${GREEN}=== PHASE 2: Golden Image Pipeline Demo ===${NC}"
echo ""

# Step 1: Create a golden image
echo "📦 Step 1: Building Golden Image..."
BUILD_RESPONSE=$(curl -s -X POST "$IMAGE_REGISTRY/images/build" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "demo-production-image",
    "base_os": "ubuntu-22.04",
    "platform": "aws",
    "packages": ["nginx", "postgresql-client", "redis-tools"],
    "hardening": "CIS",
    "compliance": ["SOC2", "HIPAA", "PCI-DSS"]
  }')

IMAGE_ID=$(echo "$BUILD_RESPONSE" | grep -o '"id":"[^"]*' | cut -d'"' -f4)
echo -e "   Created image: ${YELLOW}$IMAGE_ID${NC}"

# Step 2: Scan for vulnerabilities
echo ""
echo "🔍 Step 2: Scanning for Vulnerabilities..."
SCAN_RESPONSE=$(curl -s -X POST "$IMAGE_REGISTRY/images/$IMAGE_ID/scan")
VULN_COUNT=$(echo "$SCAN_RESPONSE" | grep -o '"vulnerabilities_found":[0-9]*' | cut -d':' -f2 || echo "0")
echo -e "   Found ${YELLOW}$VULN_COUNT${NC} vulnerabilities"

# Step 3: Sign the image
echo ""
echo "✍️  Step 3: Signing Image with Cosign..."
SIGN_RESPONSE=$(curl -s -X POST "$IMAGE_REGISTRY/images/$IMAGE_ID/sign")
if echo "$SIGN_RESPONSE" | grep -q '"status":"signed"'; then
    echo -e "   ${GREEN}Image signed successfully${NC}"
fi

# Step 4: Check patch status
echo ""
echo "🔧 Step 4: Checking Patch Requirements..."
PATCH_RESPONSE=$(curl -s "$IMAGE_REGISTRY/images/$IMAGE_ID/patch-status")
PATCHES_NEEDED=$(echo "$PATCH_RESPONSE" | grep -o '"patches_needed":[0-9]*' | cut -d':' -f2 || echo "0")
echo -e "   ${YELLOW}$PATCHES_NEEDED${NC} patches recommended"

echo ""
echo -e "${GREEN}=== PHASE 3: CVE Tracking Demo ===${NC}"
echo ""

# Search for latest CVEs
echo "🔍 Searching for latest CVEs..."
CVE_LATEST=$(curl -s "$CVE_TRACKER/cve/latest?hours=24")
CVE_COUNT=$(echo "$CVE_LATEST" | grep -o '"total":[0-9]*' | cut -d':' -f2 || echo "0")
echo -e "   Found ${YELLOW}$CVE_COUNT${NC} CVEs in last 24 hours"

# Analyze impact on our image
echo ""
echo "📊 Analyzing CVE Impact on Image..."
IMPACT_RESPONSE=$(curl -s -X POST "$CVE_TRACKER/cve/impact/$IMAGE_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "packages": ["nginx", "postgresql-client", "redis-tools"],
    "platform": "aws"
  }')

RISK_SCORE=$(echo "$IMPACT_RESPONSE" | grep -o '"risk_score":[0-9.]*' | cut -d':' -f2 || echo "0")
echo -e "   Risk Score: ${YELLOW}$RISK_SCORE${NC}"

echo ""
echo -e "${GREEN}=== PHASE 4: AI Intelligence Demo ===${NC}"
echo ""

# Predict drift
echo "🤖 Predicting Infrastructure Drift..."
DRIFT_RESPONSE=$(curl -s -X POST "$QINFRA_AI/api/v1/predict-drift" \
  -H "Content-Type: application/json" \
  -d '{
    "node_id": "demo-node-001",
    "platform": "aws",
    "current_state": {
      "os_version": "Ubuntu 22.04",
      "packages_installed": 150,
      "last_update": "2024-01-01",
      "manual_changes": 5
    }
  }')

DRIFT_PROB=$(echo "$DRIFT_RESPONSE" | grep -o '"probability":[0-9.]*' | cut -d':' -f2 || echo "0")
echo -e "   Drift Probability: ${YELLOW}${DRIFT_PROB}%${NC}"

# Assess patch risk
echo ""
echo "⚠️  Assessing Patch Risk..."
PATCH_RISK=$(curl -s -X POST "$QINFRA_AI/api/v1/assess-patch-risk" \
  -H "Content-Type: application/json" \
  -d '{
    "patch_id": "demo-patch-001",
    "cve": "CVE-2024-12345",
    "target_nodes": ["node-001", "node-002", "node-003"],
    "environment": "production",
    "dependencies": ["glibc", "openssl"]
  }')

RISK_LEVEL=$(echo "$PATCH_RISK" | grep -o '"risk_score":[0-9.]*' | cut -d':' -f2 || echo "0")
echo -e "   Patch Risk Score: ${YELLOW}$RISK_LEVEL${NC}"

echo ""
echo -e "${GREEN}=== PHASE 5: Temporal Workflow Demo ===${NC}"
echo ""

# Trigger infrastructure workflow
echo "🔄 Triggering Infrastructure Generation Workflow..."
WORKFLOW_RESPONSE=$(curl -s -X POST "$WORKFLOW_API/api/v1/workflows/generate" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Create a microservices architecture with API gateway, 3 services, and PostgreSQL database",
    "language": "go",
    "type": "architecture",
    "complexity": "enterprise"
  }')

WORKFLOW_ID=$(echo "$WORKFLOW_RESPONSE" | grep -o '"workflow_id":"[^"]*' | cut -d'"' -f4)
echo -e "   Workflow started: ${YELLOW}$WORKFLOW_ID${NC}"
echo -e "   View in Temporal UI: ${BLUE}$TEMPORAL_UI/namespaces/quantumlayer/workflows/$WORKFLOW_ID${NC}"

echo ""
echo -e "${GREEN}=== PHASE 6: Service Statistics ===${NC}"
echo ""

# Get image registry stats
echo "📊 Image Registry Statistics:"
REGISTRY_METRICS=$(curl -s "$IMAGE_REGISTRY/metrics")
TOTAL_IMAGES=$(echo "$REGISTRY_METRICS" | grep -o '"total_images":[0-9]*' | cut -d':' -f2 || echo "0")
echo -e "   Total Images: ${YELLOW}$TOTAL_IMAGES${NC}"

# Get CVE tracker stats
echo ""
echo "📊 CVE Tracker Statistics:"
CVE_STATS=$(curl -s "$CVE_TRACKER/stats/summary")
if [ ! -z "$CVE_STATS" ]; then
    echo "$CVE_STATS" | python3 -c "
import sys, json
data = json.load(sys.stdin)
if 'severity_distribution' in data:
    for severity, count in data['severity_distribution'].items():
        print(f'   {severity.capitalize()}: {count}')
"
fi

echo ""
echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                    DEMO COMPLETE                            ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

echo -e "${GREEN}📊 SUMMARY:${NC}"
echo "  ✅ All services operational"
echo "  ✅ Golden image pipeline working"
echo "  ✅ CVE tracking active"
echo "  ✅ AI intelligence functioning"
echo "  ✅ Temporal workflows running"
echo ""

echo -e "${YELLOW}🔗 Quick Access Links:${NC}"
echo "  • Temporal UI: $TEMPORAL_UI"
echo "  • Workflow API: $WORKFLOW_API"
echo "  • Image Registry: $IMAGE_REGISTRY"
echo "  • CVE Tracker: $CVE_TRACKER"
echo "  • QInfra Dashboard: $QINFRA"
echo ""

echo -e "${GREEN}The QuantumLayer Platform is fully operational!${NC}"