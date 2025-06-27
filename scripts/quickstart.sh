#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="${SCRIPT_DIR}/.."
MOCK_DIR="$ROOT_DIR/mock-api"

# Install Node dependencies if missing
if [ ! -d "$MOCK_DIR/node_modules" ]; then
  echo "Installing mock API dependencies..."
  (cd "$MOCK_DIR" && npm install)
fi

# Start mock API server
echo "Starting mock API..."
node "$MOCK_DIR/server.js" &
MOCK_PID=$!
sleep 2

# Export environment variables for the exporter
export ENV=development
export AWS_ACCESS_KEY_ID=mock-access-key-id
export AWS_SECRET_ACCESS_KEY=mock-secret-key
export AWS_SESSION_TOKEN=mock-session-token
export AWS_REGION=eu-west-2
export AWS_USE_SSL=false
export AWS_ENDPOINT_URL_CONNECT=http://localhost:3000

cd "$ROOT_DIR"
./aws-connect-exporter --config.file=config.development.yml --region=eu-west-2 --no-advertise

# Cleanup
kill $MOCK_PID
