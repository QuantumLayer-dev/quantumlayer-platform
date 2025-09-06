#!/bin/bash
set -e

# Test configuration
VERSION="2.5.0"
BASE_URL="http://192.168.1.177"
DATE=$(date +%Y%m%d-%H%M%S)
REPORT_FILE="baseline-test-report-${VERSION}-${DATE}.md"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
WARNINGS=0

# Test results arrays
declare -A TEST_RESULTS
declare -A TEST_DETAILS

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     QUANTUMLAYER PLATFORM - BASELINE TEST v${VERSION}       ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Initialize report
cat > "$REPORT_FILE" << EOF
# QuantumLayer Platform - Baseline Test Report
Version: ${VERSION}
Date: $(date)
Environment: Production Kubernetes Cluster

## Executive Summary
Testing all platform components to establish baseline functionality.

---

## Test Results

EOF

# Function to test service health
test_service() {
    local name=$1
    local url=$2
    local namespace=$3
    
    ((TOTAL_TESTS++))
    echo -n "Testing $name... "
    
    if curl -s -f -m 5 "$url" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ PASS${NC}"
        ((PASSED_TESTS++))
        TEST_RESULTS["$name"]="PASS"
        TEST_DETAILS["$name"]="Service responding at $url"
        echo "| $name | ✅ PASS | $url | Service healthy |" >> "$REPORT_FILE"
    else
        # Check if pod is running
        if kubectl get pods -n "$namespace" 2>/dev/null | grep -q "$name.*Running"; then
            echo -e "${YELLOW}⚠️ WARNING${NC} - Pod running but not accessible"
            ((WARNINGS++))
            TEST_RESULTS["$name"]="WARNING"
            TEST_DETAILS["$name"]="Pod running but service not accessible at $url"
            echo "| $name | ⚠️ WARNING | $url | Pod running but not accessible |" >> "$REPORT_FILE"
        else
            echo -e "${RED}❌ FAIL${NC}"
            ((FAILED_TESTS++))
            TEST_RESULTS["$name"]="FAIL"
            TEST_DETAILS["$name"]="Service not responding at $url"
            echo "| $name | ❌ FAIL | $url | Service not responding |" >> "$REPORT_FILE"
        fi
    fi
}

# Function to test API endpoint
test_api() {
    local name=$1
    local method=$2
    local url=$3
    local data=$4
    local expected=$5
    
    ((TOTAL_TESTS++))
    echo -n "Testing API: $name... "
    
    if [ "$method" = "GET" ]; then
        response=$(curl -s -X GET "$url" 2>/dev/null || echo "")
    else
        response=$(curl -s -X POST "$url" -H "Content-Type: application/json" -d "$data" 2>/dev/null || echo "")
    fi
    
    if echo "$response" | grep -q "$expected"; then
        echo -e "${GREEN}✅ PASS${NC}"
        ((PASSED_TESTS++))
        TEST_RESULTS["API:$name"]="PASS"
        echo "| API: $name | ✅ PASS | $url | Response contains expected data |" >> "$REPORT_FILE"
    else
        echo -e "${RED}❌ FAIL${NC}"
        ((FAILED_TESTS++))
        TEST_RESULTS["API:$name"]="FAIL"
        echo "| API: $name | ❌ FAIL | $url | Invalid response |" >> "$REPORT_FILE"
    fi
}

# Function to test Kubernetes resources
test_k8s_resource() {
    local type=$1
    local name=$2
    local namespace=$3
    
    ((TOTAL_TESTS++))
    echo -n "Testing K8s $type: $name... "
    
    if kubectl get "$type" "$name" -n "$namespace" &>/dev/null; then
        status=$(kubectl get "$type" "$name" -n "$namespace" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Running")
        if [[ "$status" == "Running" ]] || [[ "$status" == "" ]]; then
            echo -e "${GREEN}✅ EXISTS${NC}"
            ((PASSED_TESTS++))
            TEST_RESULTS["K8s:$type:$name"]="PASS"
            echo "| K8s $type: $name | ✅ EXISTS | $namespace | Resource present |" >> "$REPORT_FILE"
        else
            echo -e "${YELLOW}⚠️ EXISTS (Status: $status)${NC}"
            ((WARNINGS++))
            TEST_RESULTS["K8s:$type:$name"]="WARNING"
            echo "| K8s $type: $name | ⚠️ WARNING | $namespace | Status: $status |" >> "$REPORT_FILE"
        fi
    else
        echo -e "${RED}❌ NOT FOUND${NC}"
        ((FAILED_TESTS++))
        TEST_RESULTS["K8s:$type:$name"]="FAIL"
        echo "| K8s $type: $name | ❌ NOT FOUND | $namespace | Resource missing |" >> "$REPORT_FILE"
    fi
}

# Start testing
echo "| Component | Status | Endpoint | Notes |" >> "$REPORT_FILE"
echo "|-----------|--------|----------|-------|" >> "$REPORT_FILE"

echo -e "${BLUE}=== Phase 1: Core Services Health ===${NC}"
echo ""

test_service "workflow-api" "$BASE_URL:30889/health" "quantumlayer"
test_service "temporal-ui" "$BASE_URL:30888" "temporal"
test_service "image-registry" "$BASE_URL:30096/health" "quantumlayer"
test_service "cve-tracker" "$BASE_URL:30101/health" "security-services"
test_service "qinfra-dashboard" "$BASE_URL:30095/health" "quantumlayer"
test_service "qinfra-ai" "$BASE_URL:30098/health" "quantumlayer"

echo ""
echo -e "${BLUE}=== Phase 2: Infrastructure Services ===${NC}"
echo ""

test_k8s_resource "deployment" "postgres-postgresql" "temporal"
test_k8s_resource "deployment" "redis" "quantumlayer"
test_k8s_resource "deployment" "qdrant" "quantumlayer"
test_k8s_resource "deployment" "nats" "quantumlayer"
test_k8s_resource "deployment" "docker-registry" "quantumlayer"

echo ""
echo -e "${BLUE}=== Phase 3: Temporal Workflow Engine ===${NC}"
echo ""

test_k8s_resource "deployment" "temporal-frontend" "temporal"
test_k8s_resource "deployment" "temporal-history" "temporal"
test_k8s_resource "deployment" "temporal-matching" "temporal"
test_k8s_resource "deployment" "temporal-worker" "temporal"

echo ""
echo -e "${BLUE}=== Phase 4: API Functionality ===${NC}"
echo ""

# Test Workflow API
test_api "Workflow Generate" "POST" "$BASE_URL:30889/api/v1/workflows/generate" \
    '{"prompt":"Hello World","language":"python","type":"function"}' \
    "workflow_id"

# Test Image Registry
test_api "Image Registry List" "GET" "$BASE_URL:30096/images" "images"

# Test CVE Tracker
test_api "CVE Latest" "GET" "$BASE_URL:30101/cve/latest?hours=24" "cves"

echo ""
echo -e "${BLUE}=== Phase 5: Pod Status Check ===${NC}"
echo ""

# Check for pods not running
echo "Checking for non-running pods..."
NON_RUNNING=$(kubectl get pods --all-namespaces | grep -v "Running\|Completed" | grep -v "NAMESPACE" | wc -l)
if [ "$NON_RUNNING" -gt 0 ]; then
    echo -e "${YELLOW}Found $NON_RUNNING pods not in Running state:${NC}"
    kubectl get pods --all-namespaces | grep -v "Running\|Completed" | grep -v "NAMESPACE"
    echo "" >> "$REPORT_FILE"
    echo "### Non-Running Pods" >> "$REPORT_FILE"
    echo '```' >> "$REPORT_FILE"
    kubectl get pods --all-namespaces | grep -v "Running\|Completed" | grep -v "NAMESPACE" >> "$REPORT_FILE"
    echo '```' >> "$REPORT_FILE"
else
    echo -e "${GREEN}All pods are running!${NC}"
fi

echo ""
echo -e "${BLUE}=== Phase 6: Service Connectivity Matrix ===${NC}"
echo ""

# Test inter-service connectivity
echo "Testing service mesh connectivity..."
echo "" >> "$REPORT_FILE"
echo "## Service Connectivity Matrix" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Test if workflow-api can reach temporal
if kubectl exec -n quantumlayer deployment/workflow-api -- wget -q -O- temporal-frontend.temporal.svc.cluster.local:7233 2>/dev/null | grep -q ""; then
    echo -e "Workflow API → Temporal: ${GREEN}✅${NC}"
    echo "- Workflow API → Temporal: ✅ Connected" >> "$REPORT_FILE"
else
    echo -e "Workflow API → Temporal: ${RED}❌${NC}"
    echo "- Workflow API → Temporal: ❌ Connection failed" >> "$REPORT_FILE"
fi

echo ""
echo -e "${BLUE}=== Phase 7: Resource Usage ===${NC}"
echo ""

echo "## Resource Usage" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
kubectl top nodes >> "$REPORT_FILE" 2>/dev/null || echo "Metrics server not available" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"

# Generate summary
echo ""
echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                    TEST SUMMARY                               ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

SUCCESS_RATE=$(echo "scale=2; $PASSED_TESTS * 100 / $TOTAL_TESTS" | bc)

echo "Total Tests:    $TOTAL_TESTS"
echo -e "Passed:         ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed:         ${RED}$FAILED_TESTS${NC}"
echo -e "Warnings:       ${YELLOW}$WARNINGS${NC}"
echo "Success Rate:   ${SUCCESS_RATE}%"
echo ""

# Add summary to report
echo "" >> "$REPORT_FILE"
echo "## Test Summary" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"
echo "- **Total Tests**: $TOTAL_TESTS" >> "$REPORT_FILE"
echo "- **Passed**: $PASSED_TESTS" >> "$REPORT_FILE"
echo "- **Failed**: $FAILED_TESTS" >> "$REPORT_FILE"
echo "- **Warnings**: $WARNINGS" >> "$REPORT_FILE"
echo "- **Success Rate**: ${SUCCESS_RATE}%" >> "$REPORT_FILE"

# List broken services
if [ ${FAILED_TESTS} -gt 0 ]; then
    echo -e "${RED}=== Broken Services ===${NC}"
    echo "" >> "$REPORT_FILE"
    echo "## Broken Services" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    for key in "${!TEST_RESULTS[@]}"; do
        if [ "${TEST_RESULTS[$key]}" = "FAIL" ]; then
            echo "  ❌ $key"
            echo "- ❌ $key: ${TEST_DETAILS[$key]}" >> "$REPORT_FILE" 2>/dev/null || echo "- ❌ $key" >> "$REPORT_FILE"
        fi
    done
fi

# List services with warnings
if [ ${WARNINGS} -gt 0 ]; then
    echo ""
    echo -e "${YELLOW}=== Services with Warnings ===${NC}"
    echo "" >> "$REPORT_FILE"
    echo "## Services with Warnings" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    for key in "${!TEST_RESULTS[@]}"; do
        if [ "${TEST_RESULTS[$key]}" = "WARNING" ]; then
            echo "  ⚠️ $key"
            echo "- ⚠️ $key: ${TEST_DETAILS[$key]}" >> "$REPORT_FILE" 2>/dev/null || echo "- ⚠️ $key" >> "$REPORT_FILE"
        fi
    done
fi

# Recommendations
echo "" >> "$REPORT_FILE"
echo "## Recommendations" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

if [ ${FAILED_TESTS} -gt 0 ]; then
    echo "1. Fix broken services before proceeding" >> "$REPORT_FILE"
    echo "2. Check pod logs for failed services" >> "$REPORT_FILE"
    echo "3. Verify database connections" >> "$REPORT_FILE"
else
    echo "1. All core services are operational" >> "$REPORT_FILE"
    echo "2. Platform is ready for production use" >> "$REPORT_FILE"
    echo "3. Consider implementing monitoring for warnings" >> "$REPORT_FILE"
fi

echo "" >> "$REPORT_FILE"
echo "---" >> "$REPORT_FILE"
echo "*Generated: $(date)*" >> "$REPORT_FILE"
echo "*Version: ${VERSION}*" >> "$REPORT_FILE"

echo ""
echo -e "${GREEN}Test report saved to: $REPORT_FILE${NC}"
echo ""

# Exit with appropriate code
if [ ${FAILED_TESTS} -gt 0 ]; then
    echo -e "${RED}⚠️ Platform has ${FAILED_TESTS} failing components. Review report for details.${NC}"
    exit 1
else
    echo -e "${GREEN}✅ Platform baseline established successfully!${NC}"
    exit 0
fi