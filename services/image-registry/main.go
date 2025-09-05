package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	DefaultPort         = "8096"
	DefaultRegistryURL  = "http://docker-registry.image-registry.svc.cluster.local:5000"
	DefaultDatabaseURL  = "postgres://postgres:postgres@quantum-drops-db.quantumlayer.svc.cluster.local/quantumdrops"
)

// GoldenImage represents a golden image with metadata
type GoldenImage struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Version        string                 `json:"version"`
	BaseOS         string                 `json:"base_os"`
	Platform       string                 `json:"platform"` // aws, azure, gcp, vmware, docker
	Packages       []string               `json:"packages"`
	Hardening      string                 `json:"hardening"` // CIS, STIG, custom
	Compliance     []string               `json:"compliance"` // SOC2, HIPAA, PCI-DSS
	RegistryURL    string                 `json:"registry_url"`
	Digest         string                 `json:"digest"`
	Size           int64                  `json:"size"`
	SBOM           map[string]interface{} `json:"sbom,omitempty"`
	Vulnerabilities []Vulnerability        `json:"vulnerabilities,omitempty"`
	Attestation    *Attestation           `json:"attestation,omitempty"`
	BuildTime      time.Time              `json:"build_time"`
	LastScanned    time.Time              `json:"last_scanned"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// Vulnerability represents a security vulnerability
type Vulnerability struct {
	ID          string `json:"id"`
	CVE         string `json:"cve"`
	Severity    string `json:"severity"` // critical, high, medium, low
	Description string `json:"description"`
	FixVersion  string `json:"fix_version,omitempty"`
}

// Attestation represents image signing and verification
type Attestation struct {
	Signature  string    `json:"signature"`
	SignedBy   string    `json:"signed_by"`
	SignedAt   time.Time `json:"signed_at"`
	Verified   bool      `json:"verified"`
	VerifiedAt time.Time `json:"verified_at,omitempty"`
}

// BuildRequest represents a request to build a golden image
type BuildRequest struct {
	Name       string                 `json:"name"`
	BaseOS     string                 `json:"base_os"`
	Platform   string                 `json:"platform"`
	Packages   []string               `json:"packages"`
	Hardening  string                 `json:"hardening,omitempty"`
	Compliance []string               `json:"compliance,omitempty"`
	Scripts    []string               `json:"scripts,omitempty"` // Custom hardening scripts
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// PatchStatus represents the patch status of an image
type PatchStatus struct {
	ImageID        string    `json:"image_id"`
	CurrentVersion string    `json:"current_version"`
	LatestVersion  string    `json:"latest_version"`
	PatchesNeeded  int       `json:"patches_needed"`
	CVEsFixed      []string  `json:"cves_fixed"`
	LastChecked    time.Time `json:"last_checked"`
	UpToDate       bool      `json:"up_to_date"`
}

// ImageRegistry manages golden images
type ImageRegistry struct {
	registryURL string
	images      map[string]*GoldenImage // In-memory storage for MVP
}

func NewImageRegistry() *ImageRegistry {
	registryURL := os.Getenv("REGISTRY_URL")
	if registryURL == "" {
		registryURL = DefaultRegistryURL
	}

	return &ImageRegistry{
		registryURL: registryURL,
		images:      make(map[string]*GoldenImage),
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}

	registry := NewImageRegistry()
	
	r := gin.Default()
	
	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"service": "image-registry",
			"timestamp": time.Now().Unix(),
		})
	})

	// Golden Image Management APIs
	r.POST("/images/build", registry.buildImage)
	r.GET("/images", registry.listImages)
	r.GET("/images/:id", registry.getImage)
	r.POST("/images/:id/scan", registry.scanImage)
	r.POST("/images/:id/sign", registry.signImage)
	r.GET("/images/:id/patch-status", registry.getPatchStatus)
	r.DELETE("/images/:id", registry.deleteImage)

	// Platform-specific image queries
	r.GET("/images/platform/:platform", registry.getImagesByPlatform)
	r.GET("/images/compliance/:framework", registry.getCompliantImages)

	// Drift detection
	r.POST("/drift/detect", registry.detectDrift)
	
	// Metrics endpoint
	r.GET("/metrics", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"total_images": len(registry.images),
			"registry_url": registry.registryURL,
		})
	})

	log.Printf("Starting Image Registry service on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// buildImage initiates a golden image build
func (ir *ImageRegistry) buildImage(c *gin.Context) {
	var req BuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create image metadata
	image := &GoldenImage{
		ID:         uuid.New().String(),
		Name:       req.Name,
		Version:    "1.0.0",
		BaseOS:     req.BaseOS,
		Platform:   req.Platform,
		Packages:   req.Packages,
		Hardening:  req.Hardening,
		Compliance: req.Compliance,
		BuildTime:  time.Now(),
		Metadata:   req.Metadata,
	}

	// In production, this would trigger Packer build
	// For now, simulate the build
	image.RegistryURL = fmt.Sprintf("%s/%s:%s", ir.registryURL, req.Name, image.Version)
	image.Digest = fmt.Sprintf("sha256:%s", uuid.New().String())
	image.Size = 524288000 // 500MB simulated

	// Store in memory (production would use PostgreSQL)
	ir.images[image.ID] = image

	c.JSON(http.StatusAccepted, gin.H{
		"id": image.ID,
		"status": "building",
		"message": fmt.Sprintf("Golden image build initiated for %s", req.Name),
		"estimated_time": "10-15 minutes",
		"image": image,
	})
}

// listImages returns all golden images
func (ir *ImageRegistry) listImages(c *gin.Context) {
	images := make([]*GoldenImage, 0, len(ir.images))
	for _, img := range ir.images {
		images = append(images, img)
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(images),
		"images": images,
	})
}

// getImage returns a specific golden image
func (ir *ImageRegistry) getImage(c *gin.Context) {
	id := c.Param("id")
	
	image, exists := ir.images[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	c.JSON(http.StatusOK, image)
}

// scanImage performs vulnerability scanning on an image
func (ir *ImageRegistry) scanImage(c *gin.Context) {
	id := c.Param("id")
	
	image, exists := ir.images[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	// In production, this would call Trivy or another scanner
	// For now, simulate scan results
	image.Vulnerabilities = []Vulnerability{
		{
			ID:          uuid.New().String(),
			CVE:         "CVE-2024-12345",
			Severity:    "medium",
			Description: "Sample vulnerability for demonstration",
			FixVersion:  "1.0.1",
		},
	}
	image.LastScanned = time.Now()

	c.JSON(http.StatusOK, gin.H{
		"id": id,
		"status": "scanned",
		"vulnerabilities_found": len(image.Vulnerabilities),
		"scan_time": image.LastScanned,
		"vulnerabilities": image.Vulnerabilities,
	})
}

// signImage signs a golden image for attestation
func (ir *ImageRegistry) signImage(c *gin.Context) {
	id := c.Param("id")
	
	image, exists := ir.images[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	// In production, this would use Cosign or Notary
	// For now, simulate signing
	image.Attestation = &Attestation{
		Signature:  fmt.Sprintf("sig-%s", uuid.New().String()),
		SignedBy:   "quantumlayer-ca",
		SignedAt:   time.Now(),
		Verified:   true,
		VerifiedAt: time.Now(),
	}

	c.JSON(http.StatusOK, gin.H{
		"id": id,
		"status": "signed",
		"attestation": image.Attestation,
	})
}

// getPatchStatus checks if an image needs patches
func (ir *ImageRegistry) getPatchStatus(c *gin.Context) {
	id := c.Param("id")
	
	image, exists := ir.images[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	// In production, this would check against CVE databases
	// For now, simulate patch status
	status := PatchStatus{
		ImageID:        id,
		CurrentVersion: image.Version,
		LatestVersion:  "1.0.1",
		PatchesNeeded:  2,
		CVEsFixed:      []string{"CVE-2024-12345", "CVE-2024-67890"},
		LastChecked:    time.Now(),
		UpToDate:       false,
	}

	c.JSON(http.StatusOK, status)
}

// deleteImage removes a golden image
func (ir *ImageRegistry) deleteImage(c *gin.Context) {
	id := c.Param("id")
	
	if _, exists := ir.images[id]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	delete(ir.images, id)

	c.JSON(http.StatusOK, gin.H{
		"id": id,
		"status": "deleted",
	})
}

// getImagesByPlatform returns images for a specific platform
func (ir *ImageRegistry) getImagesByPlatform(c *gin.Context) {
	platform := c.Param("platform")
	
	var images []*GoldenImage
	for _, img := range ir.images {
		if img.Platform == platform {
			images = append(images, img)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"platform": platform,
		"total": len(images),
		"images": images,
	})
}

// getCompliantImages returns images compliant with a framework
func (ir *ImageRegistry) getCompliantImages(c *gin.Context) {
	framework := c.Param("framework")
	
	var images []*GoldenImage
	for _, img := range ir.images {
		for _, comp := range img.Compliance {
			if comp == framework {
				images = append(images, img)
				break
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"framework": framework,
		"total": len(images),
		"images": images,
	})
}

// DriftDetectionRequest represents a drift detection request
type DriftDetectionRequest struct {
	Platform    string   `json:"platform"`
	DataCenter  string   `json:"datacenter,omitempty"`
	Environment string   `json:"environment,omitempty"`
	ImageIDs    []string `json:"image_ids,omitempty"`
}

// DriftReport represents drift detection results
type DriftReport struct {
	Timestamp   time.Time             `json:"timestamp"`
	TotalNodes  int                   `json:"total_nodes"`
	DriftedNodes int                  `json:"drifted_nodes"`
	Details     []DriftDetail         `json:"details"`
}

// DriftDetail represents drift details for a node
type DriftDetail struct {
	NodeID         string `json:"node_id"`
	CurrentImage   string `json:"current_image"`
	ExpectedImage  string `json:"expected_image"`
	DriftType      string `json:"drift_type"` // version, packages, config
	Severity       string `json:"severity"`   // critical, high, medium, low
}

// detectDrift checks for configuration drift
func (ir *ImageRegistry) detectDrift(c *gin.Context) {
	var req DriftDetectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In production, this would query actual infrastructure
	// For now, simulate drift detection
	report := DriftReport{
		Timestamp:    time.Now(),
		TotalNodes:   10,
		DriftedNodes: 2,
		Details: []DriftDetail{
			{
				NodeID:        "node-001",
				CurrentImage:  "ubuntu-20.04-v1.0.0",
				ExpectedImage: "ubuntu-20.04-v1.0.1",
				DriftType:     "version",
				Severity:      "high",
			},
			{
				NodeID:        "node-005",
				CurrentImage:  "centos-8-v2.1.0",
				ExpectedImage: "centos-8-v2.2.0",
				DriftType:     "packages",
				Severity:      "medium",
			},
		},
	}

	c.JSON(http.StatusOK, report)
}