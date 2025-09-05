# QInfra Deployment Guide

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Quick Start](#quick-start)
3. [Production Deployment](#production-deployment)
4. [Configuration](#configuration)
5. [Monitoring Setup](#monitoring-setup)
6. [Troubleshooting](#troubleshooting)

## Prerequisites

### System Requirements

#### Minimum (Development)
- **Kubernetes:** v1.24+
- **Nodes:** 3 worker nodes
- **CPU:** 4 vCPUs per node
- **Memory:** 8GB RAM per node  
- **Storage:** 100GB for registry, 50GB for database

#### Recommended (Production)
- **Kubernetes:** v1.26+
- **Nodes:** 5+ worker nodes
- **CPU:** 8 vCPUs per node
- **Memory:** 16GB RAM per node
- **Storage:** 500GB SSD for registry, 200GB SSD for database

### Required Tools

```bash
# Check tool versions
kubectl version --client
helm version
docker version
packer version

# Install required tools (macOS)
brew install kubectl helm docker packer

# Install required tools (Linux)
curl -LO https://dl.k8s.io/release/v1.28.0/bin/linux/amd64/kubectl
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

### Cluster Access

```bash
# Verify cluster access
kubectl cluster-info
kubectl get nodes

# Expected output:
NAME            STATUS   ROLES           AGE   VERSION
k8s-master      Ready    control-plane   10d   v1.28.2
k8s-worker-01   Ready    <none>          10d   v1.28.2
k8s-worker-02   Ready    <none>          10d   v1.28.2
k8s-worker-03   Ready    <none>          10d   v1.28.2
```

## Quick Start

### 1. Clone Repository

```bash
git clone https://github.com/quantumlayer/platform.git
cd quantumlayer-platform
```

### 2. Create Namespaces

```bash
kubectl create namespace image-registry
kubectl create namespace quantumlayer
kubectl create namespace temporal
```

### 3. Deploy Docker Registry

```bash
# Deploy Docker Registry with authentication
kubectl apply -f infrastructure/kubernetes/docker-registry.yaml

# Wait for registry to be ready
kubectl wait --for=condition=ready pod -l app=docker-registry -n image-registry --timeout=300s

# Verify registry is accessible
curl -u admin:quantum2025 http://<node-ip>:30500/v2/_catalog
```

### 4. Deploy Image Registry Service

```bash
# Build and push image
cd services/image-registry
docker build -t ghcr.io/quantumlayer-dev/image-registry:latest .
docker push ghcr.io/quantumlayer-dev/image-registry:latest

# Deploy service
kubectl apply -f ../../infrastructure/kubernetes/image-registry-service.yaml

# Wait for service to be ready
kubectl wait --for=condition=ready pod -l app=image-registry -n quantumlayer --timeout=300s

# Test service health
curl http://<node-ip>:30096/health
```

### 5. Quick Test

```bash
# Run test script
chmod +x test-golden-images.sh
./test-golden-images.sh
```

## Production Deployment

### 1. Infrastructure Setup

#### Storage Configuration

```yaml
# storage-class.yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: qinfra-ssd
provisioner: kubernetes.io/aws-ebs  # Or your cloud provider
parameters:
  type: gp3
  iopsPerGB: "10"
  encrypted: "true"
reclaimPolicy: Retain
volumeBindingMode: WaitForFirstConsumer
```

#### Database Setup

```bash
# Deploy PostgreSQL with HA
helm repo add bitnami https://charts.bitnami.com/bitnami
helm install postgres bitnami/postgresql-ha \
  --namespace quantumlayer \
  --set global.postgresql.auth.postgresPassword=secretpassword \
  --set global.postgresql.auth.database=qinfra \
  --set postgresql.replicaCount=3 \
  --set persistence.storageClass=qinfra-ssd \
  --set persistence.size=100Gi
```

#### Redis Setup

```bash
# Deploy Redis with Sentinel
helm install redis bitnami/redis \
  --namespace quantumlayer \
  --set auth.password=redispassword \
  --set sentinel.enabled=true \
  --set replica.replicaCount=3 \
  --set persistence.storageClass=qinfra-ssd
```

### 2. Temporal Deployment

```bash
# Add Temporal Helm repo
helm repo add temporal https://temporalio.github.io/helm-charts

# Deploy Temporal
helm install temporal temporal/temporal \
  --namespace temporal \
  --set server.replicaCount=3 \
  --set cassandra.enabled=false \
  --set postgresql.enabled=false \
  --set mysql.enabled=false \
  --set elasticsearch.enabled=false \
  --set prometheus.enabled=false \
  --set grafana.enabled=false \
  --set server.config.persistence.default.sql.host=postgres-postgresql.quantumlayer.svc.cluster.local \
  --set server.config.persistence.default.sql.port=5432 \
  --set server.config.persistence.default.sql.user=temporal \
  --set server.config.persistence.default.sql.password=temporalpassword
```

### 3. Service Deployment

#### Configure Secrets

```bash
# Create secrets
kubectl create secret generic qinfra-secrets \
  --namespace quantumlayer \
  --from-literal=database-url='postgres://qinfra:password@postgres-postgresql.quantumlayer.svc.cluster.local/qinfra' \
  --from-literal=redis-url='redis://:redispassword@redis-master.quantumlayer.svc.cluster.local:6379' \
  --from-literal=temporal-host='temporal-frontend.temporal.svc.cluster.local:7233' \
  --from-literal=registry-password='quantum2025'

# Create image pull secret
kubectl create secret docker-registry ghcr-secret \
  --namespace quantumlayer \
  --docker-server=ghcr.io \
  --docker-username=<github-username> \
  --docker-password=<github-token>
```

#### Deploy Services

```bash
# Deploy all QInfra services
kubectl apply -f infrastructure/kubernetes/production/

# Verify deployments
kubectl get deployments -n quantumlayer
kubectl get pods -n quantumlayer
kubectl get services -n quantumlayer
```

### 4. Ingress Configuration

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: qinfra-ingress
  namespace: quantumlayer
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/rate-limit: "100"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - api.qinfra.example.com
    secretName: qinfra-tls
  rules:
  - host: api.qinfra.example.com
    http:
      paths:
      - path: /api/v1/images
        pathType: Prefix
        backend:
          service:
            name: image-registry
            port:
              number: 8096
      - path: /api/v1/drift
        pathType: Prefix
        backend:
          service:
            name: drift-engine
            port:
              number: 8097
```

### 5. Auto-scaling Configuration

```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: image-registry-hpa
  namespace: quantumlayer
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: image-registry
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
```

## Configuration

### Environment Variables

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: qinfra-config
  namespace: quantumlayer
data:
  # Service Configuration
  PORT: "8096"
  LOG_LEVEL: "info"
  
  # Registry Configuration
  REGISTRY_URL: "http://docker-registry.image-registry.svc.cluster.local:5000"
  REGISTRY_USERNAME: "admin"
  
  # Database Configuration
  DATABASE_POOL_SIZE: "20"
  DATABASE_MAX_IDLE: "10"
  DATABASE_MAX_LIFETIME: "3600"
  
  # Redis Configuration
  REDIS_POOL_SIZE: "10"
  REDIS_MAX_RETRIES: "3"
  
  # Temporal Configuration
  TEMPORAL_NAMESPACE: "quantumlayer"
  TEMPORAL_TASK_QUEUE: "infrastructure-generation"
  
  # Security Configuration
  JWT_ISSUER: "https://auth.qinfra.example.com"
  JWT_AUDIENCE: "qinfra-api"
  
  # Feature Flags
  ENABLE_GOLDEN_IMAGES: "true"
  ENABLE_PATCH_MANAGEMENT: "true"
  ENABLE_DRIFT_DETECTION: "true"
  ENABLE_COMPLIANCE: "true"
```

### Packer Configuration

```hcl
# packer/config.pkr.hcl
variable "registry_url" {
  type    = string
  default = env("REGISTRY_URL")
}

variable "registry_username" {
  type    = string
  default = env("REGISTRY_USERNAME")
}

variable "registry_password" {
  type      = string
  sensitive = true
  default   = env("REGISTRY_PASSWORD")
}

source "docker" "base" {
  image  = var.base_image
  commit = true
}

build {
  sources = ["source.docker.base"]
  
  provisioner "shell" {
    scripts = [
      "scripts/base-setup.sh",
      "scripts/cis-hardening.sh",
      "scripts/compliance-validation.sh"
    ]
  }
  
  post-processor "docker-tag" {
    repository = "${var.registry_url}/${var.image_name}"
    tags       = [var.image_version, "latest"]
  }
  
  post-processor "docker-push" {
    login          = true
    login_username = var.registry_username
    login_password = var.registry_password
  }
}
```

## Monitoring Setup

### 1. Prometheus Installation

```bash
# Add Prometheus Helm repo
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts

# Install Prometheus with custom values
cat > prometheus-values.yaml <<EOF
prometheus:
  prometheusSpec:
    serviceMonitorSelectorNilUsesHelmValues: false
    retention: 30d
    storageSpec:
      volumeClaimTemplate:
        spec:
          storageClassName: qinfra-ssd
          accessModes: ["ReadWriteOnce"]
          resources:
            requests:
              storage: 50Gi
EOF

helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --values prometheus-values.yaml
```

### 2. Grafana Dashboards

```bash
# Import QInfra dashboards
kubectl apply -f monitoring/dashboards/

# Access Grafana
kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80

# Default credentials: admin/prom-operator
```

### 3. ServiceMonitor Configuration

```yaml
# servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: qinfra-metrics
  namespace: quantumlayer
spec:
  selector:
    matchLabels:
      app: qinfra
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
    scheme: http
```

### 4. Alerts Configuration

```yaml
# alerts.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: qinfra-alerts
  namespace: quantumlayer
spec:
  groups:
  - name: qinfra.rules
    interval: 30s
    rules:
    - alert: HighDriftPercentage
      expr: qinfra_drift_percentage > 10
      for: 5m
      labels:
        severity: warning
        team: platform
      annotations:
        summary: "High infrastructure drift detected"
        description: "{{ $value }}% of infrastructure has drifted"
    
    - alert: CriticalVulnerabilities
      expr: qinfra_vulnerabilities_critical > 0
      for: 1m
      labels:
        severity: critical
        team: security
      annotations:
        summary: "Critical vulnerabilities detected"
        description: "{{ $value }} critical vulnerabilities found"
    
    - alert: ComplianceViolation
      expr: qinfra_compliance_score < 80
      for: 10m
      labels:
        severity: warning
        team: compliance
      annotations:
        summary: "Compliance score below threshold"
        description: "Compliance score is {{ $value }}%"
```

## Health Checks

### Service Health Endpoints

```bash
# Check all service health
for service in image-registry patch-manager drift-engine compliance-validator; do
  echo "Checking $service..."
  curl -s http://<node-ip>:30096/health | jq .
done
```

### Database Health

```bash
# Check PostgreSQL
kubectl exec -n quantumlayer postgres-postgresql-0 -- pg_isready

# Check Redis
kubectl exec -n quantumlayer redis-master-0 -- redis-cli ping
```

### Temporal Health

```bash
# Check Temporal
kubectl exec -n temporal deployment/temporal-frontend -- tctl cluster health
```

## Backup and Recovery

### Database Backup

```bash
# Create backup
kubectl exec -n quantumlayer postgres-postgresql-0 -- \
  pg_dump -U postgres qinfra | gzip > qinfra-backup-$(date +%Y%m%d).sql.gz

# Restore backup
gunzip < qinfra-backup-20240101.sql.gz | \
  kubectl exec -i -n quantumlayer postgres-postgresql-0 -- \
  psql -U postgres qinfra
```

### Registry Backup

```bash
# Backup registry data
kubectl exec -n image-registry docker-registry-0 -- \
  tar czf /tmp/registry-backup.tar.gz /var/lib/registry

kubectl cp image-registry/docker-registry-0:/tmp/registry-backup.tar.gz \
  ./registry-backup-$(date +%Y%m%d).tar.gz
```

## Troubleshooting

### Common Issues

#### 1. Pod Not Starting

```bash
# Check pod status
kubectl describe pod <pod-name> -n quantumlayer

# Check logs
kubectl logs <pod-name> -n quantumlayer --previous

# Common fixes:
# - Insufficient resources: Scale down replicas or add nodes
# - Image pull errors: Check image pull secrets
# - Database connection: Verify database credentials and connectivity
```

#### 2. Service Not Accessible

```bash
# Check service endpoints
kubectl get endpoints -n quantumlayer

# Test service connectivity
kubectl run debug --image=nicolaka/netshoot -it --rm -- /bin/bash
curl http://image-registry.quantumlayer.svc.cluster.local:8096/health

# Check network policies
kubectl get networkpolicies -n quantumlayer
```

#### 3. High Memory Usage

```bash
# Check resource usage
kubectl top pods -n quantumlayer

# Adjust resource limits
kubectl edit deployment image-registry -n quantumlayer

# Enable vertical pod autoscaling
kubectl autoscale deployment image-registry \
  --min=2 --max=10 --cpu-percent=70 -n quantumlayer
```

#### 4. Drift Detection Not Working

```bash
# Check drift engine logs
kubectl logs -l app=drift-engine -n quantumlayer

# Verify agent connectivity
kubectl exec -n quantumlayer drift-engine-0 -- \
  curl http://image-registry:8096/health

# Reset drift baseline
curl -X POST http://<node-ip>:30097/drift/reset
```

### Debug Mode

```bash
# Enable debug logging
kubectl set env deployment/image-registry LOG_LEVEL=debug -n quantumlayer

# Watch logs
kubectl logs -f deployment/image-registry -n quantumlayer

# Disable debug logging
kubectl set env deployment/image-registry LOG_LEVEL=info -n quantumlayer
```

## Upgrade Procedure

### 1. Backup Current State

```bash
# Backup database
./scripts/backup-database.sh

# Backup configurations
kubectl get configmaps -n quantumlayer -o yaml > configmaps-backup.yaml
kubectl get secrets -n quantumlayer -o yaml > secrets-backup.yaml
```

### 2. Rolling Update

```bash
# Update image version
kubectl set image deployment/image-registry \
  image-registry=ghcr.io/quantumlayer-dev/image-registry:v1.1.0 \
  -n quantumlayer

# Monitor rollout
kubectl rollout status deployment/image-registry -n quantumlayer

# Verify new version
kubectl exec deployment/image-registry -n quantumlayer -- /app/image-registry --version
```

### 3. Rollback if Needed

```bash
# Rollback to previous version
kubectl rollout undo deployment/image-registry -n quantumlayer

# Check rollback status
kubectl rollout status deployment/image-registry -n quantumlayer
```

## Security Hardening

### Network Policies

```yaml
# network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: qinfra-network-policy
  namespace: quantumlayer
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
  egress:
  - to:
    - namespaceSelector: {}
  - to:
    - podSelector:
        matchLabels:
          app: postgres
  - to:
    - podSelector:
        matchLabels:
          app: redis
```

### Pod Security Policies

```yaml
# pod-security-policy.yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: qinfra-psp
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
  - ALL
  volumes:
  - configMap
  - secret
  - persistentVolumeClaim
  runAsUser:
    rule: MustRunAsNonRoot
  seLinux:
    rule: RunAsAny
  fsGroup:
    rule: RunAsAny
  readOnlyRootFilesystem: true
```

## Performance Tuning

### JVM Options (if using Java services)

```yaml
env:
- name: JAVA_OPTS
  value: "-Xms512m -Xmx1024m -XX:+UseG1GC -XX:MaxGCPauseMillis=200"
```

### Database Connection Pooling

```yaml
env:
- name: DATABASE_POOL_SIZE
  value: "20"
- name: DATABASE_MAX_IDLE
  value: "10"
- name: DATABASE_IDLE_TIMEOUT
  value: "300"
```

### Resource Optimization

```bash
# Enable resource recommendations
kubectl apply -f - <<EOF
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: image-registry-vpa
  namespace: quantumlayer
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: image-registry
  updatePolicy:
    updateMode: "Off"
EOF

# Check recommendations
kubectl describe vpa image-registry-vpa -n quantumlayer
```

## Maintenance

### Regular Tasks

```bash
# Daily
- Check service health
- Review error logs
- Monitor disk usage

# Weekly
- Run vulnerability scans
- Review compliance scores
- Update golden images

# Monthly
- Database maintenance
- Certificate renewal
- Security patches

# Quarterly
- Disaster recovery drill
- Performance review
- Capacity planning
```

---

**Deployment Guide Version:** 1.0.0  
**Last Updated:** 2024-09-05  
**Support:** support@quantumlayer.io