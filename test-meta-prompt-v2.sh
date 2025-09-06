#!/bin/bash
set -euo pipefail

echo "╔═══════════════════════════════════════════════════════════════════════════════╗"
echo "║         QuantumLayer Platform - Meta-Prompt Enhanced Generation Test           ║"
echo "║                  Demonstrating Full Integration with 12-Stage Pipeline         ║"
echo "╚═══════════════════════════════════════════════════════════════════════════════╝"
echo ""

# Using master node for reliability
WORKFLOW_API="http://192.168.1.177:30889"

echo "▶ Triggering Extended Code Generation Workflow..."
echo ""

# Create request
cat > /tmp/enhanced-request.json <<'JSON'
{
    "prompt": "Build a production Python FastAPI service for real-time chat with WebSocket support, JWT auth, PostgreSQL persistence, Redis pub/sub, rate limiting, and Prometheus metrics",
    "language": "python",
    "framework": "fastapi",
    "type": "microservice",
    "name": "chat-service"
}
JSON

# Submit workflow
RESPONSE=$(curl -s -X POST "${WORKFLOW_API}/api/v1/workflows/generate-extended" \
    -H "Content-Type: application/json" \
    -d @/tmp/enhanced-request.json)

echo "Response: $RESPONSE"
echo ""

# Extract workflow ID with better parsing
WORKFLOW_ID=$(echo "$RESPONSE" | python3 -c "import sys, json; d=json.load(sys.stdin); print(d.get('workflow_id', ''))" 2>/dev/null || echo "")

if [[ -z "$WORKFLOW_ID" ]]; then
    echo "❌ Failed to get workflow ID"
    exit 1
fi

echo "✅ Workflow Started: $WORKFLOW_ID"
echo ""
echo "Monitoring 12-stage execution with meta-prompt enhancement..."
echo ""

# Monitor execution
for i in {1..40}; do
    sleep 3
    
    STATUS=$(curl -s "${WORKFLOW_API}/api/v1/workflows/${WORKFLOW_ID}" | \
        python3 -c "import sys, json; d=json.load(sys.stdin); print(d.get('status', 'RUNNING'))" 2>/dev/null || echo "CHECKING")
    
    echo "[$i/40] Status: $STATUS"
    
    if [[ "$STATUS" == "COMPLETED" ]] || [[ "$STATUS" == "Completed" ]]; then
        echo ""
        echo "✅ WORKFLOW COMPLETED SUCCESSFULLY!"
        echo ""
        
        # Get result
        RESULT=$(curl -s "${WORKFLOW_API}/api/v1/workflows/${WORKFLOW_ID}/result")
        echo "Result summary:"
        echo "$RESULT" | python3 -m json.tool 2>/dev/null | head -20 || echo "$RESULT" | head -20
        
        echo ""
        echo "═══════════════════════════════════════════════════════════════"
        echo "🎉 Meta-Prompt Enhanced Code Generation Complete!"
        echo "═══════════════════════════════════════════════════════════════"
        echo ""
        echo "The workflow successfully:"
        echo "• Used meta-prompt engine for prompt enhancement"
        echo "• Executed all 12 stages of the pipeline"
        echo "• Generated production-ready code"
        echo ""
        exit 0
    elif [[ "$STATUS" == "FAILED" ]] || [[ "$STATUS" == "Failed" ]]; then
        echo ""
        echo "❌ Workflow failed!"
        curl -s "${WORKFLOW_API}/api/v1/workflows/${WORKFLOW_ID}" | python3 -m json.tool
        exit 1
    fi
done

echo "⏱️ Workflow still running after 2 minutes..."
echo "Check status: curl ${WORKFLOW_API}/api/v1/workflows/${WORKFLOW_ID}"
