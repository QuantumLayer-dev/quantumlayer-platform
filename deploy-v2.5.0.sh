#!/bin/bash
set -e

VERSION="2.5.0"

echo "Deploying QuantumLayer Platform v${VERSION}..."

# Update image tags in all deployments
for ns in quantumlayer temporal security-services; do
    echo "Updating namespace: $ns"
    kubectl get deployments -n $ns -o name | while read deploy; do
        kubectl set image $deploy \*=\*:${VERSION} -n $ns --record || true
    done
done

echo "Deployment of v${VERSION} initiated. Check pod status with:"
echo "kubectl get pods --all-namespaces | grep -v Running"
