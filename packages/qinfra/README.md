# 🚀 QInfra - Enterprise Infrastructure Automation Engine

## Overview

QInfra is a revolutionary infrastructure automation platform that manages entire data centers, automates SOPs, ensures compliance, and provides enterprise-grade infrastructure intelligence. It goes far beyond basic Infrastructure as Code (IaC) generation to provide comprehensive infrastructure management.

## 🌟 Key Features

### 1. **Data Center as Code (DCaaC)**
- Physical infrastructure management (racks, power, cooling)
- Hybrid cloud orchestration
- Multi-region deployment
- Disaster recovery automation

### 2. **Golden Image Factory**
- Automated image building pipeline
- Security hardening (CIS/STIG benchmarks)
- Compliance validation
- Version control and rollback
- SBOM generation

### 3. **SOP Automation**
- Runbook automation
- Incident response playbooks
- AI-powered SOP generation
- Approval workflows

### 4. **Vulnerability Management**
- Continuous infrastructure scanning
- CVE detection and tracking
- Automated remediation
- Security posture reporting

### 5. **Compliance Automation**
- Multi-framework support (SOC2, HIPAA, PCI-DSS, GDPR)
- Continuous compliance monitoring
- Automated evidence collection
- Audit report generation

### 6. **Cost Intelligence**
- Advanced cost optimization
- Spot instance orchestration
- Reserved instance planning
- Predictive cost analysis

## 🔧 Installation

### Build from source:
```bash
cd packages/qinfra
go build -o qinfra main.go
./qinfra
```

### Docker:
```bash
docker build -t qinfra:latest .
docker run -p 8095:8095 qinfra:latest
```

### Kubernetes:
```bash
kubectl apply -f ../../infrastructure/kubernetes/qinfra.yaml
```

## 📡 API Endpoints

### Core Infrastructure Generation

#### Generate Infrastructure
```bash
POST /generate

{
  "type": "cloud",
  "provider": "aws",
  "requirements": "High-availability web application with auto-scaling",
  "resources": [
    {
      "type": "compute",
      "name": "web-servers",
      "properties": {
        "instance_type": "t3.medium",
        "count": 3
      }
    }
  ],
  "compliance": ["SOC2", "HIPAA"],
  "golden_image": {
    "base_os": "ubuntu-22.04",
    "hardening": "CIS",
    "packages": ["nginx", "prometheus-node-exporter"]
  }
}

Response:
{
  "id": "infra-123",
  "status": "generated",
  "framework": "terraform",
  "code": {
    "main.tf": "...",
    "variables.tf": "...",
    "outputs.tf": "..."
  },
  "compliance_report": {
    "score": 95.5,
    "passed": 38,
    "failed": 2
  },
  "vulnerabilities": [],
  "optimizations": [
    {
      "type": "cost",
      "description": "Use spot instances",
      "savings": 1500.00
    }
  ]
}
```

### Golden Image Management

#### Build Golden Image
```bash
POST /golden-image/build

{
  "base_os": "ubuntu-22.04",
  "hardening": "CIS",
  "packages": ["nginx", "docker", "prometheus"],
  "compliance": ["SOC2", "HIPAA"],
  "validation": true
}

Response:
{
  "image_id": "img-abc123",
  "status": "building",
  "estimated_time": "15 minutes"
}
```

### SOP Automation

#### Generate SOP Runbook
```bash
POST /sop/generate

{
  "name": "Database Failover",
  "type": "incident",
  "steps": [
    {
      "name": "Verify primary failure",
      "command": "pg_isready -h primary.db",
      "validation": "exit_code != 0"
    },
    {
      "name": "Promote secondary",
      "command": "pg_ctl promote -D /var/lib/postgresql",
      "rollback": "pg_ctl demote"
    }
  ],
  "automation": true,
  "approvals": ["ops-team", "db-admin"]
}

Response:
{
  "id": "sop-456",
  "name": "Database Failover",
  "executable": true,
  "estimated_duration": "10 minutes"
}
```

### Vulnerability Scanning

#### Scan Infrastructure
```bash
POST /scan/infrastructure

{
  "code": {
    "main.tf": "resource \"aws_security_group\" \"web\" {\n  ingress {\n    from_port = 22\n    to_port = 22\n    protocol = \"tcp\"\n    cidr_blocks = [\"0.0.0.0/0\"]\n  }\n}"
  },
  "framework": "terraform"
}

Response:
{
  "vulnerabilities": [
    {
      "severity": "high",
      "cve": "CWE-284",
      "description": "Unrestricted network access detected",
      "affected": "main.tf",
      "fix": "Restrict CIDR blocks to specific IP ranges"
    }
  ],
  "scan_date": "2024-09-05T10:00:00Z"
}
```

### Compliance Validation

#### Validate Compliance
```bash
POST /compliance/validate

{
  "code": {
    "main.tf": "..."
  },
  "frameworks": ["SOC2", "HIPAA"]
}

Response:
{
  "framework": "SOC2, HIPAA",
  "score": 87.5,
  "passed": 35,
  "failed": 5,
  "findings": [...],
  "remediation": [
    "Enable encryption for data at rest",
    "Add audit logging"
  ]
}
```

### Cost Optimization

#### Get Cost Optimizations
```bash
POST /optimize/cost

{
  "type": "cloud",
  "provider": "aws",
  "resources": [...]
}

Response:
{
  "optimizations": [
    {
      "type": "cost",
      "description": "Use spot instances for non-critical workloads",
      "impact": "70% cost reduction",
      "savings": 1500.00
    },
    {
      "type": "cost", 
      "description": "Purchase 3-year reserved instances",
      "impact": "45% cost reduction",
      "savings": 3200.00
    }
  ],
  "total_monthly_savings": 5500.00,
  "roi_percentage": 55.0
}
```

### Data Center Planning

#### Plan Data Center
```bash
POST /datacenter/plan

{
  "requirements": "Support 1000 servers with N+1 redundancy"
}

Response:
{
  "racks": 10,
  "servers": 200,
  "network": "10Gbps redundant",
  "power": "2N+1 redundancy",
  "cooling": "N+1 CRAC units",
  "tier": "Tier III"
}
```

## 🏗️ Architecture

```
QInfra Engine
├── Core Engine
│   ├── Multi-provider orchestration
│   ├── Template management
│   └── State management
├── Golden Image Factory
│   ├── Packer integration
│   ├── Compliance validation
│   └── Registry management
├── SOP Automation
│   ├── Workflow engine
│   ├── Runbook library
│   └── Approval system
├── Vulnerability Scanner
│   ├── Trivy/Grype integration
│   ├── CVE database
│   └── Remediation workflows
├── Compliance Manager
│   ├── Policy as Code
│   ├── Framework mappings
│   └── Report generation
└── Cost Intelligence
    ├── Pricing APIs
    ├── Usage analytics
    └── Optimization algorithms
```

## 🚀 Quick Start Examples

### 1. Generate AWS Infrastructure with Compliance
```bash
curl -X POST http://localhost:8095/generate \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "aws",
    "requirements": "Secure web application",
    "compliance": ["PCI-DSS"]
  }'
```

### 2. Build Hardened Golden Image
```bash
curl -X POST http://localhost:8095/golden-image/build \
  -H "Content-Type: application/json" \
  -d '{
    "base_os": "rhel-8",
    "hardening": "STIG"
  }'
```

### 3. Get Cost Optimizations
```bash
curl -X POST http://localhost:8095/optimize/cost \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "aws",
    "type": "cloud"
  }'
```

## 🔐 Security Features

- **Zero-trust architecture**: All requests authenticated and authorized
- **Vulnerability scanning**: Continuous security assessment
- **Compliance validation**: Multi-framework support
- **Secret management**: Integration with HashiCorp Vault
- **Audit logging**: Complete audit trail

## 📈 Metrics & Monitoring

QInfra exposes Prometheus metrics at `/metrics`:

- `qinfra_infra_generated_total`: Total infrastructure generations
- `qinfra_compliance_score`: Current compliance score
- `qinfra_vulnerabilities_detected`: Number of vulnerabilities found
- `qinfra_cost_savings_usd`: Estimated cost savings
- `qinfra_golden_images_built`: Golden images created

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## 📜 License

QInfra is part of the QuantumLayer Platform. See [LICENSE](../../LICENSE) for details.

## 🆘 Support

- Documentation: [QInfra Docs](https://quantumlayer.dev/docs/qinfra)
- Issues: [GitHub Issues](https://github.com/quantumlayer/platform/issues)
- Discord: [Join our community](https://discord.gg/quantumlayer)

---

**"Infrastructure that manages itself"** - QInfra Team