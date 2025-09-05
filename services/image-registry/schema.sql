-- Golden Images Database Schema
-- Database: image_registry

CREATE DATABASE IF NOT EXISTS image_registry;

\c image_registry;

-- Golden Images table
CREATE TABLE IF NOT EXISTS golden_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    base_os VARCHAR(100) NOT NULL,
    platform VARCHAR(50) NOT NULL,
    packages TEXT[],
    hardening VARCHAR(50),
    compliance TEXT[],
    registry_url TEXT,
    digest VARCHAR(255),
    size BIGINT,
    build_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_scanned TIMESTAMP,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Vulnerabilities table
CREATE TABLE IF NOT EXISTS vulnerabilities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID REFERENCES golden_images(id) ON DELETE CASCADE,
    cve VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    description TEXT,
    fix_version VARCHAR(50),
    discovered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Attestations table
CREATE TABLE IF NOT EXISTS attestations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID REFERENCES golden_images(id) ON DELETE CASCADE,
    signature TEXT NOT NULL,
    signed_by VARCHAR(255) NOT NULL,
    signed_at TIMESTAMP NOT NULL,
    verified BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMP
);

-- SBOM (Software Bill of Materials) table
CREATE TABLE IF NOT EXISTS sbom (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID REFERENCES golden_images(id) ON DELETE CASCADE,
    component_name VARCHAR(255) NOT NULL,
    component_version VARCHAR(100),
    component_type VARCHAR(50),
    license VARCHAR(100),
    metadata JSONB
);

-- Patch Status table
CREATE TABLE IF NOT EXISTS patch_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID REFERENCES golden_images(id) ON DELETE CASCADE,
    current_version VARCHAR(50) NOT NULL,
    latest_version VARCHAR(50),
    patches_needed INTEGER DEFAULT 0,
    cves_fixed TEXT[],
    last_checked TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    up_to_date BOOLEAN DEFAULT TRUE
);

-- Drift Detection table
CREATE TABLE IF NOT EXISTS drift_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform VARCHAR(50),
    datacenter VARCHAR(100),
    environment VARCHAR(50),
    total_nodes INTEGER,
    drifted_nodes INTEGER,
    details JSONB,
    reported_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Build History table
CREATE TABLE IF NOT EXISTS build_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID REFERENCES golden_images(id) ON DELETE CASCADE,
    build_status VARCHAR(50) NOT NULL,
    build_log TEXT,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    triggered_by VARCHAR(255),
    packer_template VARCHAR(100)
);

-- Indexes for performance
CREATE INDEX idx_golden_images_name ON golden_images(name);
CREATE INDEX idx_golden_images_platform ON golden_images(platform);
CREATE INDEX idx_golden_images_compliance ON golden_images USING GIN(compliance);
CREATE INDEX idx_vulnerabilities_image_id ON vulnerabilities(image_id);
CREATE INDEX idx_vulnerabilities_severity ON vulnerabilities(severity);
CREATE INDEX idx_attestations_image_id ON attestations(image_id);
CREATE INDEX idx_sbom_image_id ON sbom(image_id);
CREATE INDEX idx_patch_status_image_id ON patch_status(image_id);
CREATE INDEX idx_drift_reports_platform ON drift_reports(platform);
CREATE INDEX idx_build_history_image_id ON build_history(image_id);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_golden_images_updated_at BEFORE UPDATE
    ON golden_images FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();