#!/bin/bash
#
# AWS Connect Exporter - Build Script
# 
# This script builds the AWS Connect Exporter binary with proper versioning
# and places it in the bin/ directory.
#

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo -e "${BLUE}🔨 AWS Connect Exporter Build Script${NC}"
echo "Project root: $PROJECT_ROOT"

# Change to project root
cd "$PROJECT_ROOT"

# Ensure bin directory exists
mkdir -p bin

# Get version information
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "unknown")
BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION=$(go version | awk '{print $3}')

echo -e "${YELLOW}📦 Build Information:${NC}"
echo "  Version: $VERSION"
echo "  Branch: $BRANCH"  
echo "  Commit: $COMMIT"
echo "  Build Time: $BUILD_TIME"
echo "  Go Version: $GO_VERSION"

# Build flags for version information
LDFLAGS="-X main.version=$VERSION -X main.branch=$BRANCH -X main.revision=$COMMIT -X main.buildDate=$BUILD_TIME"

echo -e "${YELLOW}🏗️  Building binary...${NC}"

# Build the binary
go build \
    -ldflags "$LDFLAGS" \
    -o bin/aws-connect-exporter \
    ./cmd/connect-exporter_exporter

# Check if build was successful
if [[ -f "bin/aws-connect-exporter" ]]; then
    # Get binary size
    BINARY_SIZE=$(du -h bin/aws-connect-exporter | cut -f1)
    
    echo -e "${GREEN}✅ Build completed successfully!${NC}"
    echo "  Output: bin/aws-connect-exporter"
    echo "  Size: $BINARY_SIZE"
    
    # Make it executable
    chmod +x bin/aws-connect-exporter
    
    # Show binary info
    echo -e "${BLUE}🔍 Binary Information:${NC}"
    ls -la bin/aws-connect-exporter
    
    # Test binary (show help)
    echo -e "${BLUE}🧪 Testing binary:${NC}"
    if ./bin/aws-connect-exporter --help > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Binary test passed${NC}"
    else
        echo -e "${RED}❌ Binary test failed${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}🚀 Build complete! Ready to run:${NC}"
    echo "  ./bin/aws-connect-exporter --help"
    
else
    echo -e "${RED}❌ Build failed - binary not found${NC}"
    exit 1
fi