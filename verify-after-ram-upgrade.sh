#!/bin/bash

echo "=== Cluster Health Check After RAM Upgrade ==="
echo ""

# Check all nodes are ready
echo "1. Node Status:"
kubectl get nodes
echo ""

# Check memory capacity
echo "2. Memory Capacity (Should show ~8GB per worker):"
kubectl describe nodes | grep -A2 "Capacity:" | grep memory
echo ""

# Check all critical pods are running
echo "3. Critical Services Status:"
kubectl get pods -n quantumlayer --no-headers | wc -l
echo "Quantumlayer pods running: $(kubectl get pods -n quantumlayer --field-selector=status.phase=Running --no-headers | wc -l)"
kubectl get pods -n temporal --no-headers | wc -l  
echo "Temporal pods running: $(kubectl get pods -n temporal --field-selector=status.phase=Running --no-headers | wc -l)"
echo ""

# Check memory usage
echo "4. Current Memory Usage:"
kubectl top nodes
echo ""

# Test critical services
echo "5. Service Health Checks:"
curl -s http://192.168.1.177:30889/health > /dev/null && echo "✅ Workflow API: Healthy" || echo "❌ Workflow API: Down"
curl -s http://192.168.1.177:30095/health > /dev/null && echo "✅ QInfra: Healthy" || echo "❌ QInfra: Down"
curl -s http://192.168.1.177:30098/health > /dev/null && echo "✅ QInfra-AI: Healthy" || echo "❌ QInfra-AI: Down"
echo ""

echo "=== RAM Upgrade Verification Complete ==="