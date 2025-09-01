# NodePort Allocation Strategy

## QuantumLayer Platform NodePort Assignments

NodePorts are cluster-wide and must be unique across ALL namespaces. This document tracks our allocations.

## Allocated Ports (quantumlayer namespace)

| Service | Internal Port | NodePort | Description | Status |
|---------|--------------|----------|-------------|--------|
| API Gateway | 8000 | 30800 | GraphQL/REST API | Reserved |
| Web Frontend | 3000 | 30300 | Next.js UI | Reserved |
| PostgreSQL | 5432 | 30432 | Primary Database | ✅ Active |
| Redis | 6379 | 30379 | Cache/Queue | ✅ Active |
| Qdrant | 6333 | 30333 | Vector Database | Reserved |
| NATS | 4222 | 30222 | Messaging | Reserved |
| Temporal | 7233 | 30233 | Workflow Engine | Reserved |
| Temporal UI | 8080 | 30808 | Temporal Dashboard | Reserved |
| MinIO | 9000 | 30900 | Object Storage | Reserved |
| MinIO Console | 9001 | 30901 | MinIO UI | Reserved |
| Prometheus | 9090 | 30909 | Metrics | Reserved |
| Grafana | 3000 | 30301 | Dashboards | Reserved |
| LLM Router | 8080 | 30881 | LLM Service | Reserved |
| Agent Orchestrator | 8090 | 30890 | Agent Service | Reserved |

## Other Namespaces (for reference)

| Namespace | Service | NodePort | Notes |
|-----------|---------|----------|-------|
| qlayer-dev | API | 30080 | Existing QLLayer |
| qlayer-dev | Frontend | 30081 | Existing QLLayer |
| qlayer-dev | Sandbox | 30082 | Existing QLLayer |
| qlayer-dev | Worker Metrics | 30090 | Existing QLLayer |
| qlayer-dev | QTest Agent | 30091-30092 | Existing QLLayer |
| qlayer-dev | Temporal Web | 30088 | Existing QLLayer |
| qlayer-registry | Registry | 30500 | Docker Registry |

## Port Allocation Rules

1. **Range**: Use 30000-32767 (Kubernetes default range)
2. **Spacing**: Leave gaps between services for future expansion
3. **Grouping**: Group related services (e.g., 308xx for core services)
4. **Documentation**: Always update this file when allocating new ports
5. **Verification**: Check with `kubectl get svc --all-namespaces | grep NodePort`

## Checking for Conflicts

Before allocating a new NodePort:

```bash
# Check if port is already in use
kubectl get svc --all-namespaces -o json | jq '.items[].spec.ports[]?.nodePort' | sort -u | grep 30800

# Or use this one-liner to see all allocated NodePorts
kubectl get svc --all-namespaces -o go-template='{{range .items}}{{range .spec.ports}}{{if .nodePort}}{{.nodePort}}{{"\n"}}{{end}}{{end}}{{end}}' | sort -u
```

## Access URLs

All services are accessible from outside the cluster using:
- `http://192.168.7.235:<nodeport>` (master node)
- `http://192.168.7.236:<nodeport>` (worker-01)
- `http://192.168.7.237:<nodeport>` (worker-02)
- `http://192.168.7.238:<nodeport>` (worker-03)

Any node IP works due to kube-proxy routing.

## Security Note

NodePort services expose ports on ALL nodes. For production:
1. Use LoadBalancer or Ingress instead
2. Implement network policies
3. Add authentication/authorization
4. Use TLS/HTTPS
5. Consider using ClusterIP with port-forwarding for sensitive services