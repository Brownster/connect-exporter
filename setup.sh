#!/bin/bash
# Setup script for connect-exporter
set -e

# Install Go module dependencies
if [ -f go.mod ]; then
  echo "Downloading Go modules..."
  go mod download
fi

# Install Node dependencies for mock API
MOCK_API_DIR="docs/tools/aws-mock-api-main"
if [ -d "$MOCK_API_DIR" ]; then
  echo "Installing mock API dependencies..."
  pushd "$MOCK_API_DIR" >/dev/null
  if [ -f package.json ]; then
    npm install
  else
    npm install express body-parser
  fi
  popd >/dev/null
fi

echo "Setup complete."

