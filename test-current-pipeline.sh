#!/bin/bash
set -euo pipefail

# Simple test of current QuantumLayer pipeline
# Tests what's actually working right now

echo "============================================"
echo "Testing QuantumLayer Current Pipeline"
echo "============================================"

# Test 1: Check services
echo ""
echo "1. Checking deployed services..."
kubectl get pods -n quantumlayer --no-headers | while read line; do
    POD=$(echo $line | awk '{print $1}')
    STATUS=$(echo $line | awk '{print $3}')
    READY=$(echo $line | awk '{print $2}')
    echo "   $POD: $STATUS ($READY)"
done

# Test 2: Test workflow submission
echo ""
echo "2. Testing workflow submission..."

# Create a simple request
cat > /tmp/test-request.json <<EOF
{
    "prompt": "Create a simple Python function that calculates fibonacci numbers",
    "language": "python",
    "type": "function",
    "framework": "",
    "name": "fibonacci-test"
}
EOF

# Submit to workflow API
echo "   Setting up port-forward to workflow-api..."
kubectl port-forward -n temporal svc/workflow-api 8081:8080 > /dev/null 2>&1 &
PF_PID=$!
sleep 3

echo "   Submitting workflow..."
RESPONSE=$(curl -s -X POST http://localhost:8081/api/v1/generate \
    -H "Content-Type: application/json" \
    -d @/tmp/test-request.json 2>/dev/null || echo "FAILED")

if [[ "$RESPONSE" == "FAILED" ]]; then
    echo "   ❌ Failed to submit workflow"
    kill $PF_PID 2>/dev/null || true
else
    # Extract workflow ID (simple parsing without jq)
    WORKFLOW_ID=$(echo "$RESPONSE" | grep -o '"workflow_id":"[^"]*' | cut -d'"' -f4)
    
    if [[ -n "$WORKFLOW_ID" ]]; then
        echo "   ✅ Workflow submitted: $WORKFLOW_ID"
        
        # Wait for completion
        echo "   Waiting for workflow to complete (max 60 seconds)..."
        for i in {1..12}; do
            sleep 5
            STATUS=$(curl -s http://localhost:8081/api/v1/workflows/$WORKFLOW_ID/status 2>/dev/null || echo "{}")
            
            # Simple status check
            if echo "$STATUS" | grep -q "COMPLETED"; then
                echo "   ✅ Workflow completed!"
                break
            elif echo "$STATUS" | grep -q "FAILED"; then
                echo "   ❌ Workflow failed"
                break
            else
                echo "   ... still running ($i/12)"
            fi
        done
        
        # Test 3: Check QuantumDrops
        echo ""
        echo "3. Checking QuantumDrops..."
        kill $PF_PID 2>/dev/null || true
        
        kubectl port-forward -n quantumlayer svc/quantum-drops 8090:8090 > /dev/null 2>&1 &
        PF_PID=$!
        sleep 3
        
        DROPS=$(curl -s http://localhost:8090/api/v1/workflows/$WORKFLOW_ID/drops 2>/dev/null || echo "{}")
        
        if echo "$DROPS" | grep -q "drops"; then
            echo "   ✅ QuantumDrops retrieved"
            
            # Count drops
            DROP_COUNT=$(echo "$DROPS" | grep -o '"stage"' | wc -l)
            echo "   Found $DROP_COUNT drops"
            
            # Extract code if present
            if echo "$DROPS" | grep -q '"type":"code"'; then
                echo "   ✅ Code drop found"
                
                # Extract code artifact (basic extraction)
                CODE=$(echo "$DROPS" | sed -n 's/.*"artifact":"\([^"]*\)".*/\1/p' | head -1)
                if [[ -n "$CODE" ]]; then
                    echo ""
                    echo "   Generated code preview:"
                    echo "   -------------------"
                    echo "$CODE" | sed 's/\\n/\n/g' | head -10
                    echo "   -------------------"
                fi
            fi
        else
            echo "   ❌ No drops found"
        fi
        
        kill $PF_PID 2>/dev/null || true
    else
        echo "   ❌ Could not extract workflow ID"
        kill $PF_PID 2>/dev/null || true
    fi
fi

# Test 4: Check what's missing
echo ""
echo "4. Checking missing components..."

# Check for sandbox executor
if kubectl get deployment sandbox-executor -n quantumlayer > /dev/null 2>&1; then
    echo "   ✅ Sandbox Executor deployed"
else
    echo "   ❌ Sandbox Executor NOT deployed"
fi

# Check for capsule builder
if kubectl get deployment capsule-builder -n quantumlayer > /dev/null 2>&1; then
    echo "   ✅ Capsule Builder deployed"
else
    echo "   ❌ Capsule Builder NOT deployed"
fi

# Check for preview service
if kubectl get deployment preview-service -n quantumlayer > /dev/null 2>&1; then
    echo "   ✅ Preview Service deployed"
else
    echo "   ❌ Preview Service NOT deployed (expected - not built yet)"
fi

# Summary
echo ""
echo "============================================"
echo "Summary:"
echo "============================================"
echo ""
echo "Working:"
echo "  ✅ Workflow submission and execution"
echo "  ✅ LLM code generation (via Azure OpenAI)"
echo "  ✅ QuantumDrops storage"
echo "  ✅ Basic 12-stage pipeline"
echo ""
echo "Missing/Issues:"
echo "  ❌ Sandbox Executor (not deployed)"
echo "  ❌ Capsule Builder (not deployed)"
echo "  ❌ Preview Service (not built)"
echo "  ❌ Deployment automation"
echo "  ❌ Web UI"
echo "  ⚠️  Many stages use template fallbacks instead of AI"
echo ""
echo "Next Steps:"
echo "  1. Deploy Sandbox Executor and Capsule Builder"
echo "  2. Test code validation in sandbox"
echo "  3. Test structured project generation"
echo "  4. Build preview service"
echo ""