package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

type Database struct {
	conn *sql.DB
}

func NewDatabase() (*Database, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Use Temporal's PostgreSQL instance with a new database
		dbURL = "postgres://postgres:postgres@postgres-postgresql.temporal.svc.cluster.local:5432/image_registry?sslmode=disable"
	}

	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &Database{conn: conn}
	
	// Initialize schema
	if err := db.initSchema(); err != nil {
		log.Printf("Warning: Could not initialize schema: %v", err)
	}

	return db, nil
}

func (db *Database) initSchema() error {
	// Create database if not exists (simplified version)
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS golden_images (
			id VARCHAR(36) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			version VARCHAR(50) NOT NULL,
			base_os VARCHAR(100) NOT NULL,
			platform VARCHAR(50) NOT NULL,
			packages TEXT,
			hardening VARCHAR(50),
			compliance TEXT,
			registry_url TEXT,
			digest VARCHAR(255),
			size BIGINT,
			build_time TIMESTAMP,
			last_scanned TIMESTAMP,
			metadata TEXT,
			sbom TEXT,
			vulnerabilities TEXT,
			attestation TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create golden_images table: %w", err)
	}

	return nil
}

func (db *Database) SaveImage(image *GoldenImage) error {
	packagesJSON, _ := json.Marshal(image.Packages)
	complianceJSON, _ := json.Marshal(image.Compliance)
	metadataJSON, _ := json.Marshal(image.Metadata)
	sbomJSON, _ := json.Marshal(image.SBOM)
	vulnerabilitiesJSON, _ := json.Marshal(image.Vulnerabilities)
	attestationJSON, _ := json.Marshal(image.Attestation)

	query := `
		INSERT INTO golden_images (
			id, name, version, base_os, platform, packages, hardening, 
			compliance, registry_url, digest, size, build_time, last_scanned,
			metadata, sbom, vulnerabilities, attestation
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			version = EXCLUDED.version,
			base_os = EXCLUDED.base_os,
			platform = EXCLUDED.platform,
			packages = EXCLUDED.packages,
			hardening = EXCLUDED.hardening,
			compliance = EXCLUDED.compliance,
			registry_url = EXCLUDED.registry_url,
			digest = EXCLUDED.digest,
			size = EXCLUDED.size,
			build_time = EXCLUDED.build_time,
			last_scanned = EXCLUDED.last_scanned,
			metadata = EXCLUDED.metadata,
			sbom = EXCLUDED.sbom,
			vulnerabilities = EXCLUDED.vulnerabilities,
			attestation = EXCLUDED.attestation,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := db.conn.Exec(query,
		image.ID, image.Name, image.Version, image.BaseOS, image.Platform,
		string(packagesJSON), image.Hardening, string(complianceJSON),
		image.RegistryURL, image.Digest, image.Size, image.BuildTime,
		image.LastScanned, string(metadataJSON), string(sbomJSON),
		string(vulnerabilitiesJSON), string(attestationJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}

	return nil
}

func (db *Database) GetImage(id string) (*GoldenImage, error) {
	query := `
		SELECT id, name, version, base_os, platform, packages, hardening,
		       compliance, registry_url, digest, size, build_time, last_scanned,
		       metadata, sbom, vulnerabilities, attestation
		FROM golden_images
		WHERE id = $1
	`

	var image GoldenImage
	var packagesJSON, complianceJSON, metadataJSON, sbomJSON, vulnerabilitiesJSON, attestationJSON sql.NullString
	var buildTime, lastScanned sql.NullTime
	var size sql.NullInt64

	err := db.conn.QueryRow(query, id).Scan(
		&image.ID, &image.Name, &image.Version, &image.BaseOS, &image.Platform,
		&packagesJSON, &image.Hardening, &complianceJSON,
		&image.RegistryURL, &image.Digest, &size, &buildTime, &lastScanned,
		&metadataJSON, &sbomJSON, &vulnerabilitiesJSON, &attestationJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}

	// Parse JSON fields
	if packagesJSON.Valid {
		json.Unmarshal([]byte(packagesJSON.String), &image.Packages)
	}
	if complianceJSON.Valid {
		json.Unmarshal([]byte(complianceJSON.String), &image.Compliance)
	}
	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &image.Metadata)
	}
	if sbomJSON.Valid {
		json.Unmarshal([]byte(sbomJSON.String), &image.SBOM)
	}
	if vulnerabilitiesJSON.Valid {
		json.Unmarshal([]byte(vulnerabilitiesJSON.String), &image.Vulnerabilities)
	}
	if attestationJSON.Valid {
		json.Unmarshal([]byte(attestationJSON.String), &image.Attestation)
	}

	if buildTime.Valid {
		image.BuildTime = buildTime.Time
	}
	if lastScanned.Valid {
		image.LastScanned = lastScanned.Time
	}
	if size.Valid {
		image.Size = size.Int64
	}

	return &image, nil
}

func (db *Database) ListImages() ([]*GoldenImage, error) {
	query := `
		SELECT id, name, version, base_os, platform, packages, hardening,
		       compliance, registry_url, digest, size, build_time, last_scanned,
		       metadata, sbom, vulnerabilities, attestation
		FROM golden_images
		ORDER BY created_at DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}
	defer rows.Close()

	var images []*GoldenImage
	for rows.Next() {
		var image GoldenImage
		var packagesJSON, complianceJSON, metadataJSON, sbomJSON, vulnerabilitiesJSON, attestationJSON sql.NullString
		var buildTime, lastScanned sql.NullTime
		var size sql.NullInt64

		err := rows.Scan(
			&image.ID, &image.Name, &image.Version, &image.BaseOS, &image.Platform,
			&packagesJSON, &image.Hardening, &complianceJSON,
			&image.RegistryURL, &image.Digest, &size, &buildTime, &lastScanned,
			&metadataJSON, &sbomJSON, &vulnerabilitiesJSON, &attestationJSON,
		)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		// Parse JSON fields
		if packagesJSON.Valid {
			json.Unmarshal([]byte(packagesJSON.String), &image.Packages)
		}
		if complianceJSON.Valid {
			json.Unmarshal([]byte(complianceJSON.String), &image.Compliance)
		}
		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &image.Metadata)
		}
		if sbomJSON.Valid {
			json.Unmarshal([]byte(sbomJSON.String), &image.SBOM)
		}
		if vulnerabilitiesJSON.Valid {
			json.Unmarshal([]byte(vulnerabilitiesJSON.String), &image.Vulnerabilities)
		}
		if attestationJSON.Valid {
			json.Unmarshal([]byte(attestationJSON.String), &image.Attestation)
		}

		if buildTime.Valid {
			image.BuildTime = buildTime.Time
		}
		if lastScanned.Valid {
			image.LastScanned = lastScanned.Time
		}
		if size.Valid {
			image.Size = size.Int64
		}

		images = append(images, &image)
	}

	return images, nil
}

func (db *Database) DeleteImage(id string) error {
	query := `DELETE FROM golden_images WHERE id = $1`
	_, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}
	return nil
}

func (db *Database) GetImagesByPlatform(platform string) ([]*GoldenImage, error) {
	query := `
		SELECT id, name, version, base_os, platform, packages, hardening,
		       compliance, registry_url, digest, size, build_time, last_scanned,
		       metadata, sbom, vulnerabilities, attestation
		FROM golden_images
		WHERE platform = $1
		ORDER BY created_at DESC
	`

	rows, err := db.conn.Query(query, platform)
	if err != nil {
		return nil, fmt.Errorf("failed to get images by platform: %w", err)
	}
	defer rows.Close()

	var images []*GoldenImage
	for rows.Next() {
		var image GoldenImage
		var packagesJSON, complianceJSON, metadataJSON, sbomJSON, vulnerabilitiesJSON, attestationJSON sql.NullString
		var buildTime, lastScanned sql.NullTime
		var size sql.NullInt64

		err := rows.Scan(
			&image.ID, &image.Name, &image.Version, &image.BaseOS, &image.Platform,
			&packagesJSON, &image.Hardening, &complianceJSON,
			&image.RegistryURL, &image.Digest, &size, &buildTime, &lastScanned,
			&metadataJSON, &sbomJSON, &vulnerabilitiesJSON, &attestationJSON,
		)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		// Parse JSON fields (same as above)
		if packagesJSON.Valid {
			json.Unmarshal([]byte(packagesJSON.String), &image.Packages)
		}
		if complianceJSON.Valid {
			json.Unmarshal([]byte(complianceJSON.String), &image.Compliance)
		}
		if buildTime.Valid {
			image.BuildTime = buildTime.Time
		}
		if lastScanned.Valid {
			image.LastScanned = lastScanned.Time
		}
		if size.Valid {
			image.Size = size.Int64
		}

		images = append(images, &image)
	}

	return images, nil
}

func (db *Database) Close() error {
	return db.conn.Close()
}