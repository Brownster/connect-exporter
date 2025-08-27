#!/bin/bash

# Build script for AWS Connect Exporter
# Outputs binaries to bin/ folder

set -e

echo "🔨 Building AWS Connect Exporter..."

# Create bin directory if it doesn't exist
mkdir -p bin

# Build the exporter
echo "Building connect-exporter..."
go build -o bin/connect-exporter ./cmd/connect-exporter_exporter

echo "✅ Build complete!"
echo "📦 Binary available at: bin/connect-exporter"
echo ""
echo "🚀 To run the exporter:"
echo "  ./bin/connect-exporter --help"