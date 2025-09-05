# QInfra Architecture Guide

## System Architecture Overview

QInfra is built on a microservices architecture that provides enterprise-grade infrastructure resilience capabilities. The platform consists of multiple layers working together to deliver golden image management, patch intelligence, drift detection, and compliance automation.

```
┌─────────────────────────────────────────────────────────────────────┐
│                         External Clients                             │
│  (CLI, SDK, Web UI, CI/CD, Terraform, Ansible)                     │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          API Gateway                                 │
│              (Authentication, Rate Limiting, Routing)                │
└─────────────────────────────────────────────────────────────────────┘
                                    │
        ┌───────────────┬───────────┴───────────┬───────────────┐
        ▼               ▼                       ▼               ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│   Image      │ │    Patch     │ │    Drift     │ │  Compliance  │
│  Registry    │ │   Manager    │ │   Engine     │ │  Validator   │
│   Service    │ │   Service    │ │   Service    │ │   Service    │
└──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘
        │               │                 │               │
        └───────────────┴─────────────────┴───────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Temporal Workflow Engine                         │
│         (Orchestration, State Management, Retry Logic)               │
└─────────────────────────────────────────────────────────────────────┘
                                    │
        ┌───────────────┬───────────┴───────────┬───────────────┐
        ▼               ▼                       ▼               ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│    Docker    │ │  PostgreSQL  │ │    Redis     │ │     S3       │
│   Registry   │ │   Database   │ │    Cache     │ │   Storage    │
└──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘
```

## Component Architecture

### 1. API Gateway Layer

**Purpose:** Single entry point for all client requests

**Components:**
- **Kong/Nginx:** HTTP routing and load balancing
- **OAuth2/OIDC:** Authentication and authorization
- **Rate Limiter:** Request throttling and quota management
- **API Versioning:** Version management and deprecation

**Key Features:**
- Request validation
- Response caching
- Circuit breaking
- Request/response logging
- API key management

### 2. Service Layer

#### Image Registry Service

**Purpose:** Manages golden image lifecycle

**Architecture:**
```
┌─────────────────────────────────────────┐
│         Image Registry Service          │
├─────────────────────────────────────────┤
│  API Handler                            │
│    ├── Build Controller                 │
│    ├── Scan Controller                  │
│    ├── Sign Controller                  │
│    └── Query Controller                 │
├─────────────────────────────────────────┤
│  Business Logic                         │
│    ├── Image Builder                    │
│    ├── Vulnerability Scanner            │
│    ├── Attestation Manager              │
│    └── Compliance Checker               │
├─────────────────────────────────────────┤
│  Data Access Layer                      │
│    ├── Image Repository                 │
│    ├── Registry Client                  │
│    └── Database Client                  │
└─────────────────────────────────────────┘
```

**Key Responsibilities:**
- Golden image building with Packer
- SBOM generation
- Vulnerability scanning
- Cryptographic signing
- Image versioning

#### Patch Manager Service

**Purpose:** Intelligent patch orchestration

**Architecture:**
```
┌─────────────────────────────────────────┐
│         Patch Manager Service           │
├─────────────────────────────────────────┤
│  CVE Tracker                            │
│    ├── NVD Client                       │
│    ├── OSV Client                       │
│    └── Vendor Feeds                     │
├─────────────────────────────────────────┤
│  Patch Orchestrator                     │
│    ├── Risk Scorer                      │
│    ├── Test Runner                      │
│    └── Deployment Manager               │
├─────────────────────────────────────────┤
│  Rollback Engine                        │
│    ├── State Manager                    │
│    └── Recovery Handler                 │
└─────────────────────────────────────────┘
```

**Key Responsibilities:**
- Real-time CVE monitoring
- Risk assessment (CVSS scoring)
- Automated patch testing
- Canary deployments
- Rollback orchestration

#### Drift Engine Service

**Purpose:** Continuous configuration monitoring

**Architecture:**
```
┌─────────────────────────────────────────┐
│           Drift Engine Service          │
├─────────────────────────────────────────┤
│  Scanner                                │
│    ├── Agent Manager                    │
│    ├── State Collector                  │
│    └── Diff Calculator                  │
├─────────────────────────────────────────┤
│  Analyzer                               │
│    ├── Drift Classifier                 │
│    ├── Severity Scorer                  │
│    └── Trend Analyzer                   │
├─────────────────────────────────────────┤
│  Remediator                             │
│    ├── Auto-fix Engine                  │
│    ├── Approval Manager                 │
│    └── Scheduler                        │
└─────────────────────────────────────────┘
```

**Key Responsibilities:**
- Baseline establishment
- Continuous scanning
- Drift classification
- Automated remediation
- Compliance reporting

#### Compliance Validator Service

**Purpose:** Framework compliance validation

**Architecture:**
```
┌─────────────────────────────────────────┐
│       Compliance Validator Service       │
├─────────────────────────────────────────┤
│  Framework Engine                       │
│    ├── SOC2 Validator                   │
│    ├── HIPAA Validator                  │
│    ├── PCI-DSS Validator                │
│    └── Custom Validators                │
├─────────────────────────────────────────┤
│  Evidence Collector                     │
│    ├── Config Snapshots                 │
│    ├── Audit Logs                       │
│    └── Scan Results                     │
├─────────────────────────────────────────┤
│  Report Generator                       │
│    ├── PDF Builder                      │
│    ├── CSV Exporter                     │
│    └── Dashboard API                    │
└─────────────────────────────────────────┘
```

### 3. Orchestration Layer

#### Temporal Workflows

**Purpose:** Complex workflow orchestration

**Key Workflows:**

```go
// Infrastructure Generation Workflow
InfrastructureGenerationWorkflow
├── AnalyzeCodeActivity
├── BuildGoldenImageActivity
├── ScanVulnerabilityActivity
├── SignImageActivity
├── ValidateComplianceActivity
├── DeployInfrastructureActivity
└── MonitorDriftActivity

// Patch Management Workflow
PatchManagementWorkflow
├── IdentifyCVEsActivity
├── AssessRiskActivity
├── TestPatchActivity
├── ApprovalActivity
├── DeployCanaryActivity
├── ValidateDeploymentActivity
└── RolloutActivity

// Drift Remediation Workflow
DriftRemediationWorkflow
├── DetectDriftActivity
├── ClassifyDriftActivity
├── GenerateFixActivity
├── ApprovalActivity
├── ApplyFixActivity
└── ValidateFixActivity
```

### 4. Data Layer

#### PostgreSQL Schema

```sql
-- Golden Images
CREATE TABLE golden_images (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    base_os VARCHAR(100),
    platform VARCHAR(50),
    hardening VARCHAR(50),
    registry_url TEXT,
    digest VARCHAR(255),
    size BIGINT,
    build_time TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Vulnerabilities
CREATE TABLE vulnerabilities (
    id UUID PRIMARY KEY,
    image_id UUID REFERENCES golden_images(id),
    cve VARCHAR(50),
    severity VARCHAR(20),
    cvss_score DECIMAL(3,1),
    description TEXT,
    fix_version VARCHAR(50),
    detected_at TIMESTAMP DEFAULT NOW()
);

-- Drift Records
CREATE TABLE drift_records (
    id UUID PRIMARY KEY,
    node_id VARCHAR(255),
    expected_image_id UUID REFERENCES golden_images(id),
    current_state JSONB,
    drift_type VARCHAR(50),
    severity VARCHAR(20),
    detected_at TIMESTAMP DEFAULT NOW(),
    remediated_at TIMESTAMP
);

-- Compliance Results
CREATE TABLE compliance_results (
    id UUID PRIMARY KEY,
    image_id UUID REFERENCES golden_images(id),
    framework VARCHAR(50),
    score DECIMAL(5,2),
    passed_controls INT,
    failed_controls INT,
    total_controls INT,
    details JSONB,
    validated_at TIMESTAMP DEFAULT NOW()
);

-- Patch History
CREATE TABLE patch_history (
    id UUID PRIMARY KEY,
    image_id UUID REFERENCES golden_images(id),
    cve VARCHAR(50),
    patch_version VARCHAR(50),
    applied_at TIMESTAMP,
    applied_by VARCHAR(255),
    status VARCHAR(50),
    rollback_id UUID REFERENCES patch_history(id)
);
```

#### Redis Cache Structure

```redis
# Image cache
SET image:<id> {JSON} EX 3600

# Drift detection cache
SET drift:<platform>:<datacenter> {JSON} EX 300

# CVE cache
SET cve:<id> {JSON} EX 86400

# Compliance cache
HSET compliance:<framework> <image_id> {score}

# Metrics
INCR metrics:images:built
INCR metrics:scans:completed
INCR metrics:drift:detected
```

#### S3/MinIO Storage

```
bucket-structure/
├── golden-images/
│   ├── aws/
│   │   ├── ubuntu-22.04/
│   │   └── rhel-8/
│   ├── azure/
│   ├── gcp/
│   └── vmware/
├── sboms/
│   └── <image-id>/
│       └── sbom.json
├── scan-results/
│   └── <scan-id>/
│       └── report.json
├── compliance-reports/
│   └── <year>/<month>/
│       └── report.pdf
└── attestations/
    └── <image-id>/
        └── signature.sig
```

## Security Architecture

### Authentication & Authorization

```
┌─────────────────────────────────────────┐
│            Identity Provider             │
│         (Keycloak/Auth0/Okta)           │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│           OAuth2/OIDC Flow               │
├─────────────────────────────────────────┤
│  1. Client requests token                │
│  2. IDP validates credentials            │
│  3. IDP issues JWT token                 │
│  4. Client includes token in requests   │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│            API Gateway                   │
├─────────────────────────────────────────┤
│  1. Validate JWT signature              │
│  2. Check token expiry                  │
│  3. Extract user claims                 │
│  4. Apply RBAC policies                 │
└─────────────────────────────────────────┘
```

### RBAC Model

```yaml
roles:
  admin:
    permissions:
      - images:*
      - drift:*
      - compliance:*
      - patches:*
  
  operator:
    permissions:
      - images:read
      - images:build
      - drift:read
      - drift:remediate
      - patches:apply
  
  viewer:
    permissions:
      - images:read
      - drift:read
      - compliance:read
      - patches:read

  auditor:
    permissions:
      - compliance:*
      - images:read
      - drift:read
```

### Encryption

- **At Rest:** AES-256-GCM for database and storage
- **In Transit:** TLS 1.3 for all communications
- **Key Management:** HashiCorp Vault or AWS KMS
- **Secrets:** Kubernetes Secrets with encryption at rest

## Scalability Architecture

### Horizontal Scaling

```yaml
# Auto-scaling configuration
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: image-registry-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: image-registry
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
```

### Load Balancing

```nginx
upstream image_registry {
    least_conn;
    server image-registry-1:8096 weight=5;
    server image-registry-2:8096 weight=5;
    server image-registry-3:8096 weight=5;
    
    keepalive 32;
    keepalive_timeout 60s;
}
```

### Caching Strategy

1. **L1 Cache:** Application-level (in-memory)
2. **L2 Cache:** Redis (distributed)
3. **L3 Cache:** CDN for static assets

### Database Optimization

- **Read Replicas:** For read-heavy operations
- **Connection Pooling:** PgBouncer for PostgreSQL
- **Partitioning:** Time-based for historical data
- **Indexing:** Strategic indexes on frequently queried columns

## High Availability Architecture

### Multi-Region Deployment

```
┌──────────────────────────────────────────────────────┐
│                    Global Load Balancer               │
└──────────────────────────────────────────────────────┘
            │                    │                    │
            ▼                    ▼                    ▼
    ┌─────────────┐      ┌─────────────┐      ┌─────────────┐
    │  US-East-1  │      │  EU-West-1  │      │ AP-South-1  │
    │   Primary   │      │   Secondary │      │   Secondary │
    └─────────────┘      └─────────────┘      └─────────────┘
            │                    │                    │
            └────────────────────┴────────────────────┘
                                │
                    ┌───────────────────────┐
                    │   Data Replication    │
                    │  (PostgreSQL Streaming)│
                    └───────────────────────┘
```

### Disaster Recovery

**RTO Target:** 15 minutes  
**RPO Target:** 5 minutes

```yaml
disaster_recovery:
  strategy: active_passive
  
  backup:
    frequency: hourly
    retention: 30_days
    locations:
      - s3://backup-primary
      - s3://backup-secondary
  
  replication:
    type: streaming
    lag_threshold: 5_minutes
    
  failover:
    automatic: true
    health_checks:
      - database_connectivity
      - api_response_time
      - storage_availability
```

## Monitoring & Observability

### Metrics Collection

```prometheus
# Golden image metrics
qinfra_images_built_total
qinfra_image_build_duration_seconds
qinfra_image_scan_vulnerabilities_total

# Drift metrics
qinfra_drift_detected_total
qinfra_drift_remediation_duration_seconds
qinfra_drift_percentage

# Compliance metrics
qinfra_compliance_score
qinfra_compliance_violations_total

# Performance metrics
qinfra_api_request_duration_seconds
qinfra_api_requests_total
qinfra_api_errors_total
```

### Logging Architecture

```
┌─────────────────────────────────────────┐
│           Application Logs               │
│         (Structured JSON)                │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│            Fluent Bit                    │
│      (Collection & Forwarding)           │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│          Elasticsearch                   │
│         (Storage & Indexing)             │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│             Kibana                       │
│       (Visualization & Search)           │
└─────────────────────────────────────────┘
```

### Distributed Tracing

```
┌─────────────────────────────────────────┐
│        OpenTelemetry SDK                 │
│         (Instrumentation)                │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│         OpenTelemetry Collector          │
│          (Processing & Export)           │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│              Jaeger                      │
│         (Storage & Analysis)             │
└─────────────────────────────────────────┘
```

## Integration Architecture

### CI/CD Integration

```yaml
# GitLab CI Integration
.qinfra_scan:
  script:
    - qinfra scan --image $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA
    - qinfra validate --compliance SOC2,HIPAA
    - qinfra sign --key $SIGNING_KEY

# GitHub Actions Integration  
- name: QInfra Security Scan
  uses: quantumlayer/qinfra-action@v1
  with:
    image: ${{ env.IMAGE_TAG }}
    compliance: SOC2,HIPAA
    fail-on: critical
```

### IaC Integration

```hcl
# Terraform Provider
terraform {
  required_providers {
    qinfra = {
      source = "quantumlayer/qinfra"
      version = "~> 1.0"
    }
  }
}

# Golden Image Data Source
data "qinfra_golden_image" "ubuntu" {
  platform = "aws"
  base_os = "ubuntu-22.04"
  compliance = ["SOC2", "HIPAA"]
  latest = true
}

# Use in EC2 Instance
resource "aws_instance" "web" {
  ami = data.qinfra_golden_image.ubuntu.ami_id
  instance_type = "t3.medium"
}
```

### Kubernetes Operator

```yaml
apiVersion: qinfra.quantumlayer.io/v1
kind: GoldenImage
metadata:
  name: ubuntu-golden
spec:
  baseOS: ubuntu-22.04
  platform: kubernetes
  hardening: CIS
  compliance:
    - SOC2
    - HIPAA
  autoUpdate: true
  scanSchedule: "0 */6 * * *"
```

## Performance Optimization

### Query Optimization

```sql
-- Optimized drift detection query
WITH latest_images AS (
    SELECT DISTINCT ON (platform, base_os)
        id, version, platform, base_os
    FROM golden_images
    ORDER BY platform, base_os, created_at DESC
)
SELECT 
    d.node_id,
    d.drift_type,
    li.version as expected_version,
    d.current_state->>'version' as current_version
FROM drift_records d
JOIN latest_images li ON d.expected_image_id = li.id
WHERE d.remediated_at IS NULL
AND d.severity IN ('critical', 'high')
ORDER BY d.detected_at DESC
LIMIT 100;
```

### Caching Strategy

```go
// Multi-level caching
func GetImage(id string) (*Image, error) {
    // L1: In-memory cache
    if img := memCache.Get(id); img != nil {
        return img, nil
    }
    
    // L2: Redis cache
    if img := redisCache.Get(id); img != nil {
        memCache.Set(id, img)
        return img, nil
    }
    
    // L3: Database
    img, err := db.GetImage(id)
    if err != nil {
        return nil, err
    }
    
    // Update caches
    redisCache.Set(id, img, 1*time.Hour)
    memCache.Set(id, img)
    
    return img, nil
}
```

## Deployment Architecture

### Kubernetes Deployment

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: qinfra
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: image-registry
  namespace: qinfra
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: image-registry
  template:
    metadata:
      labels:
        app: image-registry
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchExpressions:
              - key: app
                operator: In
                values:
                - image-registry
            topologyKey: kubernetes.io/hostname
      containers:
      - name: image-registry
        image: ghcr.io/quantumlayer/image-registry:latest
        ports:
        - containerPort: 8096
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: qinfra-secrets
              key: database-url
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8096
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8096
          initialDelaySeconds: 5
          periodSeconds: 5
```

## Network Architecture

### Service Mesh (Istio)

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: qinfra
spec:
  hosts:
  - qinfra.quantumlayer.io
  http:
  - match:
    - uri:
        prefix: /api/v1/images
    route:
    - destination:
        host: image-registry
        port:
          number: 8096
      weight: 100
  - match:
    - uri:
        prefix: /api/v1/drift
    route:
    - destination:
        host: drift-engine
        port:
          number: 8097
      weight: 100
```

### Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: qinfra-network-policy
  namespace: qinfra
spec:
  podSelector:
    matchLabels:
      app: image-registry
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: api-gateway
    ports:
    - protocol: TCP
      port: 8096
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: database
    ports:
    - protocol: TCP
      port: 5432
```

---

**Architecture Version:** 1.0.0  
**Last Updated:** 2024-09-05  
**Maintained By:** Platform Engineering Team