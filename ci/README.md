# Carvel Package Build Guide

This guide explains how to use the automated Carvel package build system.

## Overview

The `build-carvel-package` make target provides a fully automated way to build Carvel packages with all prerequisites installed in a Docker container. This eliminates the need to manually install tools and manage Python environments on your local machine.

## Quick Start

1. **Set Environment Variables**

   Copy the example environment file and customize it:
   ```bash
   cp ci/env.example .env
   # Edit .env with your actual values
   source .env
   ```

2. **Build Carvel Package**

   From the `vmray-cluster-operator` directory:
   ```bash
   make build-carvel-package
   ```
   
   This single command will:
   - Validate all required environment variables
   - Build the Carvel build environment Docker image
   - Build the VMRay cluster controller image  
   - Push the controller image to your registry
   - Update Kubernetes parameters
   - Build the complete Carvel package

3. **(Optinal)Build and Upload to Repository**

   To build and upload to the repository (requires additional env vars):
   ```bash
   make build-carvel-package-upload
   ```

## Environment Variables

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `CARVEL_PACKAGE_TAG` | Tag for Carvel Package | `latest` |
| `CARVEL_PACKAGE_VERSION` | Carvel Package Version | `1.0.0` |
| `DOCKER_REGISTRY_USER_NAME` | Docker Registry Username | `user01` |
| `DOCKER_REGISTRY_PASSWORD` | Docker Registry Password | `"password12"` |
| `DOCKER_ARTIFACTORY_URL` | Docker Registry URL | `your-docker-registry.example.com` |
| `CARVEL_BUNDLE_REGISTRY_URL` | Docker registry URL for Carvel bundles | `your-docker-registry.example.com/carvel/your-project` |

### Optional Variables (for upload)

| Variable | Description | Example |
|----------|-------------|---------|
| `DOCKER_BASE_IMAGE` | Base Docker image for build environment | `ubuntu:22.04` or `dockerhub-proxy.your-registry.example.com/library/ubuntu:22.04` |
| `CARVEL_PACKAGE_REPOSITORY_URL` | Generic repository URL for package YAML uploads | `https://your-artifactory-url/artifactory` |
| `PACKAGE_TYPE` | Package type for categorization | `your_package_type` |

## Docker Base Image Configuration

The build process uses a Docker container with all prerequisites installed. You can configure which base image to use:

### For Public Docker Hub (Default)
```bash
export DOCKER_BASE_IMAGE=ubuntu:22.04
# or simply don't set the variable - it defaults to ubuntu:22.04
```

### For Corporate Docker Registry Proxy
```bash
export DOCKER_BASE_IMAGE=dockerhub-proxy.your-registry.example.com/library/ubuntu:22.04
```

This allows you to use the same Dockerfile in both corporate environments (with Docker registry proxies) and public environments.

## Make Targets

### `build-carvel-package`
- Builds the Carvel package locally
- Creates Docker image with all prerequisites
- Runs the build process in an isolated container
- Outputs package to `vmray-cluster-operator/artifacts/carvel-imgpkg/`

### `build-carvel-package-upload`
- Same as `build-carvel-package` but also uploads to repository
- Requires additional environment variables for upload

### `clean-carvel`
- Cleans up build artifacts and Docker images
- Removes the `artifacts/carvel-imgpkg/` directory
- Removes the Carvel builder Docker image

## What the Build Process Does

1. **Environment Check**: Validates all required environment variables
2. **Docker Image Build**: Creates a container with all prerequisites:
   - Ubuntu 22.04 base
   - Docker, Python3, pip, kustomize, yq, sed
   - Carvel tools (kbld, imgpkg, ytt, kapp)
   - Python virtual environment with required packages
3. **Controller Image**: Builds the VMRay cluster controller Docker image
4. **Kubernetes Parameters**: Updates K8s parameters with the correct image reference
5. **Carvel Package**: Creates the final Carvel package using the containerized environment

## Prerequisites

- Docker installed and running
- Access to the Docker registry specified in environment variables
- Proper credentials for the Docker registry

## Troubleshooting

### Environment Variable Issues
If you see environment variable errors, run the check script directly:
```bash
./ci/scripts/check-carvel-env.sh
```

### Docker Issues
Ensure Docker is running and you have permissions:
```bash
docker ps
```

### Build Failures
Check the Docker logs and ensure all environment variables are correctly set. The build process will stop at the first error and provide detailed output.

## Files Created

- `ci/Dockerfile.carvel`: Docker image definition for build environment
- `ci/scripts/check-carvel-env.sh`: Environment variable validation script
- `ci/env.example`: Example environment variable file
- Updated `vmray-cluster-operator/Makefile`: New make targets

## Comparison with Manual Process

| Manual Process | Automated Process |
|----------------|-------------------|
| Install tools locally | Tools in Docker container |
| Manage Python venv | Automated venv setup |
| Manual dependency installation | Automated dependency management |
| Multiple manual steps | Single make command |
| Environment conflicts possible | Isolated container environment |

The automated process eliminates the need for local tool installation and provides a consistent, reproducible build environment.
