-- QuantumLayer V2 Database Schema
-- PostgreSQL 16

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- For text search
CREATE EXTENSION IF NOT EXISTS "btree_gist";  -- For exclusion constraints

-- Create enum types
CREATE TYPE user_role AS ENUM ('admin', 'user', 'viewer');
CREATE TYPE organization_plan AS ENUM ('free', 'pro', 'team', 'enterprise');
CREATE TYPE generation_status AS ENUM ('pending', 'parsing', 'planning', 'generating', 'validating', 'completed', 'failed', 'cancelled');
CREATE TYPE environment_type AS ENUM ('development', 'staging', 'production', 'preview');
CREATE TYPE run_type AS ENUM ('parse', 'plan', 'generate', 'test', 'package', 'deploy');
CREATE TYPE run_status AS ENUM ('pending', 'running', 'success', 'failed', 'cancelled');

-- Organizations table (top-level tenant)
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    plan organization_plan DEFAULT 'free',
    stripe_customer_id VARCHAR(255),
    stripe_subscription_id VARCHAR(255),
    
    -- Settings
    settings JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    
    -- Limits based on plan
    max_users INTEGER DEFAULT 5,
    max_projects INTEGER DEFAULT 3,
    max_generations_per_month INTEGER DEFAULT 100,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ -- Soft delete
);

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),
    role user_role DEFAULT 'user',
    
    -- Auth provider IDs
    clerk_id VARCHAR(255) UNIQUE,
    github_id VARCHAR(255),
    google_id VARCHAR(255),
    
    -- Profile
    avatar_url TEXT,
    bio TEXT,
    metadata JSONB DEFAULT '{}',
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    last_login_at TIMESTAMPTZ,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Projects table
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Configuration
    settings JSONB DEFAULT '{}',
    secrets JSONB DEFAULT '{}',  -- Encrypted
    metadata JSONB DEFAULT '{}',
    
    -- Repository
    repo_url TEXT,
    default_branch VARCHAR(255) DEFAULT 'main',
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    UNIQUE(org_id, slug)
);

-- Environments table
CREATE TABLE environments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type environment_type NOT NULL,
    
    -- Configuration
    config JSONB DEFAULT '{}',
    secrets JSONB DEFAULT '{}',  -- Encrypted
    variables JSONB DEFAULT '{}',
    
    -- Preview environments have TTL
    ttl INTERVAL,
    expires_at TIMESTAMPTZ,
    
    -- Deployment info
    url TEXT,
    is_active BOOLEAN DEFAULT true,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(project_id, name)
);

-- Generations table (main feature)
CREATE TABLE generations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    environment_id UUID REFERENCES environments(id) ON DELETE SET NULL,
    
    -- Request
    prompt TEXT NOT NULL,
    language VARCHAR(50),
    framework VARCHAR(100),
    options JSONB DEFAULT '{}',
    
    -- Status
    status generation_status DEFAULT 'pending',
    progress FLOAT DEFAULT 0.0,
    current_phase VARCHAR(100),
    
    -- Result
    code TEXT,
    files JSONB,  -- Array of generated files
    artifacts JSONB,  -- Links to stored artifacts
    
    -- Metrics
    quality_score FLOAT,
    test_coverage FLOAT,
    tokens_used INTEGER DEFAULT 0,
    cost_cents INTEGER DEFAULT 0,
    duration_ms INTEGER,
    
    -- Providers used
    providers_used JSONB DEFAULT '[]',
    fallback_count INTEGER DEFAULT 0,
    
    -- Error handling
    error_message TEXT,
    error_details JSONB,
    
    -- Tracing
    trace_id VARCHAR(255),
    span_id VARCHAR(255),
    
    -- Timestamps
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Runs table (track all operations)
CREATE TABLE runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    environment_id UUID REFERENCES environments(id) ON DELETE SET NULL,
    generation_id UUID REFERENCES generations(id) ON DELETE CASCADE,
    
    -- Run details
    type run_type NOT NULL,
    status run_status DEFAULT 'pending',
    
    -- Execution
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER,
    
    -- Resources
    cpu_used FLOAT,
    memory_used_mb INTEGER,
    tokens_used INTEGER,
    cost_cents INTEGER,
    
    -- Tracing
    trace_id VARCHAR(255) NOT NULL,
    parent_run_id UUID REFERENCES runs(id),
    
    -- Data
    input JSONB,
    output JSONB,
    error TEXT,
    metrics JSONB,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Artifacts table
CREATE TABLE artifacts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    run_id UUID REFERENCES runs(id) ON DELETE CASCADE,
    generation_id UUID REFERENCES generations(id) ON DELETE CASCADE,
    
    -- Artifact details
    type VARCHAR(50) NOT NULL, -- 'repository', 'image', 'capsule', 'sbom'
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50),
    
    -- Storage
    storage_url TEXT NOT NULL,  -- s3://, docker://, git://
    size_bytes BIGINT,
    checksum VARCHAR(255),
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    sbom JSONB,  -- Software Bill of Materials
    signatures JSONB,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

-- Agents table
CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    generation_id UUID REFERENCES generations(id) ON DELETE CASCADE,
    
    -- Agent details
    type VARCHAR(50) NOT NULL,  -- 'architect', 'developer', 'tester', etc.
    status VARCHAR(50) DEFAULT 'idle',
    progress FLOAT DEFAULT 0.0,
    
    -- Execution
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    tokens_used INTEGER DEFAULT 0,
    
    -- Communication
    messages JSONB DEFAULT '[]',
    decisions JSONB DEFAULT '{}',
    output TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Policies table
CREATE TABLE policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scope_type VARCHAR(50) NOT NULL,  -- 'organization', 'project'
    scope_id UUID NOT NULL,
    
    -- Policy configuration
    routing JSONB DEFAULT '{}',  -- LLM routing preferences
    cost_limits JSONB DEFAULT '{}',  -- Daily/monthly limits
    safety_config JSONB DEFAULT '{}',  -- HAP thresholds
    rate_limits JSONB DEFAULT '{}',
    quality_thresholds JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(scope_type, scope_id)
);

-- Audit events table (immutable)
CREATE TABLE audit_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    
    -- Actor
    actor_type VARCHAR(50) NOT NULL,  -- 'user', 'system', 'api'
    actor_id VARCHAR(255) NOT NULL,
    org_id UUID REFERENCES organizations(id),
    
    -- Action
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50),
    resource_id VARCHAR(255),
    
    -- Context
    ip_address INET,
    user_agent TEXT,
    
    -- Result
    result VARCHAR(50) NOT NULL,  -- 'success', 'failure', 'error'
    error_message TEXT,
    
    -- Audit trail
    previous_hash VARCHAR(255),
    hash VARCHAR(255) NOT NULL,
    
    -- Data
    metadata JSONB DEFAULT '{}'
);

-- API keys table
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,  -- SHA256 hash
    key_prefix VARCHAR(10) NOT NULL,  -- First 10 chars for identification
    
    -- Permissions
    scopes JSONB DEFAULT '[]',
    rate_limit INTEGER,
    
    -- Usage
    last_used_at TIMESTAMPTZ,
    usage_count INTEGER DEFAULT 0,
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    expires_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

-- Create indexes for performance
CREATE INDEX idx_organizations_slug ON organizations(slug);
CREATE INDEX idx_organizations_stripe ON organizations(stripe_customer_id);

CREATE INDEX idx_users_org ON users(org_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_clerk ON users(clerk_id);

CREATE INDEX idx_projects_org ON projects(org_id);
CREATE INDEX idx_projects_slug ON projects(org_id, slug);

CREATE INDEX idx_environments_project ON environments(project_id);
CREATE INDEX idx_environments_expires ON environments(expires_at) WHERE expires_at IS NOT NULL;

CREATE INDEX idx_generations_org ON generations(org_id);
CREATE INDEX idx_generations_project ON generations(project_id);
CREATE INDEX idx_generations_user ON generations(user_id);
CREATE INDEX idx_generations_status ON generations(status);
CREATE INDEX idx_generations_created ON generations(created_at DESC);

CREATE INDEX idx_runs_generation ON runs(generation_id);
CREATE INDEX idx_runs_trace ON runs(trace_id);
CREATE INDEX idx_runs_type_status ON runs(type, status);

CREATE INDEX idx_artifacts_generation ON artifacts(generation_id);
CREATE INDEX idx_artifacts_run ON artifacts(run_id);

CREATE INDEX idx_agents_generation ON agents(generation_id);
CREATE INDEX idx_agents_type ON agents(type);

CREATE INDEX idx_audit_timestamp ON audit_events(timestamp DESC);
CREATE INDEX idx_audit_org ON audit_events(org_id);
CREATE INDEX idx_audit_actor ON audit_events(actor_type, actor_id);

CREATE INDEX idx_api_keys_org ON api_keys(org_id);
CREATE INDEX idx_api_keys_prefix ON api_keys(key_prefix);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at trigger to relevant tables
CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_environments_updated_at BEFORE UPDATE ON environments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_policies_updated_at BEFORE UPDATE ON policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Prevent updates to audit_events (immutable)
CREATE OR REPLACE FUNCTION prevent_audit_update()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Audit events are immutable and cannot be updated';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_events_immutable
    BEFORE UPDATE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_update();

-- Sample data for development
INSERT INTO organizations (name, slug, plan) VALUES 
    ('Demo Organization', 'demo', 'free');

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO quantum;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO quantum;

-- Success message
DO $$ 
BEGIN 
    RAISE NOTICE 'QuantumLayer database initialized successfully!';
END $$;