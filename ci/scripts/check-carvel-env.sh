#!/usr/bin/env bash

# Script to check and prompt for required environment variables for Carvel package build

set -e

# ANSI color codes for output formatting
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Required environment variables
REQUIRED_VARS=(
    "CARVEL_PACKAGE_TAG"
    "CARVEL_PACKAGE_VERSION"
    "DOCKER_REGISTRY_USER_NAME"
    "DOCKER_REGISTRY_PASSWORD"
    "DOCKER_ARTIFACTORY_URL"
    "CARVEL_BUNDLE_REGISTRY_URL"
)

# Optional environment variables (for non-local mode)
OPTIONAL_VARS=(
    "DOCKER_BASE_IMAGE"
    "CARVEL_PACKAGE_REPOSITORY_URL"
    "PACKAGE_TYPE"
)

echo -e "${BLUE}🔍 Checking required environment variables for Carvel package build...${NC}"
echo

missing_vars=()
all_vars_set=true

# Check required variables
for var in "${REQUIRED_VARS[@]}"; do
    if [[ -z "${!var}" ]]; then
        missing_vars+=("$var")
        all_vars_set=false
        echo -e "${RED}❌ $var is not set${NC}"
    else
        echo -e "${GREEN}✅ $var is set${NC}"
    fi
done

# Check optional variables and warn if missing
echo
echo -e "${BLUE}📋 Optional variables (required for non-local mode):${NC}"
for var in "${OPTIONAL_VARS[@]}"; do
    if [[ -z "${!var}" ]]; then
        echo -e "${YELLOW}⚠️  $var is not set (required for upload to repository)${NC}"
    else
        echo -e "${GREEN}✅ $var is set${NC}"
    fi
done

if [[ "$all_vars_set" == false ]]; then
    echo
    echo -e "${RED}❌ Missing required environment variables. Please set the following:${NC}"
    echo
    
    for var in "${missing_vars[@]}"; do
        case $var in
            "CARVEL_PACKAGE_TAG")
                echo -e "${YELLOW}export CARVEL_PACKAGE_TAG=<Tag for Carvel Package>${NC}"
                echo -e "   Example: ${BLUE}export CARVEL_PACKAGE_TAG=latest${NC}"
                ;;
            "CARVEL_PACKAGE_VERSION")
                echo -e "${YELLOW}export CARVEL_PACKAGE_VERSION=<Carvel Package Version>${NC}"
                echo -e "   Example: ${BLUE}export CARVEL_PACKAGE_VERSION=1.0.0${NC}"
                ;;
            "DOCKER_REGISTRY_USER_NAME")
                echo -e "${YELLOW}export DOCKER_REGISTRY_USER_NAME=<Docker Registry Username>${NC}"
                echo -e "   Example: ${BLUE}export DOCKER_REGISTRY_USER_NAME=user01${NC}"
                ;;
            "DOCKER_REGISTRY_PASSWORD")
                echo -e "${YELLOW}export DOCKER_REGISTRY_PASSWORD=<Docker Registry Password>${NC}"
                echo -e "   Example: ${BLUE}export DOCKER_REGISTRY_PASSWORD=\"password12\"${NC}"
                ;;
            "DOCKER_ARTIFACTORY_URL")
                echo -e "${YELLOW}export DOCKER_ARTIFACTORY_URL=<Docker Registry URL>${NC}"
                echo -e "   Example: ${BLUE}export DOCKER_ARTIFACTORY_URL=your-docker-registry.example.com${NC}"
                ;;
            "CARVEL_BUNDLE_REGISTRY_URL")
                echo -e "${YELLOW}export CARVEL_BUNDLE_REGISTRY_URL=<Carvel Bundle Registry URL>${NC}"
                echo -e "   Example: ${BLUE}export CARVEL_BUNDLE_REGISTRY_URL=your-docker-registry.example.com/carvel/your-project${NC}"
                ;;
        esac
        echo
    done
    
    echo -e "${BLUE}💡 You can also create a .env file with these variables and source it:${NC}"
    echo -e "${BLUE}   source .env${NC}"
    echo
    exit 1
fi

echo
echo -e "${GREEN}✅ All required environment variables are set!${NC}"
echo -e "${BLUE}🚀 Ready to build Carvel package...${NC}"
echo
