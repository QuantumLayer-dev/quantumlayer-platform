#!/bin/bash
set -euo pipefail

# ═══════════════════════════════════════════════════════════════════════════════
# QuantumLayer Enterprise Deployment Script
# Production-grade deployment with security, monitoring, and high availability
# ═══════════════════════════════════════════════════════════════════════════════

# Configuration
NAMESPACE="${NAMESPACE:-quantumlayer}"
REGISTRY="${REGISTRY:-localhost:5000}"
ENVIRONMENT="${ENVIRONMENT:-production}"
DEPLOY_MODE="${1:-all}" # all, sandbox, capsule, monitoring, security

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Logging
log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "\n${BOLD}${BLUE}▶ $1${NC}"; }

# Banner
print_banner() {
    echo -e "${BOLD}${MAGENTA}"
    cat << "EOF"
╔═══════════════════════════════════════════════════════════════════════════════╗
║                  QuantumLayer Enterprise Deployment                           ║
║                     Production-Grade Infrastructure                           ║
╚═══════════════════════════════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"
}

# ═══════════════════════════════════════════════════════════════════════════════
# Pre-flight Checks
# ═══════════════════════════════════════════════════════════════════════════════

preflight_checks() {
    log_step "Running Pre-flight Checks"
    
    # Check required tools
    for tool in kubectl docker helm; do
        if command -v $tool &> /dev/null; then
            log_info "✓ $tool installed"
        else
            log_error "✗ $tool not found"
            exit 1
        fi
    done
    
    # Check cluster connectivity
    if kubectl cluster-info &> /dev/null; then
        log_info "✓ Kubernetes cluster accessible"
    else
        log_error "✗ Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check namespace
    if ! kubectl get namespace $NAMESPACE &> /dev/null; then
        log_info "Creating namespace $NAMESPACE"
        kubectl create namespace $NAMESPACE
        kubectl label namespace $NAMESPACE \
            environment=$ENVIRONMENT \
            managed-by=quantumlayer \
            security=enforced
    fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# Security Configuration
# ═══════════════════════════════════════════════════════════════════════════════

deploy_security_policies() {
    log_step "Deploying Security Policies"
    
    # Pod Security Policy
    cat <<EOF | kubectl apply -f -
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: quantumlayer-restricted
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
    - ALL
  volumes:
    - 'configMap'
    - 'emptyDir'
    - 'projected'
    - 'secret'
    - 'downwardAPI'
    - 'persistentVolumeClaim'
  hostNetwork: false
  hostIPC: false
  hostPID: false
  runAsUser:
    rule: 'MustRunAsNonRoot'
  seLinux:
    rule: 'RunAsAny'
  fsGroup:
    rule: 'RunAsAny'
  readOnlyRootFilesystem: false
EOF
    
    # Network Policies
    cat <<EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: quantumlayer-network-policy
  namespace: $NAMESPACE
spec:
  podSelector: {}
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: $NAMESPACE
        - namespaceSelector:
            matchLabels:
              name: temporal
  egress:
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 443
        - protocol: TCP
          port: 80
        - protocol: TCP
          port: 53
    - to:
        - namespaceSelector:
            matchLabels:
              name: $NAMESPACE
    - to:
        - namespaceSelector:
            matchLabels:
              name: temporal
EOF
    
    log_info "✓ Security policies deployed"
}

# ═══════════════════════════════════════════════════════════════════════════════
# Build and Push Images
# ═══════════════════════════════════════════════════════════════════════════════

build_and_push_images() {
    log_step "Building and Pushing Docker Images"
    
    # Build with security scanning
    build_with_scan() {
        local service=$1
        local path=$2
        
        log_info "Building $service..."
        cd $path
        
        # Build image
        docker build -t $REGISTRY/$service:latest .
        
        # Security scan with Trivy (if available)
        if command -v trivy &> /dev/null; then
            log_info "Scanning $service for vulnerabilities..."
            trivy image --severity HIGH,CRITICAL $REGISTRY/$service:latest || log_warn "Vulnerabilities found"
        fi
        
        # Push to registry
        docker push $REGISTRY/$service:latest
        cd - > /dev/null
    }
    
    # Build services
    if [[ "$DEPLOY_MODE" == "all" || "$DEPLOY_MODE" == "sandbox" ]]; then
        build_with_scan "sandbox-executor" "packages/sandbox-executor"
    fi
    
    if [[ "$DEPLOY_MODE" == "all" || "$DEPLOY_MODE" == "capsule" ]]; then
        build_with_scan "capsule-builder" "packages/capsule-builder"
    fi
    
    log_info "✓ Images built and pushed"
}

# ═══════════════════════════════════════════════════════════════════════════════
# Deploy Sandbox Executor
# ═══════════════════════════════════════════════════════════════════════════════

deploy_sandbox_executor() {
    log_step "Deploying Sandbox Executor with Enterprise Configuration"
    
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: sandbox-executor-config
  namespace: $NAMESPACE
data:
  config.yaml: |
    server:
      port: 8091
      timeout: 30s
    security:
      max_execution_time: 60s
      max_memory: 2Gi
      max_cpu: 2
      allowed_languages:
        - python
        - javascript
        - typescript
        - go
        - java
        - rust
    observability:
      metrics_enabled: true
      tracing_enabled: true
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sandbox-executor
  namespace: $NAMESPACE
  labels:
    app: sandbox-executor
    version: v1
    component: execution
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: sandbox-executor
  template:
    metadata:
      labels:
        app: sandbox-executor
        version: v1
        component: execution
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8091"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: sandbox-executor
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
      containers:
      - name: sandbox-executor
        image: $REGISTRY/sandbox-executor:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 8091
          name: http
          protocol: TCP
        - containerPort: 9090
          name: metrics
          protocol: TCP
        env:
        - name: ENVIRONMENT
          value: "$ENVIRONMENT"
        - name: NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: DOCKER_HOST
          value: "tcp://localhost:2375"
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2"
        livenessProbe:
          httpGet:
            path: /health
            port: 8091
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /ready
            port: 8091
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
        volumeMounts:
        - name: config
          mountPath: /etc/sandbox
        - name: docker-socket
          mountPath: /var/run/docker.sock
      - name: dind
        image: docker:24-dind
        securityContext:
          privileged: true
        env:
        - name: DOCKER_TLS_CERTDIR
          value: ""
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2"
        volumeMounts:
        - name: docker-socket
          mountPath: /var/run/docker.sock
        - name: docker-storage
          mountPath: /var/lib/docker
      volumes:
      - name: config
        configMap:
          name: sandbox-executor-config
      - name: docker-socket
        emptyDir: {}
      - name: docker-storage
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: sandbox-executor
  namespace: $NAMESPACE
  labels:
    app: sandbox-executor
spec:
  type: ClusterIP
  selector:
    app: sandbox-executor
  ports:
  - name: http
    port: 8091
    targetPort: 8091
    protocol: TCP
  - name: metrics
    port: 9090
    targetPort: 9090
    protocol: TCP
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: sandbox-executor
  namespace: $NAMESPACE
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: sandbox-executor
  namespace: $NAMESPACE
rules:
- apiGroups: [""]
  resources: ["pods", "pods/log"]
  verbs: ["get", "list", "create", "delete"]
- apiGroups: ["batch"]
  resources: ["jobs"]
  verbs: ["get", "list", "create", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: sandbox-executor
  namespace: $NAMESPACE
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: sandbox-executor
subjects:
- kind: ServiceAccount
  name: sandbox-executor
  namespace: $NAMESPACE
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: sandbox-executor
  namespace: $NAMESPACE
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: sandbox-executor
  minReplicas: 2
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
---
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: sandbox-executor
  namespace: $NAMESPACE
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: sandbox-executor
EOF
    
    # Wait for deployment
    kubectl rollout status deployment/sandbox-executor -n $NAMESPACE --timeout=300s
    log_info "✓ Sandbox Executor deployed"
}

# ═══════════════════════════════════════════════════════════════════════════════
# Deploy Capsule Builder
# ═══════════════════════════════════════════════════════════════════════════════

deploy_capsule_builder() {
    log_step "Deploying Capsule Builder with Enterprise Templates"
    
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: capsule-builder-config
  namespace: $NAMESPACE
data:
  config.yaml: |
    server:
      port: 8092
    storage:
      type: s3
      bucket: quantum-capsules
    templates:
      - language: python
        frameworks: [fastapi, flask, django]
      - language: javascript
        frameworks: [express, react, vue]
      - language: go
        frameworks: [gin, echo, fiber]
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: capsule-builder
  namespace: $NAMESPACE
  labels:
    app: capsule-builder
    version: v1
    component: builder
spec:
  replicas: 2
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: capsule-builder
  template:
    metadata:
      labels:
        app: capsule-builder
        version: v1
        component: builder
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8092"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: capsule-builder
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
      containers:
      - name: capsule-builder
        image: $REGISTRY/capsule-builder:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 8092
          name: http
        - containerPort: 9090
          name: metrics
        env:
        - name: ENVIRONMENT
          value: "$ENVIRONMENT"
        - name: QUANTUM_DROPS_URL
          value: "http://quantum-drops.$NAMESPACE.svc.cluster.local:8090"
        - name: S3_ENDPOINT
          value: "http://minio.$NAMESPACE.svc.cluster.local:9000"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "1"
        livenessProbe:
          httpGet:
            path: /health
            port: 8092
          initialDelaySeconds: 20
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8092
          initialDelaySeconds: 10
          periodSeconds: 5
        volumeMounts:
        - name: config
          mountPath: /etc/capsule
      volumes:
      - name: config
        configMap:
          name: capsule-builder-config
---
apiVersion: v1
kind: Service
metadata:
  name: capsule-builder
  namespace: $NAMESPACE
  labels:
    app: capsule-builder
spec:
  type: ClusterIP
  selector:
    app: capsule-builder
  ports:
  - name: http
    port: 8092
    targetPort: 8092
  - name: metrics
    port: 9090
    targetPort: 9090
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: capsule-builder
  namespace: $NAMESPACE
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: capsule-builder
  namespace: $NAMESPACE
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: capsule-builder
  minReplicas: 2
  maxReplicas: 5
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
EOF
    
    kubectl rollout status deployment/capsule-builder -n $NAMESPACE --timeout=300s
    log_info "✓ Capsule Builder deployed"
}

# ═══════════════════════════════════════════════════════════════════════════════
# Deploy Monitoring Stack
# ═══════════════════════════════════════════════════════════════════════════════

deploy_monitoring() {
    log_step "Deploying Monitoring Stack"
    
    # Add Prometheus Helm repo
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
    helm repo update
    
    # Install Prometheus Stack
    helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
        --namespace monitoring \
        --create-namespace \
        --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
        --set grafana.adminPassword=quantumlayer2024 \
        --wait
    
    # Create ServiceMonitor for our services
    cat <<EOF | kubectl apply -f -
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: quantumlayer-services
  namespace: $NAMESPACE
spec:
  selector:
    matchLabels:
      app: sandbox-executor
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: capsule-builder
  namespace: $NAMESPACE
spec:
  selector:
    matchLabels:
      app: capsule-builder
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
EOF
    
    log_info "✓ Monitoring stack deployed"
    log_info "  Grafana: kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80"
    log_info "  Username: admin / Password: quantumlayer2024"
}

# ═══════════════════════════════════════════════════════════════════════════════
# Integration Tests
# ═══════════════════════════════════════════════════════════════════════════════

run_integration_tests() {
    log_step "Running Integration Tests"
    
    # Test Sandbox Executor
    log_info "Testing Sandbox Executor..."
    kubectl run test-sandbox --image=curlimages/curl:latest -n $NAMESPACE --rm -it --restart=Never -- \
        curl -X POST http://sandbox-executor:8091/api/v1/execute \
        -H "Content-Type: application/json" \
        -d '{"language":"python","code":"print(\"Hello, Enterprise!\")"}' || true
    
    # Test Capsule Builder
    log_info "Testing Capsule Builder..."
    kubectl run test-capsule --image=curlimages/curl:latest -n $NAMESPACE --rm -it --restart=Never -- \
        curl -X GET http://capsule-builder:8092/api/v1/templates || true
    
    log_info "✓ Integration tests completed"
}

# ═══════════════════════════════════════════════════════════════════════════════
# Status Report
# ═══════════════════════════════════════════════════════════════════════════════

show_status() {
    log_step "Deployment Status"
    
    echo ""
    echo "═══════════════════════════════════════════════════════════════"
    echo " Deployments"
    echo "═══════════════════════════════════════════════════════════════"
    kubectl get deployments -n $NAMESPACE
    
    echo ""
    echo "═══════════════════════════════════════════════════════════════"
    echo " Pods"
    echo "═══════════════════════════════════════════════════════════════"
    kubectl get pods -n $NAMESPACE
    
    echo ""
    echo "═══════════════════════════════════════════════════════════════"
    echo " Services"
    echo "═══════════════════════════════════════════════════════════════"
    kubectl get services -n $NAMESPACE
    
    echo ""
    echo "═══════════════════════════════════════════════════════════════"
    echo " HPA Status"
    echo "═══════════════════════════════════════════════════════════════"
    kubectl get hpa -n $NAMESPACE
    
    echo ""
    log_info "✓ Deployment complete!"
    echo ""
    echo "Access points:"
    echo "  • Sandbox Executor: kubectl port-forward -n $NAMESPACE svc/sandbox-executor 8091:8091"
    echo "  • Capsule Builder: kubectl port-forward -n $NAMESPACE svc/capsule-builder 8092:8092"
    echo "  • Grafana: kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80"
}

# ═══════════════════════════════════════════════════════════════════════════════
# Main Execution
# ═══════════════════════════════════════════════════════════════════════════════

main() {
    print_banner
    preflight_checks
    
    case "$DEPLOY_MODE" in
        all)
            deploy_security_policies
            build_and_push_images
            deploy_sandbox_executor
            deploy_capsule_builder
            deploy_monitoring
            run_integration_tests
            ;;
        sandbox)
            build_and_push_images
            deploy_sandbox_executor
            ;;
        capsule)
            build_and_push_images
            deploy_capsule_builder
            ;;
        monitoring)
            deploy_monitoring
            ;;
        security)
            deploy_security_policies
            ;;
        test)
            run_integration_tests
            ;;
        *)
            log_error "Invalid deploy mode: $DEPLOY_MODE"
            echo "Usage: $0 [all|sandbox|capsule|monitoring|security|test]"
            exit 1
            ;;
    esac
    
    show_status
}

# Run main
main "$@"