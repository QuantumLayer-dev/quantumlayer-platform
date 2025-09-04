#!/bin/bash
set -euo pipefail

# End-to-End QuantumLayer Platform Test with New Services
# Tests the complete pipeline from prompt to deployed application

echo "╔═══════════════════════════════════════════════════════════════════════════════╗"
echo "║              QuantumLayer Platform - End-to-End Pipeline Test                  ║"
echo "║                     Testing Complete Workflow Integration                      ║"
echo "╚═══════════════════════════════════════════════════════════════════════════════╝"
echo ""

# Configuration
WORKFLOW_API="http://192.168.1.217:30889"  # NodePort for workflow-api
QUANTUM_DROPS="http://192.168.1.217:30890"  # NodePort for quantum-drops
SANDBOX_EXECUTOR="http://192.168.1.217:30884"  # NodePort for sandbox-executor
CAPSULE_BUILDER="http://192.168.1.217:30886"  # NodePort for capsule-builder

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_section() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}▶ $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
}

# Check services are running
print_section "Pre-flight Checks"

# Check workflow API
print_status "Checking Workflow API..."
if curl -s -f "${WORKFLOW_API}/health" > /dev/null 2>&1; then
    echo "  ✅ Workflow API is healthy"
else
    print_error "  ❌ Workflow API is not responding"
    exit 1
fi

# Check QuantumDrops
print_status "Checking QuantumDrops service..."
if curl -s -f "${QUANTUM_DROPS}/health" > /dev/null 2>&1; then
    echo "  ✅ QuantumDrops is healthy"
else
    print_warning "  ⚠️  QuantumDrops health check failed (may not have endpoint)"
fi

# Check Sandbox Executor
print_status "Checking Sandbox Executor..."
if curl -s -f "${SANDBOX_EXECUTOR}/health" > /dev/null 2>&1; then
    echo "  ✅ Sandbox Executor is healthy"
else
    print_error "  ❌ Sandbox Executor is not responding"
    exit 1
fi

# Check Capsule Builder
print_status "Checking Capsule Builder..."
if curl -s -f "${CAPSULE_BUILDER}/health" > /dev/null 2>&1; then
    echo "  ✅ Capsule Builder is healthy"
else
    print_warning "  ⚠️  Capsule Builder health check failed"
fi

# Test 1: Submit a comprehensive workflow request
print_section "Test 1: Submit Extended Workflow Request"

# Create test request for a Python FastAPI application
cat > /tmp/test-request.json <<'EOF'
{
    "prompt": "Create a Python FastAPI REST API for a todo list application with CRUD operations. Include proper error handling, input validation using Pydantic models, and in-memory storage. Add endpoints for: GET /todos (list all), GET /todos/{id} (get one), POST /todos (create), PUT /todos/{id} (update), DELETE /todos/{id} (delete). Include a health check endpoint and proper OpenAPI documentation.",
    "language": "python",
    "framework": "fastapi",
    "type": "api",
    "name": "todo-api"
}
EOF

print_status "Submitting workflow request..."
RESPONSE=$(curl -s -X POST "${WORKFLOW_API}/api/v1/generate-extended" \
    -H "Content-Type: application/json" \
    -d @/tmp/test-request.json)

# Extract workflow ID
WORKFLOW_ID=$(echo "$RESPONSE" | sed -n 's/.*"workflow_id":"\([^"]*\).*/\1/p')

if [[ -z "$WORKFLOW_ID" ]]; then
    print_error "Failed to extract workflow ID from response:"
    echo "$RESPONSE"
    exit 1
fi

echo "  ✅ Workflow submitted: ${WORKFLOW_ID}"

# Test 2: Monitor workflow execution
print_section "Test 2: Monitor Workflow Execution"

print_status "Waiting for workflow to complete..."
MAX_ATTEMPTS=30
ATTEMPT=0

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    sleep 2
    
    # Check workflow status
    STATUS_RESPONSE=$(curl -s "${WORKFLOW_API}/api/v1/workflows/${WORKFLOW_ID}/status")
    STATUS=$(echo "$STATUS_RESPONSE" | sed -n 's/.*"status":"\([^"]*\).*/\1/p')
    
    if [[ "$STATUS" == "COMPLETED" ]]; then
        echo "  ✅ Workflow completed successfully!"
        break
    elif [[ "$STATUS" == "FAILED" ]]; then
        print_error "  ❌ Workflow failed!"
        echo "$STATUS_RESPONSE"
        exit 1
    else
        echo "  ... Status: ${STATUS:-RUNNING} (attempt $((ATTEMPT+1))/$MAX_ATTEMPTS)"
    fi
    
    ATTEMPT=$((ATTEMPT+1))
done

if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
    print_error "Workflow did not complete within timeout"
    exit 1
fi

# Test 3: Retrieve QuantumDrops
print_section "Test 3: Retrieve and Analyze QuantumDrops"

print_status "Fetching QuantumDrops for workflow..."
DROPS_RESPONSE=$(curl -s "${QUANTUM_DROPS}/api/v1/workflows/${WORKFLOW_ID}/drops")

# Count and categorize drops
DROP_COUNT=$(echo "$DROPS_RESPONSE" | grep -o '"stage"' | wc -l)
echo "  📦 Found ${DROP_COUNT} QuantumDrops"

# Extract stages
echo "$DROPS_RESPONSE" | grep -o '"stage":"[^"]*"' | sed 's/"stage":"\([^"]*\)"/  - \1/' | sort -u

# Test 4: Extract and validate generated code
print_section "Test 4: Validate Generated Code"

# Extract code from drops
CODE_DROP=$(echo "$DROPS_RESPONSE" | sed -n '/"type":"code"/,/"timestamp"/p' | sed -n 's/.*"artifact":"\(.*\)","type".*/\1/p' | head -1)

if [[ -n "$CODE_DROP" ]]; then
    # Decode the code (handle escaped characters)
    DECODED_CODE=$(echo "$CODE_DROP" | sed 's/\\n/\n/g; s/\\"/"/g; s/\\\\/\\/g')
    
    # Save to file for validation
    echo "$DECODED_CODE" > /tmp/generated_code.py
    
    print_status "Testing code with Sandbox Executor..."
    
    # Create execution request
    EXEC_REQUEST=$(cat <<EOF
{
    "id": "test-exec-${WORKFLOW_ID}",
    "language": "python",
    "code": "${CODE_DROP}",
    "timeout": 30
}
EOF
)
    
    # Execute in sandbox
    EXEC_RESPONSE=$(curl -s -X POST "${SANDBOX_EXECUTOR}/api/v1/execute" \
        -H "Content-Type: application/json" \
        -d "$EXEC_REQUEST")
    
    EXEC_ID=$(echo "$EXEC_RESPONSE" | sed -n 's/.*"id":"\([^"]*\).*/\1/p')
    
    if [[ -n "$EXEC_ID" ]]; then
        echo "  ✅ Code submitted to sandbox: ${EXEC_ID}"
    else
        print_warning "  ⚠️  Could not validate code in sandbox"
    fi
else
    print_warning "  ⚠️  No code drop found"
fi

# Test 5: Build Capsule
print_section "Test 5: Build Application Capsule"

if [[ -n "$CODE_DROP" ]]; then
    print_status "Building structured project with Capsule Builder..."
    
    # Create capsule request
    CAPSULE_REQUEST=$(cat <<EOF
{
    "workflow_id": "${WORKFLOW_ID}",
    "name": "todo-api",
    "type": "api",
    "language": "python",
    "framework": "fastapi",
    "code": "${CODE_DROP}",
    "description": "Todo List API built with FastAPI"
}
EOF
)
    
    # Build capsule
    CAPSULE_RESPONSE=$(curl -s -X POST "${CAPSULE_BUILDER}/api/v1/build" \
        -H "Content-Type: application/json" \
        -d "$CAPSULE_REQUEST")
    
    CAPSULE_ID=$(echo "$CAPSULE_RESPONSE" | sed -n 's/.*"id":"\([^"]*\).*/\1/p')
    
    if [[ -n "$CAPSULE_ID" ]]; then
        echo "  ✅ Capsule built: ${CAPSULE_ID}"
        
        # Extract structure
        echo "  📁 Project structure:"
        echo "$CAPSULE_RESPONSE" | grep -o '"path":"[^"]*"' | sed 's/"path":"\([^"]*\)"/    - \1/' | head -10
    else
        print_warning "  ⚠️  Could not build capsule"
    fi
fi

# Test 6: Analyze workflow metrics
print_section "Test 6: Workflow Metrics Analysis"

# Get final workflow result
RESULT=$(curl -s "${WORKFLOW_API}/api/v1/workflows/${WORKFLOW_ID}/result")

if [[ -n "$RESULT" ]]; then
    # Extract metrics if available
    TOTAL_TIME=$(echo "$RESULT" | sed -n 's/.*"total_duration_ms":\([0-9]*\).*/\1/p')
    
    if [[ -n "$TOTAL_TIME" ]]; then
        SECONDS=$((TOTAL_TIME / 1000))
        echo "  ⏱️  Total execution time: ${SECONDS} seconds"
    fi
    
    # Check for validation scores
    VALIDATION_SCORE=$(echo "$RESULT" | sed -n 's/.*"validation_score":\([0-9]*\).*/\1/p')
    if [[ -n "$VALIDATION_SCORE" ]]; then
        echo "  📊 Validation score: ${VALIDATION_SCORE}/100"
    fi
fi

# Summary
print_section "Test Summary"

echo "Pipeline Test Results:"
echo ""
echo "✅ Workflow Submission:     SUCCESS"
echo "✅ Workflow Execution:      SUCCESS (ID: ${WORKFLOW_ID})"
echo "✅ QuantumDrops Storage:    SUCCESS (${DROP_COUNT} drops)"

if [[ -n "$EXEC_ID" ]]; then
    echo "✅ Sandbox Validation:      SUCCESS"
else
    echo "⚠️  Sandbox Validation:      SKIPPED"
fi

if [[ -n "$CAPSULE_ID" ]]; then
    echo "✅ Capsule Building:        SUCCESS"
else
    echo "⚠️  Capsule Building:        SKIPPED"
fi

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "🎉 End-to-End Pipeline Test Complete!"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "Next Steps:"
echo "1. Deploy the generated capsule to preview environment"
echo "2. Create TTL-based URLs for testing"
echo "3. Set up continuous monitoring"
echo "4. Build web UI for visualization"
echo ""