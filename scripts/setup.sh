#!/bin/bash
#
# AWS Connect Exporter - Quick Setup Script
#
# This script helps new users get started quickly with the AWS Connect Exporter
#

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 AWS Connect Exporter - Quick Setup${NC}"
echo

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed. Please install Go 1.21 or newer.${NC}"
    echo "   Visit: https://golang.org/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo -e "${GREEN}✅ Go $GO_VERSION detected${NC}"

# Check if we're in the right directory
if [[ ! -f "go.mod" ]] || [[ ! -d "cmd/connect-exporter_exporter" ]]; then
    echo -e "${RED}❌ Please run this script from the project root directory${NC}"
    exit 1
fi

echo -e "${YELLOW}📦 Installing dependencies...${NC}"
go mod tidy

echo -e "${YELLOW}🔨 Building exporter...${NC}"
./scripts/build.sh

echo -e "${YELLOW}🧪 Setting up development environment...${NC}"

# Check if mock API dependencies are installed
if [[ -d "docs/tools/aws-mock-api-main" ]]; then
    cd docs/tools/aws-mock-api-main
    if [[ ! -d "node_modules" ]]; then
        echo -e "${YELLOW}📦 Installing Node.js dependencies for mock API...${NC}"
        if command -v npm &> /dev/null; then
            npm install
            echo -e "${GREEN}✅ Mock API dependencies installed${NC}"
        else
            echo -e "${YELLOW}⚠️  npm not found. You'll need Node.js to use the development mock API${NC}"
            echo "   Visit: https://nodejs.org/"
        fi
    else
        echo -e "${GREEN}✅ Mock API dependencies already installed${NC}"
    fi
    cd - > /dev/null
fi

echo
echo -e "${GREEN}🎉 Setup Complete!${NC}"
echo
echo -e "${BLUE}📋 Next Steps:${NC}"
echo

echo -e "${YELLOW}1. For Development (No AWS costs):${NC}"
echo "   # Terminal 1: Start mock API"
echo "   cd docs/tools/aws-mock-api-main && node server.js"
echo
echo "   # Terminal 2: Run exporter"
echo "   ENV=development ./bin/aws-connect-exporter \\"
echo "     --web.listen-port=9090 \\"
echo "     --labels.fqdn=localhost \\"
echo "     --labels.location=test \\"
echo "     --labels.country=UK \\"
echo "     --labels.ip-address=127.0.0.1 \\"
echo "     --config.file=configs/config.development.yml \\"
echo "     --no-advertise \\"
echo "     --log.level=debug"
echo

echo -e "${YELLOW}2. For Production:${NC}"
echo "   # 1. Configure your AWS credentials"
echo "   export AWS_PROFILE=your-profile"
echo "   export AWS_REGION=us-east-1"
echo
echo "   # 2. Update config.aws-connect-validated.yml with your instance ARN"
echo "   # 3. Run the exporter"
echo "   ./bin/aws-connect-exporter \\"
echo "     --web.listen-port=9090 \\"
echo "     --labels.fqdn=your-hostname \\"
echo "     --labels.location=your-location \\"
echo "     --labels.country=US \\"
echo "     --labels.ip-address=127.0.0.1 \\"
echo "     --config.file=config.aws-connect-validated.yml \\"
echo "     --no-advertise"
echo

echo -e "${YELLOW}3. Check Metrics:${NC}"
echo "   curl http://localhost:9090/metrics"
echo "   curl http://localhost:9090/health"
echo

echo -e "${BLUE}📚 Documentation:${NC}"
echo "   Setup Guide:     docs/aws-connect-setup.md"
echo "   Quick Start:     docs/quick-start.md"  
echo "   Configuration:   configs/README.md"
echo "   Development:     docs/development-with-mock-api.md"
echo

echo -e "${GREEN}Happy monitoring! 📊${NC}"