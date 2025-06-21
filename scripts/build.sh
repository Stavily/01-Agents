#!/bin/bash

# Stavily Agents Build Script
# This script builds both sensor and action agents for multiple platforms

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Build configuration
VERSION="${VERSION:-dev}"
BUILD_TIME="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
BUILD_DIR="${PROJECT_ROOT}/bin"
DIST_DIR="${PROJECT_ROOT}/dist"

# Go build flags
LDFLAGS="-w -s -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}"
BUILD_FLAGS="-trimpath -ldflags=\"${LDFLAGS}\""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
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

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    local go_version
    go_version=$(go version | cut -d' ' -f3 | sed 's/go//')
    log_info "Using Go version: ${go_version}"
    
    if ! command -v git &> /dev/null; then
        log_warning "Git is not installed - version info may be incomplete"
    fi
}

# Clean build artifacts
clean() {
    log_info "Cleaning build artifacts..."
    rm -rf "${BUILD_DIR}" "${DIST_DIR}"
    
    # Clean module caches in each component
    for component in shared sensor-agent action-agent; do
        if [[ -d "${PROJECT_ROOT}/${component}" ]]; then
            cd "${PROJECT_ROOT}/${component}"
            go clean -cache -testcache
        fi
    done
    
    log_success "Clean completed"
}

# Download dependencies
download_deps() {
    log_info "Downloading dependencies..."
    
    for component in shared sensor-agent action-agent; do
        if [[ -d "${PROJECT_ROOT}/${component}" ]]; then
            log_info "Downloading dependencies for ${component}..."
            cd "${PROJECT_ROOT}/${component}"
            go mod download
            go mod tidy
        fi
    done
    
    log_success "Dependencies downloaded"
}

# Build single agent for single platform
build_agent() {
    local agent="$1"
    local goos="$2"
    local goarch="$3"
    
    local binary_name="${agent}"
    if [[ "${goos}" == "windows" ]]; then
        binary_name="${agent}.exe"
    fi
    
    local output_path="${BUILD_DIR}/${agent}-${goos}-${goarch}"
    if [[ "${goos}" == "windows" ]]; then
        output_path="${output_path}.exe"
    fi
    
    log_info "Building ${agent} for ${goos}/${goarch}..."
    
    cd "${PROJECT_ROOT}/${agent}"
    
    env GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 \
        go build ${BUILD_FLAGS} \
        -o "${output_path}" \
        "cmd/${agent}/main.go"
    
    if [[ -f "${output_path}" ]]; then
        log_success "Built ${agent} for ${goos}/${goarch}: $(basename "${output_path}")"
        
        # Create platform-specific directory and copy binary
        local platform_dir="${DIST_DIR}/${agent}/${goos}-${goarch}"
        mkdir -p "${platform_dir}"
        cp "${output_path}" "${platform_dir}/${binary_name}"
        
        # Copy configuration examples
        if [[ -d "configs" ]]; then
            cp -r configs "${platform_dir}/"
        fi
        
        # Copy documentation
        if [[ -f "README.md" ]]; then
            cp "README.md" "${platform_dir}/"
        fi
        
        # Create archive
        create_archive "${agent}" "${goos}" "${goarch}"
    else
        log_error "Failed to build ${agent} for ${goos}/${goarch}"
        return 1
    fi
}

# Create distribution archive
create_archive() {
    local agent="$1"
    local goos="$2"
    local goarch="$3"
    
    local platform_dir="${DIST_DIR}/${agent}/${goos}-${goarch}"
    local archive_name="${agent}-${VERSION}-${goos}-${goarch}"
    
    cd "${DIST_DIR}/${agent}"
    
    if [[ "${goos}" == "windows" ]]; then
        # Create ZIP for Windows
        if command -v zip &> /dev/null; then
            zip -r "${archive_name}.zip" "${goos}-${goarch}/"
            log_info "Created archive: ${archive_name}.zip"
        fi
    else
        # Create tar.gz for Unix-like systems
        tar -czf "${archive_name}.tar.gz" "${goos}-${goarch}/"
        log_info "Created archive: ${archive_name}.tar.gz"
    fi
}

# Build all agents for all platforms
build_all() {
    log_info "Building all agents for all platforms..."
    
    # Platforms to build for
    local platforms=(
        "linux amd64"
        "linux arm64"
        "darwin amd64"
        "darwin arm64"
        "windows amd64"
    )
    
    # Agents to build
    local agents=("sensor-agent" "action-agent")
    
    mkdir -p "${BUILD_DIR}" "${DIST_DIR}"
    
    for agent in "${agents[@]}"; do
        if [[ ! -d "${PROJECT_ROOT}/${agent}" ]]; then
            log_warning "Agent directory not found: ${agent}"
            continue
        fi
        
        for platform in "${platforms[@]}"; do
            read -r goos goarch <<< "${platform}"
            build_agent "${agent}" "${goos}" "${goarch}" || log_error "Failed to build ${agent} for ${goos}/${goarch}"
        done
    done
}

# Build for current platform only
build_local() {
    log_info "Building for current platform..."
    
    local goos
    local goarch
    goos=$(go env GOOS)
    goarch=$(go env GOARCH)
    
    log_info "Target platform: ${goos}/${goarch}"
    
    mkdir -p "${BUILD_DIR}"
    
    for agent in sensor-agent action-agent; do
        if [[ -d "${PROJECT_ROOT}/${agent}" ]]; then
            build_agent "${agent}" "${goos}" "${goarch}"
        fi
    done
}

# Run tests
run_tests() {
    log_info "Running tests..."
    
    for component in shared sensor-agent action-agent; do
        if [[ -d "${PROJECT_ROOT}/${component}" ]]; then
            log_info "Testing ${component}..."
            cd "${PROJECT_ROOT}/${component}"
            
            if ! go test -v -race -cover ./...; then
                log_error "Tests failed for ${component}"
                return 1
            fi
        fi
    done
    
    log_success "All tests passed"
}

# Run linting
run_lint() {
    log_info "Running linting..."
    
    if ! command -v golangci-lint &> /dev/null; then
        log_warning "golangci-lint not found, skipping linting"
        return 0
    fi
    
    for component in shared sensor-agent action-agent; do
        if [[ -d "${PROJECT_ROOT}/${component}" ]]; then
            log_info "Linting ${component}..."
            cd "${PROJECT_ROOT}/${component}"
            
            if ! golangci-lint run; then
                log_error "Linting failed for ${component}"
                return 1
            fi
        fi
    done
    
    log_success "Linting completed"
}

# Generate checksums
generate_checksums() {
    log_info "Generating checksums..."
    
    cd "${DIST_DIR}"
    
    for agent in sensor-agent action-agent; do
        if [[ -d "${agent}" ]]; then
            cd "${agent}"
            
            # Generate SHA256 checksums for all archives
            find . -name "*.tar.gz" -o -name "*.zip" | while read -r file; do
                if command -v sha256sum &> /dev/null; then
                    sha256sum "${file}" >> "checksums.sha256"
                elif command -v shasum &> /dev/null; then
                    shasum -a 256 "${file}" >> "checksums.sha256"
                fi
            done
            
            if [[ -f "checksums.sha256" ]]; then
                log_info "Generated checksums for ${agent}"
            fi
            
            cd ..
        fi
    done
}

# Show usage
usage() {
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  build-local    Build for current platform only"
    echo "  build-all      Build for all platforms"
    echo "  test           Run tests"
    echo "  lint           Run linting"
    echo "  clean          Clean build artifacts"
    echo "  deps           Download dependencies"
    echo "  checksums      Generate checksums"
    echo "  help           Show this help"
    echo ""
    echo "Environment variables:"
    echo "  VERSION        Version to build (default: dev)"
    echo ""
    echo "Examples:"
    echo "  $0 build-local"
    echo "  VERSION=1.0.0 $0 build-all"
    echo "  $0 test && $0 build-local"
}

# Main function
main() {
    cd "${PROJECT_ROOT}"
    
    case "${1:-build-local}" in
        "build-local")
            check_prerequisites
            download_deps
            build_local
            ;;
        "build-all")
            check_prerequisites
            download_deps
            build_all
            generate_checksums
            ;;
        "test")
            check_prerequisites
            download_deps
            run_tests
            ;;
        "lint")
            check_prerequisites
            run_lint
            ;;
        "clean")
            clean
            ;;
        "deps")
            check_prerequisites
            download_deps
            ;;
        "checksums")
            generate_checksums
            ;;
        "help"|"-h"|"--help")
            usage
            ;;
        *)
            log_error "Unknown command: $1"
            usage
            exit 1
            ;;
    esac
}

# Run main function
main "$@" 