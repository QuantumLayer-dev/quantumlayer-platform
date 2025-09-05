# QInfra API Reference

## Base Configuration

### Base URL
```
Production: https://api.quantumlayer.io/qinfra/v1
Development: http://localhost:30096
```

### Authentication
```http
Authorization: Bearer <token>
X-API-Key: <api-key>
```

### Content Type
```http
Content-Type: application/json
Accept: application/json
```

### Rate Limiting
- 1000 requests per hour per API key
- 100 concurrent requests maximum
- Burst: 50 requests per second

## Golden Image APIs

### Build Golden Image

Creates a new golden image with specified configuration.

**Endpoint:** `POST /images/build`

**Request Body:**
```json
{
  "name": "string",           // Required: Image name
  "base_os": "string",        // Required: Base OS (ubuntu-22.04, rhel-8, etc.)
  "platform": "string",       // Required: Target platform (aws|azure|gcp|vmware|docker)
  "packages": ["string"],     // Optional: Packages to install
  "hardening": "string",      // Optional: Hardening standard (CIS|STIG|custom)
  "compliance": ["string"],   // Optional: Compliance frameworks
  "scripts": ["string"],      // Optional: Custom scripts to run
  "metadata": {              // Optional: Additional metadata
    "key": "value"
  }
}
```

**Response:** `202 Accepted`
```json
{
  "id": "uuid",
  "status": "building",
  "message": "Golden image build initiated",
  "estimated_time": "10-15 minutes",
  "image": {
    "id": "uuid",
    "name": "ubuntu-22.04-golden",
    "version": "1.0.0",
    "platform": "aws",
    "registry_url": "registry.url/image:tag",
    "build_time": "2024-01-01T00:00:00Z"
  }
}
```

**Error Responses:**
- `400 Bad Request`: Invalid input parameters
- `409 Conflict`: Image with same name already exists
- `500 Internal Server Error`: Build failed

---

### List Golden Images

Retrieves all golden images with optional filtering.

**Endpoint:** `GET /images`

**Query Parameters:**
- `platform` (string): Filter by platform
- `compliance` (string): Filter by compliance framework
- `hardening` (string): Filter by hardening standard
- `page` (int): Page number (default: 1)
- `limit` (int): Items per page (default: 20, max: 100)
- `sort` (string): Sort field (name|build_time|version)
- `order` (string): Sort order (asc|desc)

**Response:** `200 OK`
```json
{
  "total": 50,
  "page": 1,
  "limit": 20,
  "images": [
    {
      "id": "uuid",
      "name": "ubuntu-22.04-golden",
      "version": "1.0.0",
      "base_os": "ubuntu-22.04",
      "platform": "aws",
      "packages": ["nginx", "docker"],
      "hardening": "CIS",
      "compliance": ["SOC2", "HIPAA"],
      "registry_url": "registry.url/image:tag",
      "digest": "sha256:abc123",
      "size": 524288000,
      "build_time": "2024-01-01T00:00:00Z",
      "last_scanned": "2024-01-01T00:00:00Z"
    }
  ]
}
```

---

### Get Golden Image

Retrieves detailed information about a specific golden image.

**Endpoint:** `GET /images/{id}`

**Response:** `200 OK`
```json
{
  "id": "uuid",
  "name": "ubuntu-22.04-golden",
  "version": "1.0.0",
  "base_os": "ubuntu-22.04",
  "platform": "aws",
  "packages": ["nginx", "docker", "prometheus"],
  "hardening": "CIS",
  "compliance": ["SOC2", "HIPAA"],
  "registry_url": "registry.url/image:tag",
  "digest": "sha256:abc123",
  "size": 524288000,
  "sbom": {
    "packages": [
      {
        "name": "nginx",
        "version": "1.24.0",
        "type": "deb"
      }
    ]
  },
  "vulnerabilities": [
    {
      "id": "CVE-2024-12345",
      "severity": "medium",
      "description": "Sample vulnerability",
      "fix_version": "1.24.1"
    }
  ],
  "attestation": {
    "signature": "sha256:xyz789",
    "signed_by": "quantumlayer-ca",
    "signed_at": "2024-01-01T00:00:00Z",
    "verified": true
  },
  "build_time": "2024-01-01T00:00:00Z",
  "last_scanned": "2024-01-02T00:00:00Z",
  "metadata": {
    "team": "platform",
    "environment": "production"
  }
}
```

**Error Responses:**
- `404 Not Found`: Image not found

---

### Scan Golden Image

Performs vulnerability scanning on a golden image.

**Endpoint:** `POST /images/{id}/scan`

**Request Body:** (Optional)
```json
{
  "scan_type": "full|quick",  // Default: full
  "severity_threshold": "critical|high|medium|low"  // Default: low
}
```

**Response:** `200 OK`
```json
{
  "id": "scan-uuid",
  "image_id": "image-uuid",
  "status": "completed",
  "scan_time": "2024-01-01T00:00:00Z",
  "duration": "45s",
  "vulnerabilities_found": 5,
  "vulnerabilities": [
    {
      "id": "CVE-2024-12345",
      "cve": "CVE-2024-12345",
      "severity": "critical",
      "cvss_score": 9.8,
      "description": "Remote code execution vulnerability",
      "affected_package": "openssl",
      "installed_version": "1.1.1",
      "fix_version": "1.1.1w",
      "references": [
        "https://nvd.nist.gov/vuln/detail/CVE-2024-12345"
      ]
    }
  ],
  "summary": {
    "critical": 1,
    "high": 2,
    "medium": 2,
    "low": 0
  }
}
```

---

### Sign Golden Image

Cryptographically signs a golden image for attestation.

**Endpoint:** `POST /images/{id}/sign`

**Request Body:** (Optional)
```json
{
  "key_id": "string",        // Optional: Signing key ID
  "algorithm": "string"      // Optional: Signing algorithm (default: RSA256)
}
```

**Response:** `200 OK`
```json
{
  "status": "signed",
  "image_id": "uuid",
  "attestation": {
    "signature": "MEUCIQDx...",
    "signed_by": "quantumlayer-ca",
    "key_id": "key-123",
    "algorithm": "RSA256",
    "signed_at": "2024-01-01T00:00:00Z",
    "verified": true,
    "verification_url": "https://verify.quantumlayer.io/attestation/uuid"
  }
}
```

---

### Get Patch Status

Checks if a golden image needs patches or updates.

**Endpoint:** `GET /images/{id}/patch-status`

**Response:** `200 OK`
```json
{
  "image_id": "uuid",
  "current_version": "1.0.0",
  "latest_version": "1.0.2",
  "up_to_date": false,
  "patches_needed": 5,
  "patches": [
    {
      "cve": "CVE-2024-12345",
      "severity": "critical",
      "package": "openssl",
      "current": "1.1.1",
      "required": "1.1.1w"
    }
  ],
  "last_checked": "2024-01-01T00:00:00Z",
  "next_check": "2024-01-01T06:00:00Z"
}
```

---

### Delete Golden Image

Deletes a golden image from the registry.

**Endpoint:** `DELETE /images/{id}`

**Response:** `200 OK`
```json
{
  "id": "uuid",
  "status": "deleted",
  "message": "Image successfully deleted"
}
```

**Error Responses:**
- `404 Not Found`: Image not found
- `409 Conflict`: Image is in use and cannot be deleted

---

## Drift Detection APIs

### Detect Infrastructure Drift

Analyzes infrastructure for configuration drift.

**Endpoint:** `POST /drift/detect`

**Request Body:**
```json
{
  "platform": "string",      // Required: Platform to check
  "datacenter": "string",    // Optional: Specific datacenter
  "environment": "string",   // Optional: Environment (dev|staging|prod)
  "image_ids": ["string"],   // Optional: Specific images to check
  "deep_scan": boolean       // Optional: Perform deep scan (default: false)
}
```

**Response:** `200 OK`
```json
{
  "scan_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "total_nodes": 100,
  "scanned_nodes": 100,
  "drifted_nodes": 5,
  "drift_percentage": 5.0,
  "details": [
    {
      "node_id": "node-001",
      "hostname": "web-server-01",
      "current_image": "ubuntu-20.04-v1.0.0",
      "expected_image": "ubuntu-20.04-v1.0.2",
      "drift_type": "version",
      "drift_details": {
        "packages_added": ["vim"],
        "packages_removed": [],
        "packages_modified": ["nginx"],
        "config_changes": ["/etc/nginx/nginx.conf"],
        "permission_changes": ["/var/log/nginx"]
      },
      "severity": "high",
      "remediation": "Update to latest golden image",
      "auto_fixable": true
    }
  ],
  "summary": {
    "version_drift": 2,
    "package_drift": 3,
    "config_drift": 1,
    "permission_drift": 1
  }
}
```

---

### Remediate Drift

Fixes detected drift automatically or schedules remediation.

**Endpoint:** `POST /drift/remediate`

**Request Body:**
```json
{
  "scan_id": "string",        // Required: Drift scan ID
  "node_ids": ["string"],     // Optional: Specific nodes to fix
  "strategy": "string",       // Required: immediate|scheduled|manual
  "schedule": "string",       // Required if strategy=scheduled (cron format)
  "approval": {              // Optional: Approval requirements
    "required": boolean,
    "approvers": ["email"]
  }
}
```

**Response:** `202 Accepted`
```json
{
  "remediation_id": "uuid",
  "status": "scheduled",
  "strategy": "scheduled",
  "schedule": "0 2 * * *",
  "affected_nodes": 5,
  "estimated_time": "30 minutes",
  "message": "Drift remediation scheduled for maintenance window"
}
```

---

## Platform APIs

### Get Images by Platform

Retrieves golden images for a specific platform.

**Endpoint:** `GET /images/platform/{platform}`

**Path Parameters:**
- `platform`: aws|azure|gcp|vmware|docker|kubernetes

**Response:** `200 OK`
```json
{
  "platform": "aws",
  "total": 10,
  "images": [
    {
      "id": "uuid",
      "name": "ubuntu-22.04-golden",
      "version": "1.0.0",
      "ami_id": "ami-12345678",
      "region": "us-east-1",
      "compliance": ["SOC2", "HIPAA"]
    }
  ]
}
```

---

## Compliance APIs

### Get Compliant Images

Retrieves images compliant with specific framework.

**Endpoint:** `GET /images/compliance/{framework}`

**Path Parameters:**
- `framework`: SOC2|HIPAA|PCI-DSS|GDPR|FedRAMP

**Response:** `200 OK`
```json
{
  "framework": "SOC2",
  "total": 8,
  "images": [
    {
      "id": "uuid",
      "name": "ubuntu-22.04-golden",
      "version": "1.0.0",
      "compliance_score": 98.5,
      "passed_controls": 195,
      "total_controls": 198,
      "last_validated": "2024-01-01T00:00:00Z"
    }
  ]
}
```

---

### Validate Compliance

Validates an image against compliance framework.

**Endpoint:** `POST /compliance/validate`

**Request Body:**
```json
{
  "image_id": "string",       // Required: Image to validate
  "frameworks": ["string"],   // Required: Frameworks to check
  "detailed": boolean        // Optional: Return detailed results
}
```

**Response:** `200 OK`
```json
{
  "image_id": "uuid",
  "validation_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "results": [
    {
      "framework": "SOC2",
      "compliant": true,
      "score": 98.5,
      "passed": 195,
      "failed": 3,
      "total": 198,
      "failed_controls": [
        {
          "id": "CC6.1",
          "description": "Logical access controls",
          "severity": "medium",
          "remediation": "Enable MFA for all users"
        }
      ]
    }
  ]
}
```

---

## Metrics API

### Get Service Metrics

Returns service health and usage metrics.

**Endpoint:** `GET /metrics`

**Response:** `200 OK`
```json
{
  "service": "qinfra",
  "version": "1.0.0",
  "uptime": "7d 14h 32m",
  "metrics": {
    "total_images": 50,
    "images_by_platform": {
      "aws": 15,
      "azure": 10,
      "gcp": 10,
      "vmware": 10,
      "docker": 5
    },
    "compliance_coverage": {
      "SOC2": 45,
      "HIPAA": 30,
      "PCI-DSS": 20
    },
    "vulnerabilities": {
      "critical": 0,
      "high": 5,
      "medium": 15,
      "low": 30
    },
    "drift_percentage": 2.5,
    "patch_compliance": 98.5
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

---

## Webhooks

### Configure Webhooks

QInfra can send notifications to external systems.

**Supported Events:**
- `image.built`: Golden image build completed
- `image.scanned`: Vulnerability scan completed
- `drift.detected`: Configuration drift detected
- `patch.available`: New patches available
- `compliance.failed`: Compliance validation failed

**Webhook Payload:**
```json
{
  "event": "drift.detected",
  "timestamp": "2024-01-01T00:00:00Z",
  "data": {
    "scan_id": "uuid",
    "drifted_nodes": 5,
    "severity": "high"
  }
}
```

---

## Error Responses

### Standard Error Format
```json
{
  "error": {
    "code": "INVALID_INPUT",
    "message": "Invalid platform specified",
    "details": {
      "field": "platform",
      "value": "invalid",
      "allowed": ["aws", "azure", "gcp", "vmware", "docker"]
    }
  },
  "request_id": "req-uuid",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| INVALID_INPUT | 400 | Invalid request parameters |
| UNAUTHORIZED | 401 | Authentication required |
| FORBIDDEN | 403 | Insufficient permissions |
| NOT_FOUND | 404 | Resource not found |
| CONFLICT | 409 | Resource conflict |
| RATE_LIMITED | 429 | Too many requests |
| INTERNAL_ERROR | 500 | Internal server error |
| SERVICE_UNAVAILABLE | 503 | Service temporarily unavailable |

---

## SDK Examples

### Python
```python
import qinfra

client = qinfra.Client(api_key="your-api-key")

# Build golden image
image = client.images.build(
    name="ubuntu-golden",
    base_os="ubuntu-22.04",
    platform="aws",
    hardening="CIS",
    compliance=["SOC2", "HIPAA"]
)

# Scan for vulnerabilities
scan_result = client.images.scan(image.id)

# Detect drift
drift = client.drift.detect(
    platform="aws",
    environment="production"
)
```

### Go
```go
package main

import "github.com/quantumlayer/qinfra-go"

func main() {
    client := qinfra.NewClient("your-api-key")
    
    // Build golden image
    image, err := client.Images.Build(&qinfra.BuildRequest{
        Name:       "ubuntu-golden",
        BaseOS:     "ubuntu-22.04",
        Platform:   "aws",
        Hardening:  "CIS",
        Compliance: []string{"SOC2", "HIPAA"},
    })
    
    // Detect drift
    drift, err := client.Drift.Detect(&qinfra.DriftRequest{
        Platform:    "aws",
        Environment: "production",
    })
}
```

### JavaScript/TypeScript
```typescript
import { QInfraClient } from '@quantumlayer/qinfra';

const client = new QInfraClient({ apiKey: 'your-api-key' });

// Build golden image
const image = await client.images.build({
  name: 'ubuntu-golden',
  baseOS: 'ubuntu-22.04',
  platform: 'aws',
  hardening: 'CIS',
  compliance: ['SOC2', 'HIPAA']
});

// Scan for vulnerabilities
const scanResult = await client.images.scan(image.id);

// Detect drift
const drift = await client.drift.detect({
  platform: 'aws',
  environment: 'production'
});
```

---

## Rate Limiting

### Headers
```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1640995200
```

### Backoff Strategy
```
Retry-After: 60
```

Recommended exponential backoff:
- 1st retry: 1 second
- 2nd retry: 2 seconds
- 3rd retry: 4 seconds
- Max retries: 5

---

## Versioning

The API uses URL versioning:
- Current: `/v1`
- Beta: `/v2-beta`
- Deprecated: `/v0` (sunset: 2024-12-31)

### Deprecation Policy
- 6 months notice before deprecation
- 12 months support after deprecation announcement
- Migration guides provided

---

**Last Updated:** 2024-09-05  
**API Version:** 1.0.0