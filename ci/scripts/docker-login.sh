#!/usr/bin/env bash

# Script to handle Docker login for corporate environment

set -e

# ANSI color codes for output formatting
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔑 Handling Docker authentication...${NC}"

# Check if we need to login to the corporate registry
if [[ -n "${DOCKER_ARTIFACTORY_URL}" && -n "${DOCKER_REGISTRY_USER_NAME}" && -n "${DOCKER_REGISTRY_PASSWORD}" ]]; then
    echo -e "${BLUE}📡 Logging into corporate Docker registry: ${DOCKER_ARTIFACTORY_URL}${NC}"
    echo "${DOCKER_REGISTRY_PASSWORD}" | docker login "${DOCKER_ARTIFACTORY_URL}" -u "${DOCKER_REGISTRY_USER_NAME}" --password-stdin
    
    echo -e "${BLUE}📡 Logging into Dockerhub proxy: dockerhub-proxy.your-registry.example.com${NC}"
    echo "${DOCKER_REGISTRY_PASSWORD}" | docker login dockerhub-proxy.your-registry.example.com -u "${DOCKER_REGISTRY_USER_NAME}" --password-stdin
    
    echo -e "${GREEN}✅ Docker authentication successful${NC}"
else
    echo -e "${YELLOW}⚠️  Docker credentials not found, attempting without authentication${NC}"
    echo -e "${YELLOW}   If build fails, ensure DOCKER_ARTIFACTORY_URL, DOCKER_REGISTRY_USER_NAME, and DOCKER_REGISTRY_PASSWORD are set${NC}"
fi
