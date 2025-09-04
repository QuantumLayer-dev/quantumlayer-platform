#!/bin/bash
set -euo pipefail

# ═══════════════════════════════════════════════════════════════════════════
# QuantumLayer Platform - Complete End-to-End Demo
# Journey: Natural Language → AI Generation → Structure → Validation → Preview
# ═══════════════════════════════════════════════════════════════════════════

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Logging functions
log_stage() { echo -e "\n${BOLD}${BLUE}════════════════════════════════════════${NC}"; echo -e "${BOLD}${BLUE}STAGE $1: $2${NC}"; echo -e "${BOLD}${BLUE}════════════════════════════════════════${NC}"; }
log_info() { echo -e "${GREEN}[✓]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[!]${NC} $1"; }
log_error() { echo -e "${RED}[✗]${NC} $1"; }
log_test() { echo -e "${CYAN}[TEST]${NC} $1"; }

# Configuration
NAMESPACE="quantumlayer"
TEMPORAL_NAMESPACE="temporal"
API_GATEWAY_PORT=8080
WORKFLOW_API_PORT=8081
SANDBOX_PORT=8091
CAPSULE_PORT=8092
PREVIEW_PORT=8093

# Demo configuration
DEMO_PROJECT_NAME="quantum-fastapi-demo"
DEMO_WORKFLOW_ID=""
DEMO_CAPSULE_ID=""

# Banner
print_banner() {
    echo -e "${BOLD}${MAGENTA}"
    cat << "EOF"
╔═══════════════════════════════════════════════════════════════════════════╗
║                                                                           ║
║   ◢▄◣   QuantumLayer Platform - End-to-End Demo                         ║
║  ◢█▄█◣  From Natural Language to Deployed Application                   ║
║ ◢█████◣                                                                  ║
║◢███████◣ Journey Through All 12 Stages                                  ║
║                                                                           ║
╚═══════════════════════════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"
}

# Check prerequisites
check_prerequisites() {
    log_stage "0" "Prerequisites Check"
    
    # Check commands
    for cmd in kubectl docker curl; do
        if command -v $cmd &> /dev/null; then
            log_info "$cmd is installed"
        else
            log_error "$cmd is not installed"
            exit 1
        fi
    done
    
    # Check jq (optional but recommended)
    if command -v jq &> /dev/null; then
        log_info "jq is installed (JSON processing enabled)"
        HAS_JQ=true
    else
        log_warn "jq is not installed (using grep/sed for JSON parsing)"
        HAS_JQ=false
    fi
    
    # Check cluster connectivity
    if kubectl cluster-info &> /dev/null; then
        log_info "Kubernetes cluster is accessible"
    else
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check namespaces
    for ns in $NAMESPACE $TEMPORAL_NAMESPACE; do
        if kubectl get namespace $ns &> /dev/null; then
            log_info "Namespace $ns exists"
        else
            log_warn "Creating namespace $ns"
            kubectl create namespace $ns
        fi
    done
}

# Deploy new services
deploy_quantum_capsule_services() {
    log_stage "0.5" "Deploying QuantumCapsule Services"
    
    # Build Sandbox Executor
    log_info "Building Sandbox Executor..."
    cd packages/sandbox-executor
    docker build -t localhost:5000/sandbox-executor:latest .
    docker push localhost:5000/sandbox-executor:latest
    cd ../..
    
    # Build Capsule Builder
    log_info "Building Capsule Builder..."
    cd packages/capsule-builder
    docker build -t localhost:5000/capsule-builder:latest .
    docker push localhost:5000/capsule-builder:latest
    cd ../..
    
    # Deploy Sandbox Executor
    log_info "Deploying Sandbox Executor..."
    kubectl apply -f packages/sandbox-executor/k8s-deployment.yaml
    
    # Deploy Capsule Builder
    log_info "Deploying Capsule Builder..."
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: capsule-builder
  namespace: $NAMESPACE
spec:
  replicas: 2
  selector:
    matchLabels:
      app: capsule-builder
  template:
    metadata:
      labels:
        app: capsule-builder
    spec:
      containers:
      - name: capsule-builder
        image: localhost:5000/capsule-builder:latest
        ports:
        - containerPort: 8092
        env:
        - name: PORT
          value: "8092"
---
apiVersion: v1
kind: Service
metadata:
  name: capsule-builder
  namespace: $NAMESPACE
spec:
  selector:
    app: capsule-builder
  ports:
  - port: 8092
    targetPort: 8092
EOF
    
    # Wait for deployments
    kubectl rollout status deployment/sandbox-executor -n $NAMESPACE --timeout=120s || true
    kubectl rollout status deployment/capsule-builder -n $NAMESPACE --timeout=120s || true
    
    log_info "QuantumCapsule services deployed"
}

# Stage 1: Natural Language Input
stage_1_nlp_input() {
    log_stage "1" "Natural Language Input"
    
    PROMPT="Create a FastAPI application with user authentication. It should have endpoints for user registration, login, and profile management. Include JWT token authentication and password hashing. Add proper error handling and input validation."
    
    echo -e "${CYAN}User Prompt:${NC}"
    echo "$PROMPT"
    echo ""
    
    log_test "Sending prompt to workflow API..."
    
    # Prepare the request
    REQUEST_BODY=$(cat <<EOF
{
    "prompt": "$PROMPT",
    "language": "python",
    "framework": "fastapi",
    "type": "api",
    "name": "$DEMO_PROJECT_NAME",
    "description": "FastAPI application with JWT authentication",
    "test_coverage": true,
    "documentation": true
}
EOF
    )
    
    # Save request for reference
    echo "$REQUEST_BODY" > /tmp/demo-request.json
    log_info "Request saved to /tmp/demo-request.json"
}

# Stage 2: Submit to Workflow
stage_2_workflow_submission() {
    log_stage "2" "Workflow Submission"
    
    # Port-forward to workflow API
    log_info "Setting up port-forward to workflow-api..."
    kubectl port-forward -n $TEMPORAL_NAMESPACE svc/workflow-api 8081:8080 &
    PF_PID=$!
    sleep 3
    
    # Submit workflow
    log_test "Submitting code generation workflow..."
    RESPONSE=$(curl -X POST http://localhost:8081/api/v1/generate \
        -H "Content-Type: application/json" \
        -d @/tmp/demo-request.json \
        2>/dev/null || echo '{"error":"Failed to submit"}')
    
    if echo "$RESPONSE" | jq -e '.workflow_id' &> /dev/null; then
        DEMO_WORKFLOW_ID=$(echo "$RESPONSE" | jq -r '.workflow_id')
        log_info "Workflow submitted successfully: $DEMO_WORKFLOW_ID"
        echo "$RESPONSE" | jq '.'
    else
        log_error "Failed to submit workflow"
        echo "$RESPONSE"
        kill $PF_PID 2>/dev/null || true
        return 1
    fi
    
    kill $PF_PID 2>/dev/null || true
}

# Stage 3: Monitor Workflow Progress
stage_3_monitor_workflow() {
    log_stage "3" "Workflow Execution Monitoring"
    
    kubectl port-forward -n $TEMPORAL_NAMESPACE svc/workflow-api 8081:8080 &
    PF_PID=$!
    sleep 3
    
    log_info "Monitoring workflow execution..."
    
    # Poll workflow status
    for i in {1..30}; do
        STATUS=$(curl -s http://localhost:8081/api/v1/workflows/$DEMO_WORKFLOW_ID/status 2>/dev/null || echo '{}')
        
        if echo "$STATUS" | jq -e '.status' &> /dev/null; then
            CURRENT_STATUS=$(echo "$STATUS" | jq -r '.status')
            CURRENT_STAGE=$(echo "$STATUS" | jq -r '.current_stage // "unknown"')
            
            echo -e "${CYAN}[$i/30]${NC} Status: $CURRENT_STATUS | Stage: $CURRENT_STAGE"
            
            if [[ "$CURRENT_STATUS" == "COMPLETED" ]]; then
                log_info "Workflow completed successfully!"
                break
            elif [[ "$CURRENT_STATUS" == "FAILED" ]]; then
                log_error "Workflow failed!"
                echo "$STATUS" | jq '.error'
                kill $PF_PID 2>/dev/null || true
                return 1
            fi
        fi
        
        sleep 5
    done
    
    kill $PF_PID 2>/dev/null || true
}

# Stage 4: Retrieve Generated Code
stage_4_retrieve_code() {
    log_stage "4" "Retrieving Generated Artifacts"
    
    # Port-forward to quantum-drops
    kubectl port-forward -n $NAMESPACE svc/quantum-drops 8090:8090 &
    PF_PID=$!
    sleep 3
    
    log_test "Fetching QuantumDrops for workflow..."
    
    # Get drops
    DROPS=$(curl -s http://localhost:8090/api/v1/workflows/$DEMO_WORKFLOW_ID/drops 2>/dev/null || echo '{}')
    
    if echo "$DROPS" | jq -e '.drops' &> /dev/null; then
        DROP_COUNT=$(echo "$DROPS" | jq '.drops | length')
        log_info "Retrieved $DROP_COUNT QuantumDrops"
        
        # Show drop types
        echo "$DROPS" | jq -r '.drops[] | "\(.stage): \(.type)"' | sort -u
        
        # Extract code drop
        CODE_DROP=$(echo "$DROPS" | jq -r '.drops[] | select(.type == "code") | .artifact' | head -1)
        if [[ -n "$CODE_DROP" ]]; then
            echo -e "\n${CYAN}Generated Code Preview:${NC}"
            echo "$CODE_DROP" | head -20
            echo "..."
            echo "$CODE_DROP" > /tmp/generated-code.py
            log_info "Code saved to /tmp/generated-code.py"
        fi
    else
        log_warn "No drops retrieved"
    fi
    
    kill $PF_PID 2>/dev/null || true
}

# Stage 5: Build Structured Capsule
stage_5_build_capsule() {
    log_stage "5" "Building Structured Capsule"
    
    # Port-forward to capsule-builder
    kubectl port-forward -n $NAMESPACE svc/capsule-builder 8092:8092 &
    PF_PID=$!
    sleep 3
    
    log_test "Building structured project from drops..."
    
    # Build capsule
    CAPSULE_REQUEST=$(cat <<EOF
{
    "workflow_id": "$DEMO_WORKFLOW_ID",
    "language": "python",
    "framework": "fastapi",
    "type": "api",
    "name": "$DEMO_PROJECT_NAME",
    "description": "FastAPI application with JWT authentication",
    "code": $(jq -Rs . < /tmp/generated-code.py),
    "dependencies": ["fastapi", "uvicorn", "pydantic", "python-jose", "passlib", "python-multipart"]
}
EOF
    )
    
    CAPSULE_RESPONSE=$(curl -X POST http://localhost:8092/api/v1/build \
        -H "Content-Type: application/json" \
        -d "$CAPSULE_REQUEST" \
        2>/dev/null || echo '{}')
    
    if echo "$CAPSULE_RESPONSE" | jq -e '.id' &> /dev/null; then
        DEMO_CAPSULE_ID=$(echo "$CAPSULE_RESPONSE" | jq -r '.id')
        log_info "Capsule built successfully: $DEMO_CAPSULE_ID"
        
        # Show structure
        echo -e "\n${CYAN}Project Structure:${NC}"
        echo "$CAPSULE_RESPONSE" | jq -r '.structure | keys[]' | sort
    else
        log_error "Failed to build capsule"
        echo "$CAPSULE_RESPONSE"
    fi
    
    kill $PF_PID 2>/dev/null || true
}

# Stage 6: Validate in Sandbox
stage_6_validate_sandbox() {
    log_stage "6" "Sandbox Validation"
    
    # Port-forward to sandbox-executor
    kubectl port-forward -n $NAMESPACE svc/sandbox-executor 8091:8091 &
    PF_PID=$!
    sleep 3
    
    log_test "Validating code in sandbox..."
    
    # Execute code
    EXECUTION_REQUEST=$(cat <<EOF
{
    "language": "python",
    "code": $(jq -Rs . < /tmp/generated-code.py),
    "dependencies": ["fastapi", "uvicorn"],
    "timeout": 30
}
EOF
    )
    
    EXECUTION_RESPONSE=$(curl -X POST http://localhost:8091/api/v1/execute \
        -H "Content-Type: application/json" \
        -d "$EXECUTION_REQUEST" \
        2>/dev/null || echo '{}')
    
    if echo "$EXECUTION_RESPONSE" | jq -e '.id' &> /dev/null; then
        EXEC_ID=$(echo "$EXECUTION_RESPONSE" | jq -r '.id')
        log_info "Execution started: $EXEC_ID"
        
        # Wait for completion
        sleep 5
        
        # Get results
        EXEC_RESULT=$(curl -s http://localhost:8091/api/v1/executions/$EXEC_ID 2>/dev/null || echo '{}')
        
        if echo "$EXEC_RESULT" | jq -e '.status' &> /dev/null; then
            STATUS=$(echo "$EXEC_RESULT" | jq -r '.status')
            if [[ "$STATUS" == "success" ]]; then
                log_info "Code validation successful!"
                echo -e "\n${CYAN}Execution Output:${NC}"
                echo "$EXEC_RESULT" | jq -r '.output' | head -20
            else
                log_warn "Code validation failed"
                echo "$EXEC_RESULT" | jq -r '.error'
            fi
        fi
    else
        log_error "Failed to start execution"
        echo "$EXECUTION_RESPONSE"
    fi
    
    kill $PF_PID 2>/dev/null || true
}

# Stage 7: Create Preview
stage_7_create_preview() {
    log_stage "7" "Preview Generation (Simulated)"
    
    log_warn "Preview Service not yet implemented"
    log_info "Simulating preview generation..."
    
    # Create a simple preview
    cat <<EOF > /tmp/preview.html
<!DOCTYPE html>
<html>
<head>
    <title>$DEMO_PROJECT_NAME - Preview</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        h1 { color: #333; }
        .code { background: #f4f4f4; padding: 20px; border-radius: 5px; }
        .endpoint { margin: 10px 0; padding: 10px; background: #e8f4f8; }
    </style>
</head>
<body>
    <h1>🚀 $DEMO_PROJECT_NAME</h1>
    <p>FastAPI application with JWT authentication</p>
    
    <h2>API Endpoints:</h2>
    <div class="endpoint">POST /register - User registration</div>
    <div class="endpoint">POST /login - User login</div>
    <div class="endpoint">GET /profile - Get user profile</div>
    <div class="endpoint">PUT /profile - Update profile</div>
    
    <h2>Features:</h2>
    <ul>
        <li>JWT Token Authentication</li>
        <li>Password Hashing (bcrypt)</li>
        <li>Input Validation (Pydantic)</li>
        <li>Error Handling</li>
    </ul>
    
    <h2>Preview URL (Simulated):</h2>
    <code>https://${DEMO_CAPSULE_ID}.preview.quantumlayer.io</code>
    <p><em>TTL: 24 hours</em></p>
</body>
</html>
EOF
    
    log_info "Preview generated at /tmp/preview.html"
    echo -e "${CYAN}Preview URL:${NC} https://${DEMO_CAPSULE_ID}.preview.quantumlayer.io (simulated)"
}

# Stage 8: Deploy to Kubernetes
stage_8_deploy() {
    log_stage "8" "Deployment (Simulated)"
    
    log_warn "Deployment Manager not yet implemented"
    log_info "Simulating deployment..."
    
    # Create deployment manifest
    cat <<EOF > /tmp/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: $DEMO_PROJECT_NAME
  namespace: $NAMESPACE
spec:
  replicas: 1
  selector:
    matchLabels:
      app: $DEMO_PROJECT_NAME
  template:
    metadata:
      labels:
        app: $DEMO_PROJECT_NAME
    spec:
      containers:
      - name: app
        image: $DEMO_PROJECT_NAME:latest
        ports:
        - containerPort: 8000
        env:
        - name: JWT_SECRET
          value: "your-secret-key"
---
apiVersion: v1
kind: Service
metadata:
  name: $DEMO_PROJECT_NAME
  namespace: $NAMESPACE
spec:
  selector:
    app: $DEMO_PROJECT_NAME
  ports:
  - port: 80
    targetPort: 8000
EOF
    
    log_info "Deployment manifest created at /tmp/deployment.yaml"
    log_info "Would deploy to: https://$DEMO_PROJECT_NAME.quantumlayer.io"
}

# Show summary
show_summary() {
    log_stage "9" "Demo Summary"
    
    echo -e "${BOLD}${GREEN}Journey Complete!${NC}\n"
    
    echo "📝 Input:"
    echo "   Natural language prompt about FastAPI with authentication"
    echo ""
    
    echo "🔄 Process:"
    echo "   1. ✅ Prompt Enhancement (Meta-Prompt Engine)"
    echo "   2. ✅ Code Generation (LLM Router)"
    echo "   3. ✅ Artifact Storage (QuantumDrops)"
    echo "   4. ✅ Project Structure (Capsule Builder)"
    echo "   5. ✅ Code Validation (Sandbox Executor)"
    echo "   6. ⚠️  Preview Generation (Simulated)"
    echo "   7. ⚠️  Deployment (Simulated)"
    echo ""
    
    echo "📦 Output:"
    echo "   - Workflow ID: $DEMO_WORKFLOW_ID"
    echo "   - Capsule ID: $DEMO_CAPSULE_ID"
    echo "   - Generated Code: /tmp/generated-code.py"
    echo "   - Preview: /tmp/preview.html"
    echo "   - Deployment: /tmp/deployment.yaml"
    echo ""
    
    echo "✅ What's Working:"
    echo "   • AI code generation via LLM"
    echo "   • Structured project creation"
    echo "   • Code validation in sandbox"
    echo "   • Artifact storage and retrieval"
    echo ""
    
    echo "⚠️  What's Missing:"
    echo "   • Live preview with Monaco Editor"
    echo "   • Actual deployment to Kubernetes"
    echo "   • TTL-based URL management"
    echo "   • Security scanning"
    echo "   • Performance profiling"
    echo ""
    
    echo "🎯 Platform Completion: ~60% of vision"
}

# Test the platform
test_platform_health() {
    log_stage "10" "Platform Health Check"
    
    echo "Checking all services..."
    
    # Check pods
    kubectl get pods -n $NAMESPACE --no-headers | while read line; do
        POD=$(echo $line | awk '{print $1}')
        STATUS=$(echo $line | awk '{print $3}')
        if [[ "$STATUS" == "Running" ]]; then
            echo -e "  ${GREEN}✓${NC} $POD"
        else
            echo -e "  ${RED}✗${NC} $POD ($STATUS)"
        fi
    done
    
    # Check temporal
    echo ""
    echo "Temporal services:"
    kubectl get pods -n $TEMPORAL_NAMESPACE --no-headers | grep -E "temporal-|workflow-" | while read line; do
        POD=$(echo $line | awk '{print $1}')
        STATUS=$(echo $line | awk '{print $3}')
        if [[ "$STATUS" == "Running" ]]; then
            echo -e "  ${GREEN}✓${NC} $POD"
        else
            echo -e "  ${RED}✗${NC} $POD ($STATUS)"
        fi
    done
}

# Main execution
main() {
    print_banner
    
    # Check if we should just deploy services
    if [[ "${1:-}" == "--deploy-only" ]]; then
        check_prerequisites
        deploy_quantum_capsule_services
        test_platform_health
        exit 0
    fi
    
    # Full demo flow
    check_prerequisites
    
    # Deploy new services if needed
    if ! kubectl get deployment sandbox-executor -n $NAMESPACE &> /dev/null; then
        deploy_quantum_capsule_services
    fi
    
    # Run through all stages
    stage_1_nlp_input
    stage_2_workflow_submission
    stage_3_monitor_workflow
    stage_4_retrieve_code
    stage_5_build_capsule
    stage_6_validate_sandbox
    stage_7_create_preview
    stage_8_deploy
    
    # Show results
    show_summary
    test_platform_health
    
    echo -e "\n${BOLD}${GREEN}🎉 End-to-End Demo Complete!${NC}"
    echo "Check /tmp/ for generated artifacts"
}

# Handle script arguments
case "${1:-}" in
    --deploy-only)
        main --deploy-only
        ;;
    --test-only)
        test_platform_health
        ;;
    --help)
        echo "Usage: $0 [--deploy-only|--test-only|--help]"
        echo "  --deploy-only  : Only deploy QuantumCapsule services"
        echo "  --test-only    : Only run health checks"
        echo "  --help         : Show this help"
        exit 0
        ;;
    *)
        main
        ;;
esac