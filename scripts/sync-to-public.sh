#!/bin/bash

# VMRay Public Repository Sync Script
# This script helps sync code from private repo to public repo

set -e

# Configuration
PRIVATE_REMOTE="origin"
PUBLIC_REMOTE="public"
PRIVATE_BRANCH="main"
PUBLIC_BRANCH="main"
SYNC_BRANCH="public-sync"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if we're in a git repository
check_git_repo() {
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        log_error "Not in a git repository!"
        exit 1
    fi
}

# Function to check if remotes exist
check_remotes() {
    if ! git remote | grep -q "^${PRIVATE_REMOTE}$"; then
        log_error "Remote '${PRIVATE_REMOTE}' not found!"
        exit 1
    fi
    
    if ! git remote | grep -q "^${PUBLIC_REMOTE}$"; then
        log_error "Remote '${PUBLIC_REMOTE}' not found!"
        log_info "Add it with: git remote add ${PUBLIC_REMOTE} https://github.com/vmware/ray-on-vcf.git"
        exit 1
    fi
}

# Function to clean sensitive data
clean_sensitive_data() {
    local branch=$1
    log_info "Cleaning sensitive data from branch ${branch}..."
    
    # Remove GitLab CI configuration
    if [ -f ".gitlab-ci.yml" ]; then
        log_info "Removing GitLab CI configuration"
        rm -f .gitlab-ci.yml
        git add .gitlab-ci.yml
    fi
    
    # Remove internal GitLab workflow if exists
    if [ -f ".github/workflows/cleanup-artifactory.yml" ]; then
        log_info "Removing internal GitHub workflow"
        rm -f .github/workflows/cleanup-artifactory.yml
        git add .github/workflows/cleanup-artifactory.yml
    fi
    
    # Remove GitLab runner Dockerfile
    if [ -f "ci/gitlab-runner/Dockerfile" ]; then
        log_info "Removing GitLab runner configuration"
        rm -rf ci/gitlab-runner/
        git add -A
    fi
    
    # Update PROJECT file references
    if [ -f "vmray-cluster-operator/PROJECT" ]; then
        log_info "Updating PROJECT file references"
        sed -i.bak 's/gitlab\.eng\.vmware\.com\/xlabs\/x77-taiga\/vmray/github.com\/vmware\/ray-on-vcf/g' vmray-cluster-operator/PROJECT
        sed -i.bak 's/domain: broadcom\.com/domain: vmware.com/g' vmray-cluster-operator/PROJECT
        rm -f vmray-cluster-operator/PROJECT.bak
        git add vmray-cluster-operator/PROJECT
    fi
    
    # Clean internal references from Go files
    log_info "Cleaning internal references from Go files"
    find vmray-cluster-operator -name "*.go" -type f -exec sed -i.bak 's/gitlab\.eng\.vmware\.com\/xlabs\/x77-taiga\/vmray\/vmray-cluster-operator/github.com\/vmware\/ray-on-vcf\/vmray-cluster-operator/g' {} \;
    find vmray-cluster-operator -name "*.go.bak" -delete
    
    # Update go.mod if it exists
    if [ -f "vmray-cluster-operator/go.mod" ]; then
        log_info "Updating go.mod module path"
        sed -i.bak 's/gitlab\.eng\.vmware\.com\/xlabs\/x77-taiga\/vmray\/vmray-cluster-operator/github.com\/vmware\/ray-on-vcf\/vmray-cluster-operator/g' vmray-cluster-operator/go.mod
        rm -f vmray-cluster-operator/go.mod.bak
        git add vmray-cluster-operator/go.mod
    fi
    
        
    # Clean Dockerfile references
    if [ -f "vmray-cluster-operator/Dockerfile" ]; then
        log_info "Cleaning Dockerfile references"
        sed -i.bak 's/dockerhub\.artifactory\.vcfd\.broadcom\.net\//docker.io\//g' vmray-cluster-operator/Dockerfile
        sed -i.bak 's/project-taiga-docker-local\.artifactory\.vcfd\.broadcom\.net\//docker.io\//g' vmray-cluster-operator/Dockerfile
        rm -f vmray-cluster-operator/Dockerfile.bak
        git add vmray-cluster-operator/Dockerfile
    fi
    
    # Clean internal URLs from documentation and config files
    log_info "Cleaning internal URLs from documentation"
    find . -name "*.md" -type f -exec sed -i.bak 's/gitlab-vmw\.devops\.broadcom\.net/github.com\/vmware/g' {} \;
    find . -name "*.md" -type f -exec sed -i.bak 's/gitlab\.eng\.vmware\.com/github.com\/vmware/g' {} \;
    find . -name "*.md" -type f -exec sed -i.bak 's/broadcom\.com/vmware.com/g' {} \;
    find . -name "*.md.bak" -delete
    
    # Clean YAML files
    find . -name "*.yaml" -o -name "*.yml" -type f -exec sed -i.bak 's/gitlab\.eng\.vmware\.com/github.com\/vmware/g' {} \;
    find . -name "*.yaml.bak" -o -name "*.yml.bak" -delete
    
    # Remove internal environment files and configs
    if [ -f "bdd-functional-tests/bdd-test.env" ]; then
        log_info "Removing internal test environment file"
        rm -f bdd-functional-tests/bdd-test.env
        git add bdd-functional-tests/bdd-test.env
    fi
    
    
    # Clean requirements.txt files of internal packages
    find . -name "requirements.txt" -type f -exec sed -i.bak '/vcf\./d' {} \;
    find . -name "requirements.txt" -type f -exec sed -i.bak '/broadcom\./d' {} \;
    find . -name "requirements.txt.bak" -delete

    # Update .pre-commit-config.yaml to remove local hook that uses ci/ directory
    if [ -f ".pre-commit-config.yaml" ]; then
        log_info "Updating .pre-commit-config.yaml to remove local hook"
        cat > .pre-commit-config.yaml << 'EOF'
repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.5.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-added-large-files
  - repo: https://github.com/Bahjat/pre-commit-golang
    rev: v1.0.3
    hooks:
      - id: go-fmt-import
      - id: golangci-lint
      - id: go-static-check
EOF
        git add .pre-commit-config.yaml
    fi
    
  
    # Stage all changes
    git add -A
    
    log_success "Sensitive data cleaning completed"
}

# Function to prepare public sync branch
prepare_sync_branch() {
    log_info "Preparing sync branch..."
    
    # Fetch latest from private repo
    git fetch ${PRIVATE_REMOTE}
    
    # Check if sync branch exists
    if git branch | grep -q "${SYNC_BRANCH}"; then
        log_info "Switching to existing ${SYNC_BRANCH} branch"
        git checkout ${SYNC_BRANCH}
        git reset --hard ${PRIVATE_REMOTE}/${PRIVATE_BRANCH}
    else
        log_info "Creating new ${SYNC_BRANCH} branch"
        git checkout -b ${SYNC_BRANCH} ${PRIVATE_REMOTE}/${PRIVATE_BRANCH}
    fi
    
    # Clean sensitive data
    clean_sensitive_data ${SYNC_BRANCH}
    
    # Commit changes if any
    if ! git diff --cached --quiet; then
        git commit -m "Clean sensitive data for public release"
    fi
    
    log_success "Sync branch prepared"
}

# Function to sync to public repository
sync_to_public() {
    log_info "Syncing to public repository..."
    
    # Fetch from public repo to check for conflicts
    git fetch ${PUBLIC_REMOTE}
    
    # Push to public repo
    git push --force ${PUBLIC_REMOTE} ${SYNC_BRANCH}:refs/heads/${SYNC_BRANCH}
    
    log_success "Code synced to public repository"
    log_info "You can now create a PR from ${SYNC_BRANCH} to ${PUBLIC_BRANCH} on GitHub"
}

# Function to create a feature branch for development
create_feature_branch() {
    local feature_name=$1
    
    if [ -z "$feature_name" ]; then
        log_error "Feature name is required!"
        echo "Usage: $0 feature <feature-name>"
        exit 1
    fi
    
    local branch_name="feature/${feature_name}"
    
    log_info "Creating feature branch: ${branch_name}"
    
    # Ensure we're on main and up to date
    git checkout ${PRIVATE_BRANCH}
    git pull ${PRIVATE_REMOTE} ${PRIVATE_BRANCH}
    
    # Create and switch to feature branch
    git checkout -b ${branch_name}
    
    log_success "Feature branch '${branch_name}' created"
    log_info "You can now develop your feature and push to private repo"
}

# Main script logic
case "${1:-}" in
    "sync")
        check_git_repo
        check_remotes
        prepare_sync_branch
        sync_to_public
        ;;
    "feature")
        check_git_repo
        check_remotes
        create_feature_branch "$2"
        ;;
    "clean")
        check_git_repo
        if [ "$(git branch --show-current)" != "${SYNC_BRANCH}" ]; then
            log_error "Must be on ${SYNC_BRANCH} branch to clean"
            exit 1
        fi
        clean_sensitive_data "${SYNC_BRANCH}"
        ;;
    *)
        echo "VMRay Public Repository Sync Script"
        echo ""
        echo "Usage:"
        echo "  $0 sync                    - Sync current main to public repo"
        echo "  $0 feature <name>          - Create a new feature branch"
        echo "  $0 clean                   - Clean sensitive data from current branch"
        echo ""
        echo "Examples:"
        echo "  $0 feature user-auth       - Creates feature/user-auth branch"
        echo "  $0 sync                    - Sync to public repo and prepare for PR"
        exit 1
        ;;
esac
