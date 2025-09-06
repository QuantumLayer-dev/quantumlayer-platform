#!/bin/bash

# Fix LLM Secrets - Extract working credentials from running pod and update secret

echo "Extracting working LLM credentials from running pod..."

# Get credentials from running pod
GROQ_API_KEY=$(kubectl exec -n quantumlayer deployment/llm-router -- sh -c 'echo $GROQ_API_KEY' 2>/dev/null)
AZURE_OPENAI_KEY=$(kubectl exec -n quantumlayer deployment/llm-router -- sh -c 'echo $AZURE_OPENAI_KEY' 2>/dev/null)
AZURE_OPENAI_ENDPOINT=$(kubectl exec -n quantumlayer deployment/llm-router -- sh -c 'echo $AZURE_OPENAI_ENDPOINT' 2>/dev/null)
AZURE_OPENAI_DEPLOYMENT=$(kubectl exec -n quantumlayer deployment/llm-router -- sh -c 'echo $AZURE_OPENAI_DEPLOYMENT' 2>/dev/null)
AWS_ACCESS_KEY_ID=$(kubectl exec -n quantumlayer deployment/llm-router -- sh -c 'echo $AWS_ACCESS_KEY_ID' 2>/dev/null)
AWS_SECRET_ACCESS_KEY=$(kubectl exec -n quantumlayer deployment/llm-router -- sh -c 'echo $AWS_SECRET_ACCESS_KEY' 2>/dev/null)
AWS_BEDROCK_REGION=$(kubectl exec -n quantumlayer deployment/llm-router -- sh -c 'echo $AWS_BEDROCK_REGION' 2>/dev/null)

echo "Found credentials:"
echo "  GROQ_API_KEY: ${GROQ_API_KEY:0:10}..."
echo "  AZURE_OPENAI_KEY: ${AZURE_OPENAI_KEY:0:10}..."
echo "  AZURE_OPENAI_ENDPOINT: $AZURE_OPENAI_ENDPOINT"
echo "  AWS_ACCESS_KEY_ID: ${AWS_ACCESS_KEY_ID:0:10}..."

# Delete the old secret
echo "Deleting old secret..."
kubectl delete secret llm-credentials -n quantumlayer 2>/dev/null || true

# Create new secret with actual values
echo "Creating new secret with actual credentials..."
kubectl create secret generic llm-credentials \
  --namespace=quantumlayer \
  --from-literal=GROQ_API_KEY="$GROQ_API_KEY" \
  --from-literal=AZURE_OPENAI_KEY="$AZURE_OPENAI_KEY" \
  --from-literal=AZURE_OPENAI_ENDPOINT="$AZURE_OPENAI_ENDPOINT" \
  --from-literal=AZURE_OPENAI_DEPLOYMENT="$AZURE_OPENAI_DEPLOYMENT" \
  --from-literal=AZURE_OPENAI_API_VERSION="2024-02-01" \
  --from-literal=AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_ID" \
  --from-literal=AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY" \
  --from-literal=AWS_BEDROCK_REGION="$AWS_BEDROCK_REGION" \
  --from-literal=AWS_BEDROCK_MODEL="anthropic.claude-3-haiku-20240307-v1:0" \
  --from-literal=OPENAI_API_KEY="${OPENAI_API_KEY:-sk-placeholder}" \
  --from-literal=ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY:-sk-ant-placeholder}" \
  --from-literal=GROQ_MODEL="mixtral-8x7b-32768" \
  --from-literal=OPENAI_MODEL="gpt-4-turbo-preview" \
  --from-literal=ANTHROPIC_MODEL="claude-3-opus-20240229"

echo "Secret updated successfully!"

# Restart deployment to pick up new secrets
echo "Restarting LLM router deployment..."
kubectl rollout restart deployment/llm-router -n quantumlayer

echo "Waiting for rollout to complete..."
kubectl rollout status deployment/llm-router -n quantumlayer --timeout=60s

echo "Done! LLM router should now have correct credentials."