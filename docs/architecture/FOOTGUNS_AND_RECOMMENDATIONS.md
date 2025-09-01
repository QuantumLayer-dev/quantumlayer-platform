# ⚠️ QuantumLayer V2 - Architecture Footguns & Recommendations

## Executive Summary
Critical architectural decisions and anti-patterns to avoid, ensuring scalability, security, and reliability from day one.

---

## 1. 🔫 Provider Quota Management

### The Footgun
Provider rate limits causing cascading failures and thundering herds when one provider fails.

### Solution: Token Bucket + Dynamic Rerouting
```go
// providers/quota_manager.go
package providers

import (
    "sync"
    "time"
    "golang.org/x/time/rate"
)

type ProviderQuotaManager struct {
    buckets  map[string]*TokenBucket
    fallback *FallbackChain
    mu       sync.RWMutex
}

type TokenBucket struct {
    limiter    *rate.Limiter
    capacity   int
    refillRate time.Duration
    provider   string
    healthy    bool
}

func NewProviderQuotaManager() *ProviderQuotaManager {
    return &ProviderQuotaManager{
        buckets: map[string]*TokenBucket{
            "openai": {
                limiter:    rate.NewLimiter(rate.Every(60*time.Second/10000), 10000), // 10k/min
                capacity:   10000,
                refillRate: time.Minute,
                provider:   "openai",
                healthy:    true,
            },
            "anthropic": {
                limiter:    rate.NewLimiter(rate.Every(60*time.Second/1000), 1000), // 1k/min
                capacity:   1000,
                refillRate: time.Minute,
                provider:   "anthropic",
                healthy:    true,
            },
            "groq": {
                limiter:    rate.NewLimiter(rate.Every(60*time.Second/30000), 30000), // 30k/min
                capacity:   30000,
                refillRate: time.Minute,
                provider:   "groq",
                healthy:    true,
            },
            "bedrock": {
                limiter:    rate.NewLimiter(rate.Every(60*time.Second/5000), 5000), // 5k/min
                capacity:   5000,
                refillRate: time.Minute,
                provider:   "bedrock",
                healthy:    true,
            },
        },
        fallback: &FallbackChain{
            Order: []string{"groq", "openai", "anthropic", "bedrock"},
        },
    }
}

func (pqm *ProviderQuotaManager) AcquireTokens(
    provider string, 
    tokens int,
) (*ProviderToken, error) {
    pqm.mu.RLock()
    bucket, exists := pqm.buckets[provider]
    pqm.mu.RUnlock()
    
    if !exists || !bucket.healthy {
        // Provider unavailable, try fallback
        return pqm.tryFallback(tokens)
    }
    
    // Try to acquire tokens with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()
    
    if bucket.limiter.AllowN(time.Now(), tokens) {
        return &ProviderToken{
            Provider: provider,
            Tokens:   tokens,
            ExpireAt: time.Now().Add(5 * time.Minute),
        }, nil
    }
    
    // Check if we should wait or failover
    reservation := bucket.limiter.ReserveN(time.Now(), tokens)
    delay := reservation.Delay()
    
    if delay > 5*time.Second {
        // Too long to wait, try fallback
        reservation.Cancel()
        return pqm.tryFallback(tokens)
    }
    
    // Short wait is acceptable
    select {
    case <-time.After(delay):
        return &ProviderToken{
            Provider: provider,
            Tokens:   tokens,
            ExpireAt: time.Now().Add(5 * time.Minute),
        }, nil
    case <-ctx.Done():
        reservation.Cancel()
        return pqm.tryFallback(tokens)
    }
}

func (pqm *ProviderQuotaManager) tryFallback(tokens int) (*ProviderToken, error) {
    for _, fallbackProvider := range pqm.fallback.Order {
        bucket := pqm.buckets[fallbackProvider]
        if bucket.healthy && bucket.limiter.AllowN(time.Now(), tokens) {
            pqm.recordFallback(fallbackProvider)
            return &ProviderToken{
                Provider: fallbackProvider,
                Tokens:   tokens,
                Fallback: true,
            }, nil
        }
    }
    
    return nil, ErrNoProvidersAvailable
}

// Prevent thundering herd with jittered backoff
func (pqm *ProviderQuotaManager) HandleProviderError(provider string, err error) {
    if isRateLimitError(err) {
        pqm.mu.Lock()
        defer pqm.mu.Unlock()
        
        bucket := pqm.buckets[provider]
        bucket.healthy = false
        
        // Exponential backoff with jitter
        backoff := time.Duration(rand.Int63n(int64(30*time.Second))) + 30*time.Second
        
        time.AfterFunc(backoff, func() {
            pqm.mu.Lock()
            bucket.healthy = true
            pqm.mu.Unlock()
        })
    }
}
```

---

## 2. 🥶 Cold Start Risk Management

### The Footgun
Preview environments taking too long to start, causing timeouts and poor user experience.

### Solution: Build Cache + Image Layers + Warm Pool
```yaml
# k8s/knative-preview.yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: preview-template
spec:
  template:
    metadata:
      annotations:
        # Keep minimum instances warm
        autoscaling.knative.dev/min-scale: "2"
        # Scale to zero after 5 minutes
        autoscaling.knative.dev/scale-to-zero-grace-period: "5m"
        # Use aggressive scaling
        autoscaling.knative.dev/target: "10"
    spec:
      containerConcurrency: 100
      containers:
      - image: quantumlayer/preview-base:cached
        env:
        - name: ENABLE_BUILD_CACHE
          value: "true"
        - name: CACHE_STRATEGY
          value: "aggressive"
        volumeMounts:
        - name: build-cache
          mountPath: /cache
      volumes:
      - name: build-cache
        persistentVolumeClaim:
          claimName: build-cache-pvc
```

```dockerfile
# Dockerfile.preview-base
# Multi-stage build with cache mount
FROM golang:1.22-alpine AS go-deps
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

FROM node:20-alpine AS node-deps  
RUN --mount=type=cache,target=/root/.npm \
    npm ci --only=production

FROM python:3.11-slim AS python-deps
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install -r requirements.txt

# Final minimal base image with all deps
FROM alpine:3.19
COPY --from=go-deps /go/pkg /go/pkg
COPY --from=node-deps /node_modules /node_modules
COPY --from=python-deps /usr/local/lib/python3.11 /usr/local/lib/python3.11

# Pre-warm common operations
RUN echo "Warming image..." && \
    find /usr -type f -name "*.so" -exec dd if={} of=/dev/null bs=1M 2>/dev/null \; && \
    find /node_modules -type f -name "*.js" -exec head -n 1 {} \; > /dev/null

ENTRYPOINT ["/init"]
```

```go
// preview/warmer.go
type PreviewWarmer struct {
    pool     *WarmPool
    builders map[string]*CachedBuilder
}

func (pw *PreviewWarmer) Initialize() {
    // Pre-build common base images
    commonStacks := []string{"node", "python", "go", "java"}
    
    for _, stack := range commonStacks {
        go pw.warmStack(stack)
    }
    
    // Maintain warm pool
    go pw.maintainWarmPool()
}

func (pw *PreviewWarmer) maintainWarmPool() {
    ticker := time.NewTicker(30 * time.Second)
    
    for range ticker.C {
        current := pw.pool.Count()
        target := pw.calculateTargetPool()
        
        if current < target {
            pw.scaleUp(target - current)
        }
    }
}

func (pw *PreviewWarmer) AcquireWarmInstance(stack string) (*Instance, error) {
    // Try to get from warm pool first
    if instance := pw.pool.Get(stack); instance != nil {
        go pw.replenishPool(stack) // Async replenish
        return instance, nil
    }
    
    // Fall back to cold start with cache
    return pw.coldStartWithCache(stack)
}
```

---

## 3. 🧠 Chain-of-Thought (CoT) Leakage Prevention

### The Footgun
Storing raw CoT reasoning exposes internal logic and potentially sensitive data.

### Solution: Store Decisions, Not Reasoning
```go
// reasoning/cot_handler.go
type CoTHandler struct {
    decisions *DecisionStore
    sanitizer *ReasoningSanitizer
}

type ReasoningResult struct {
    // Never persist these
    RawThoughts      string `json:"-" db:"-"`
    InternalReasoning string `json:"-" db:"-"`
    
    // Only persist these
    Decision         string    `json:"decision"`
    PlanSummary      string    `json:"plan_summary"`
    QualityScore     float64   `json:"quality_score"`
    SelectedApproach string    `json:"selected_approach"`
    Diffs           []CodeDiff `json:"diffs"`
}

func (ch *CoTHandler) ProcessCoT(reasoning string) (*ReasoningResult, error) {
    // Extract decisions from reasoning
    decisions := ch.extractDecisions(reasoning)
    
    // Generate safe summary
    summary := ch.generateSummary(decisions)
    
    // Calculate scores without exposing reasoning
    scores := ch.calculateScores(reasoning)
    
    result := &ReasoningResult{
        RawThoughts:      reasoning, // Memory only, never persisted
        Decision:         decisions.Final,
        PlanSummary:      summary,
        QualityScore:     scores.Quality,
        SelectedApproach: decisions.Approach,
        Diffs:           ch.generateDiffs(decisions),
    }
    
    // Store only safe data
    ch.decisions.Store(result.ToSafeRecord())
    
    // Clear sensitive data before returning
    result.RawThoughts = ""
    result.InternalReasoning = ""
    
    return result, nil
}

// Ensure CoT never leaks to logs
func (ch *CoTHandler) LogSafely(ctx context.Context, result *ReasoningResult) {
    log.WithContext(ctx).Info("CoT processed",
        "decision", result.Decision,
        "score", result.QualityScore,
        "approach", result.SelectedApproach,
        // Never log RawThoughts or InternalReasoning
    )
}
```

---

## 4. 🔐 Secret Management

### The Footgun
Long-lived secrets in environment variables leading to exposure and rotation nightmares.

### Solution: Short-lived, Scoped Secrets
```go
// secrets/manager.go
type SecretManager struct {
    vault    *vault.Client
    scanner  *SecretScanner
    rotation *RotationScheduler
}

type ScopedSecret struct {
    Value     string
    Scope     string // "preview", "build", "runtime"
    TTL       time.Duration
    ExpiresAt time.Time
    Leased    bool
}

func (sm *SecretManager) GetScopedSecret(
    ctx context.Context,
    name string,
    scope SecretScope,
) (*ScopedSecret, error) {
    // Generate short-lived secret
    secret := &ScopedSecret{
        Scope: scope.String(),
        TTL:   sm.getTTLForScope(scope),
    }
    
    switch scope {
    case PreviewScope:
        // Preview secrets expire in 1 hour
        secret.TTL = 1 * time.Hour
        secret.Value = sm.generateEphemeralToken(name, secret.TTL)
        
    case BuildScope:
        // Build secrets expire after build (max 30 min)
        secret.TTL = 30 * time.Minute
        secret.Value = sm.vault.GetWithLease(name, secret.TTL)
        secret.Leased = true
        
    case RuntimeScope:
        // Runtime secrets rotate every 24 hours
        secret.TTL = 24 * time.Hour
        secret.Value = sm.vault.GetRotating(name)
    }
    
    secret.ExpiresAt = time.Now().Add(secret.TTL)
    
    // Schedule cleanup
    sm.scheduleCleanup(secret)
    
    return secret, nil
}

// Automatic secret scanning in repos
func (sm *SecretManager) ScanRepository(repo string) error {
    patterns := []string{
        `(?i)(api[_-]?key|apikey).*[:=]\s*['"]?([A-Za-z0-9+/]{20,})`,
        `(?i)(secret|token|password|passwd|pwd).*[:=]\s*['"]?([A-Za-z0-9+/]{20,})`,
        `(?i)aws.*[:=]\s*['"]?([A-Za-z0-9+/]{20,})`,
        `(?i)github.*[:=]\s*['"]?([A-Za-z0-9+/]{20,})`,
    }
    
    scanner := sm.scanner.WithPatterns(patterns)
    
    findings, err := scanner.ScanRepo(repo)
    if err != nil {
        return err
    }
    
    if len(findings) > 0 {
        // Block deployment
        return fmt.Errorf("secrets detected in repository: %d findings", len(findings))
    }
    
    return nil
}

// Preview environment secret injection
func (sm *SecretManager) InjectPreviewSecrets(
    preview *PreviewEnvironment,
) error {
    // Only inject short-lived, scoped secrets
    secrets := map[string]*ScopedSecret{
        "DATABASE_URL": sm.generatePreviewDatabase(preview.ID, 1*time.Hour),
        "API_TOKEN":    sm.generatePreviewToken(preview.ID, 1*time.Hour),
        "STORAGE_KEY":  sm.generateScopedStorage(preview.ID, 1*time.Hour),
    }
    
    for name, secret := range secrets {
        preview.Env[name] = secret.Value
        
        // Track for automatic revocation
        sm.trackSecret(preview.ID, secret)
    }
    
    return nil
}
```

---

## 5. 🧪 Test Flakiness Management

### The Footgun
Flaky E2E tests blocking deployments and causing developer frustration.

### Solution: Non-blocking E2E, Strict Unit/API
```yaml
# .github/workflows/test-strategy.yaml
name: Test Strategy

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - name: Run Unit Tests
        run: |
          go test ./... -race -cover
      - name: Check Coverage
        run: |
          if [ $(go test -cover | grep -o '[0-9]*\.[0-9]*%' | sed 's/%//' | awk '{if ($1 < 80) exit 1}') ]; then
            echo "Coverage below 80%, blocking deployment"
            exit 1
          fi

  api-tests:
    runs-on: ubuntu-latest
    steps:
      - name: Run API Tests
        run: |
          npm run test:api
      - name: Validate Contracts
        run: |
          npm run test:contracts
    # These MUST pass for deployment
    
  e2e-tests:
    runs-on: ubuntu-latest
    continue-on-error: true  # Non-blocking for V1
    steps:
      - name: Run E2E Tests
        id: e2e
        run: |
          npm run test:e2e || echo "::warning::E2E tests failed but not blocking"
      - name: Report Flaky Tests
        if: failure()
        run: |
          npm run report:flaky-tests
```

```go
// testing/flaky_detector.go
type FlakyTestDetector struct {
    history  *TestHistory
    reporter *FlakyReporter
}

func (ftd *FlakyTestDetector) AnalyzeTest(test TestResult) {
    // Track test history
    ftd.history.Record(test)
    
    // Calculate flakiness score
    flakiness := ftd.calculateFlakiness(test.Name)
    
    if flakiness > 0.1 { // More than 10% failure rate
        ftd.markAsFlaky(test.Name)
        
        // Auto-quarantine flaky E2E tests
        if test.Type == "e2e" {
            ftd.quarantine(test.Name)
        }
    }
}

func (ftd *FlakyTestDetector) EnforcePolicy(suite TestSuite) error {
    // Unit tests must be stable
    if suite.Type == "unit" && suite.FailureRate > 0.01 {
        return fmt.Errorf("unit test stability below threshold: %.2f%%", 
            suite.FailureRate*100)
    }
    
    // API tests must be stable
    if suite.Type == "api" && suite.FailureRate > 0.02 {
        return fmt.Errorf("API test stability below threshold: %.2f%%", 
            suite.FailureRate*100)
    }
    
    // E2E tests are informational only for V1
    if suite.Type == "e2e" && suite.FailureRate > 0.1 {
        log.Warn("E2E tests are flaky but non-blocking",
            "failure_rate", suite.FailureRate,
            "quarantined_tests", ftd.getQuarantinedTests())
    }
    
    return nil
}
```

---

## 6. 📊 Data Model (Minimal V1)

### Core Entities
```sql
-- Organization with billing
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    plan TEXT NOT NULL CHECK (plan IN ('free', 'pro', 'enterprise')),
    billing_email TEXT,
    stripe_customer_id TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Projects belong to organizations
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID REFERENCES organizations(id),
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, name)
);

-- Environments with TTL for previews
CREATE TABLE environments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id),
    name TEXT NOT NULL,
    type TEXT CHECK (type IN ('dev', 'staging', 'prod', 'preview')),
    ttl INTERVAL, -- NULL for permanent, set for previews
    expires_at TIMESTAMPTZ, -- Calculated from created_at + ttl
    config JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(project_id, name)
);

-- Runs track all operations
CREATE TABLE runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id),
    environment_id UUID REFERENCES environments(id),
    type TEXT CHECK (type IN ('parse', 'plan', 'generate', 'test', 'package', 'deploy')),
    status TEXT CHECK (status IN ('pending', 'running', 'success', 'failed', 'cancelled')),
    
    -- Metrics
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER,
    tokens_used INTEGER,
    cost_cents INTEGER,
    
    -- Tracing
    trace_id TEXT NOT NULL,
    parent_run_id UUID REFERENCES runs(id),
    
    -- Metadata
    input JSONB,
    output JSONB,
    error TEXT,
    metrics JSONB, -- quality_score, test_coverage, etc.
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Artifacts produced by runs
CREATE TABLE artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID REFERENCES runs(id),
    type TEXT CHECK (type IN ('repo', 'image', 'capsule', 'sbom')),
    name TEXT NOT NULL,
    uri TEXT NOT NULL, -- s3://..., docker://..., git://...
    size_bytes BIGINT,
    checksum TEXT,
    metadata JSONB, -- SBOM data, signatures, etc.
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Policies for routing and limits
CREATE TABLE policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope_type TEXT CHECK (scope_type IN ('org', 'project')),
    scope_id UUID NOT NULL,
    
    -- Policy configuration
    routing JSONB, -- provider preferences, fallback chains
    cost_caps JSONB, -- daily, monthly limits
    safety_levels JSONB, -- HAP thresholds, moderation settings
    rate_limits JSONB,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(scope_type, scope_id)
);

-- Immutable audit log
CREATE TABLE audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    
    -- Who
    actor_type TEXT NOT NULL, -- 'user', 'service', 'system'
    actor_id TEXT NOT NULL,
    
    -- What
    action TEXT NOT NULL,
    resource_type TEXT,
    resource_id TEXT,
    
    -- Where
    ip_address INET,
    user_agent TEXT,
    
    -- Result
    result TEXT CHECK (result IN ('success', 'failure', 'error')),
    error_message TEXT,
    
    -- Immutability
    previous_hash TEXT,
    hash TEXT NOT NULL,
    
    -- Metadata
    metadata JSONB
);

-- Prevent updates to audit log
CREATE TRIGGER audit_events_immutable
BEFORE UPDATE ON audit_events
FOR EACH ROW EXECUTE FUNCTION raise_exception('Audit events are immutable');

-- Indexes for performance
CREATE INDEX idx_runs_project_status ON runs(project_id, status);
CREATE INDEX idx_runs_trace ON runs(trace_id);
CREATE INDEX idx_artifacts_run ON artifacts(run_id);
CREATE INDEX idx_audit_timestamp ON audit_events(timestamp);
CREATE INDEX idx_environments_ttl ON environments(expires_at) WHERE expires_at IS NOT NULL;
```

---

## 7. 🔒 Security & Compliance Baseline

### SOC2-Friendly Implementation
```go
// security/compliance.go
type ComplianceManager struct {
    sso      *SSOProvider
    mfa      *MFAEnforcer
    crypto   *CryptoManager
    audit    *AuditLogger
    sbom     *SBOMGenerator
}

// SSO + MFA
func (cm *ComplianceManager) AuthenticateUser(
    ctx context.Context,
    credentials Credentials,
) (*User, error) {
    // SSO via OAuth/SAML
    user, err := cm.sso.Authenticate(credentials)
    if err != nil {
        return nil, err
    }
    
    // Enforce MFA for all users
    if !cm.mfa.Verify(user, credentials.MFAToken) {
        cm.audit.LogFailedAuth(user, "MFA_FAILED")
        return nil, ErrMFARequired
    }
    
    // Generate scoped API token
    token := cm.generateScopedToken(user, 24*time.Hour)
    
    cm.audit.LogSuccessfulAuth(user)
    return user, nil
}

// Encryption standards
func (cm *ComplianceManager) EncryptData(data []byte) ([]byte, error) {
    // AES-256-GCM for data at rest
    key := cm.crypto.GetDataKey()
    
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }
    
    return gcm.Seal(nonce, nonce, data, nil), nil
}

// SBOM generation for every capsule
func (cm *ComplianceManager) GenerateSBOM(
    capsule *QuantumCapsule,
) (*SBOM, error) {
    sbom := &SBOM{
        Format:    "CycloneDX",
        Version:   "1.5",
        Timestamp: time.Now(),
        Components: []Component{},
    }
    
    // Scan all dependencies
    for _, dep := range capsule.Dependencies {
        component := Component{
            Name:    dep.Name,
            Version: dep.Version,
            Hashes:  cm.calculateHashes(dep),
            Licenses: cm.detectLicenses(dep),
            CVEs:    cm.scanCVEs(dep),
        }
        sbom.Components = append(sbom.Components, component)
    }
    
    // Sign SBOM
    sbom.Signature = cm.crypto.SignSBOM(sbom)
    
    return sbom, nil
}

// Immutable audit log with hash chain
func (cm *ComplianceManager) LogAuditEvent(event AuditEvent) error {
    // Get previous hash
    previousHash := cm.audit.GetLastHash()
    
    // Create immutable record
    record := AuditRecord{
        ID:           uuid.New(),
        Timestamp:    time.Now(),
        Actor:        event.Actor,
        Action:       event.Action,
        Resource:     event.Resource,
        Result:       event.Result,
        PreviousHash: previousHash,
    }
    
    // Calculate hash including previous hash
    record.Hash = cm.calculateHash(record)
    
    // Store immutably
    return cm.audit.Store(record)
}

// K8s least privilege
func (cm *ComplianceManager) CreateServiceAccount(
    namespace string,
    service string,
) (*v1.ServiceAccount, error) {
    sa := &v1.ServiceAccount{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("%s-sa", service),
            Namespace: namespace,
        },
    }
    
    // Minimal required permissions only
    role := &rbacv1.Role{
        Rules: []rbacv1.PolicyRule{
            {
                APIGroups: []string{""},
                Resources: []string{"configmaps"},
                Verbs:     []string{"get", "list"},
            },
        },
    }
    
    // No hostPath volumes allowed
    psp := &policy.PodSecurityPolicy{
        Spec: policy.PodSecurityPolicySpec{
            Volumes: []policy.FSType{
                policy.ConfigMap,
                policy.Secret,
                policy.EmptyDir,
                policy.PersistentVolumeClaim,
                // hostPath explicitly excluded
            },
            RunAsUser: policy.RunAsUserStrategyOptions{
                Rule: policy.MustRunAsNonRoot,
            },
        },
    }
    
    return sa, nil
}
```

### Data Retention Policy
```yaml
# config/retention.yaml
retention_policies:
  logs:
    hot:
      duration: 30d
      storage: elasticsearch
      redaction: false
      
    warm:
      duration: 365d
      storage: s3
      redaction: true  # PII removed
      compression: gzip
      
  previews:
    auto_gc:
      after: 7d
      action: delete_all
      
  audit:
    duration: 7y  # Legal requirement
    storage: immutable_store
    encryption: required
    
  metrics:
    raw:
      duration: 90d
      
    aggregated:
      duration: 2y
```

---

## 8. 📡 Observability Wiring

### Comprehensive Tracing
```go
// observability/tracing.go
type TraceManager struct {
    tracer trace.Tracer
}

func (tm *TraceManager) TraceRun(
    ctx context.Context,
    runType string,
) (context.Context, trace.Span) {
    // One trace ID from API to deployment
    ctx, span := tm.tracer.Start(ctx, fmt.Sprintf("run.%s", runType),
        trace.WithAttributes(
            attribute.String("run.type", runType),
            attribute.String("org.id", getOrgID(ctx)),
            attribute.String("project.id", getProjectID(ctx)),
            attribute.String("run.id", getRunID(ctx)),
        ),
    )
    
    // Propagate trace through all calls
    return ctx, span
}

// Link every operation to trace
func (tm *TraceManager) TraceProviderCall(
    ctx context.Context,
    provider string,
) (context.Context, trace.Span) {
    ctx, span := tm.tracer.Start(ctx, "provider.call",
        trace.WithAttributes(
            attribute.String("provider", provider),
            attribute.String("trace.id", trace.SpanFromContext(ctx).SpanContext().TraceID().String()),
        ),
    )
    
    return ctx, span
}
```

### Key Metrics Per Run
```go
// observability/metrics.go
type RunMetrics struct {
    // Latency histograms
    LatencyHistogram   *prometheus.HistogramVec
    
    // Resource usage
    TokensUsed         *prometheus.CounterVec
    CostDollars        *prometheus.CounterVec
    
    // Quality metrics
    QualityScore       *prometheus.GaugeVec
    TestCoverage       *prometheus.GaugeVec
    
    // Cache performance
    CacheHitRate       *prometheus.GaugeVec
    
    // Reliability
    FallbackCount      *prometheus.CounterVec
    SafetyInterventions *prometheus.CounterVec
    
    // Preview metrics
    PreviewDeployTime  *prometheus.HistogramVec
}

func (rm *RunMetrics) RecordRun(run *Run) {
    labels := prometheus.Labels{
        "org":     run.OrgID,
        "project": run.ProjectID,
        "type":    run.Type,
        "status":  run.Status,
    }
    
    rm.LatencyHistogram.With(labels).Observe(run.Duration.Seconds())
    rm.TokensUsed.With(labels).Add(float64(run.TokensUsed))
    rm.CostDollars.With(labels).Add(run.Cost)
    rm.QualityScore.With(labels).Set(run.QualityScore)
    rm.TestCoverage.With(labels).Set(run.TestCoverage)
    rm.CacheHitRate.With(labels).Set(run.CacheHitRate)
    
    if run.FallbackUsed {
        rm.FallbackCount.With(labels).Inc()
    }
    
    if run.SafetyTriggered {
        rm.SafetyInterventions.With(labels).Inc()
    }
}
```

### Structured Logging
```go
// observability/logging.go
type StructuredLogger struct {
    logger *zap.Logger
}

func (sl *StructuredLogger) LogRun(ctx context.Context, msg string, run *Run) {
    // Always include trace_id, org, project, run_id
    sl.logger.Info(msg,
        zap.String("trace_id", getTraceID(ctx)),
        zap.String("org_id", run.OrgID),
        zap.String("project_id", run.ProjectID),
        zap.String("run_id", run.ID),
        zap.String("type", run.Type),
        zap.String("status", run.Status),
        zap.Duration("duration", run.Duration),
        zap.Int("tokens", run.TokensUsed),
        zap.Float64("cost", run.Cost),
        // Redact sensitive fields
        zap.String("prompt", redact(run.Prompt)),
    )
}

func redact(s string) string {
    // Redact PII, secrets, etc.
    patterns := []string{
        `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, // emails
        `\b\d{3}-\d{2}-\d{4}\b`, // SSN
        `(?i)(api[_-]?key|token|secret|password)[:=]\S+`, // secrets
    }
    
    result := s
    for _, pattern := range patterns {
        re := regexp.MustCompile(pattern)
        result = re.ReplaceAllString(result, "[REDACTED]")
    }
    
    return result
}
```

---

## 🎯 Implementation Checklist

### Week 1: Foundation
- [ ] Implement token bucket for provider quotas
- [ ] Set up Knative with warm pools
- [ ] Create minimal data model
- [ ] Configure structured logging with trace IDs

### Week 2: Security
- [ ] Implement short-lived secret system
- [ ] Set up secret scanning in CI
- [ ] Configure audit log with hash chain
- [ ] Deploy least-privilege K8s configs

### Week 3: Observability
- [ ] Wire OpenTelemetry tracing
- [ ] Set up Prometheus metrics
- [ ] Configure log aggregation with redaction
- [ ] Create observability dashboards

### Week 4: Testing & Reliability
- [ ] Implement test flakiness detection
- [ ] Set up non-blocking E2E tests
- [ ] Configure SBOM generation
- [ ] Deploy preview auto-GC

---

*Architecture Footguns Document v1.0*  
*Last Updated: Current Session*  
*Critical for: Day 1 Production*