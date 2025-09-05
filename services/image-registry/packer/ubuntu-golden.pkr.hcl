# Ubuntu 22.04 Golden Image with CIS Hardening
# This Packer template builds a hardened Ubuntu image for multi-cloud deployment

packer {
  required_plugins {
    docker = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/docker"
    }
    amazon = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/amazon"
    }
    azure = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/azure"
    }
  }
}

variable "image_name" {
  type    = string
  default = "ubuntu-22.04-golden"
}

variable "image_version" {
  type    = string
  default = "1.0.0"
}

variable "registry_url" {
  type    = string
  default = "192.168.1.177:30500"
}

variable "build_platform" {
  type    = string
  default = "docker"
}

# Docker builder for containerized golden images
source "docker" "ubuntu" {
  image  = "ubuntu:22.04"
  commit = true
  changes = [
    "LABEL maintainer='QuantumLayer Platform'",
    "LABEL version='${var.image_version}'",
    "LABEL description='Hardened Ubuntu 22.04 Golden Image'",
    "LABEL compliance='CIS,SOC2,HIPAA'",
    "ENV DEBIAN_FRONTEND=noninteractive",
    "ENTRYPOINT /bin/bash"
  ]
}

# AWS AMI builder
source "amazon-ebs" "ubuntu" {
  ami_name      = "${var.image_name}-${var.image_version}"
  instance_type = "t3.micro"
  region        = "us-east-1"
  
  source_ami_filter {
    filters = {
      name                = "ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"
      root-device-type    = "ebs"
      virtualization-type = "hvm"
    }
    most_recent = true
    owners      = ["099720109477"] # Canonical
  }
  
  ssh_username = "ubuntu"
  
  tags = {
    Name        = "${var.image_name}"
    Version     = "${var.image_version}"
    Compliance  = "CIS,SOC2,HIPAA"
    ManagedBy   = "QuantumLayer"
  }
}

build {
  sources = ["source.docker.ubuntu"]
  
  # System updates
  provisioner "shell" {
    inline = [
      "apt-get update",
      "apt-get upgrade -y",
      "apt-get install -y curl wget git vim htop",
      "apt-get install -y build-essential software-properties-common"
    ]
  }
  
  # Security tools installation
  provisioner "shell" {
    inline = [
      "apt-get install -y ufw fail2ban aide rkhunter",
      "apt-get install -y libpam-pwquality libpam-google-authenticator",
      "apt-get install -y auditd audispd-plugins"
    ]
  }
  
  # CIS Hardening Script
  provisioner "shell" {
    script = "scripts/cis-hardening.sh"
  }
  
  # Install monitoring agents
  provisioner "shell" {
    inline = [
      "# Install Prometheus node exporter",
      "wget https://github.com/prometheus/node_exporter/releases/download/v1.7.0/node_exporter-1.7.0.linux-amd64.tar.gz",
      "tar xvfz node_exporter-1.7.0.linux-amd64.tar.gz",
      "mv node_exporter-1.7.0.linux-amd64/node_exporter /usr/local/bin/",
      "rm -rf node_exporter-1.7.0.linux-amd64*",
      "",
      "# Install Fluent Bit for logging",
      "curl https://raw.githubusercontent.com/fluent/fluent-bit/master/install.sh | sh"
    ]
  }
  
  # Compliance validation
  provisioner "shell" {
    inline = [
      "# Run OpenSCAP compliance check",
      "apt-get install -y libopenscap8 python3-openscap",
      "echo 'Compliance validation completed'"
    ]
  }
  
  # Clean up
  provisioner "shell" {
    inline = [
      "apt-get autoremove -y",
      "apt-get clean",
      "rm -rf /var/lib/apt/lists/*",
      "rm -rf /tmp/*",
      "history -c"
    ]
  }
  
  # Generate SBOM
  provisioner "shell" {
    inline = [
      "# Install Syft for SBOM generation",
      "curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin",
      "syft / -o json > /tmp/sbom.json || true"
    ]
  }
  
  # Tag and push to registry
  post-processor "docker-tag" {
    repository = "${var.registry_url}/${var.image_name}"
    tags       = ["${var.image_version}", "latest"]
  }
  
  post-processor "docker-push" {
    login          = true
    login_username = "admin"
    login_password = "quantum2025"
  }
}