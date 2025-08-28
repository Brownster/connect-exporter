#!/bin/bash

# Kensington Contact Center Dashboard Import Script
# This script imports both dashboards into Grafana via API

set -e

# Configuration
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"
GRAFANA_USER="${GRAFANA_USER:-admin}"
GRAFANA_PASSWORD="${GRAFANA_PASSWORD:-admin}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Kensington Contact Center Dashboard Import${NC}"
echo "================================================="

# Check if Grafana is accessible
echo -e "${YELLOW}📡 Checking Grafana connection...${NC}"
if curl -s -f -u "$GRAFANA_USER:$GRAFANA_PASSWORD" "$GRAFANA_URL/api/health" > /dev/null; then
    echo -e "${GREEN}✅ Grafana is accessible at $GRAFANA_URL${NC}"
else
    echo -e "${RED}❌ Cannot connect to Grafana at $GRAFANA_URL${NC}"
    echo "Please check:"
    echo "  - Grafana is running"
    echo "  - URL is correct (set GRAFANA_URL env var)"
    echo "  - Credentials are correct (set GRAFANA_USER/GRAFANA_PASSWORD env vars)"
    exit 1
fi

# Function to import a dashboard
import_dashboard() {
    local file=$1
    local name=$2
    
    echo -e "${YELLOW}📊 Importing $name...${NC}"
    
    # Prepare the import payload
    local dashboard_json=$(cat "$file")
    local import_payload=$(cat <<EOF
{
  "dashboard": $dashboard_json,
  "overwrite": true,
  "inputs": [
    {
      "name": "DS_PROMETHEUS",
      "type": "datasource",
      "pluginId": "prometheus",
      "value": "Prometheus"
    }
  ]
}
EOF
)
    
    # Import the dashboard
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -u "$GRAFANA_USER:$GRAFANA_PASSWORD" \
        -d "$import_payload" \
        "$GRAFANA_URL/api/dashboards/import")
    
    # Check if import was successful
    if echo "$response" | grep -q '"status":"success"'; then
        local dashboard_url=$(echo "$response" | jq -r '.importedUrl // empty')
        echo -e "${GREEN}✅ $name imported successfully${NC}"
        if [ -n "$dashboard_url" ]; then
            echo -e "   📱 URL: $GRAFANA_URL$dashboard_url"
        fi
    else
        echo -e "${RED}❌ Failed to import $name${NC}"
        echo "Response: $response"
        return 1
    fi
}

# Check if jq is available
if ! command -v jq &> /dev/null; then
    echo -e "${YELLOW}⚠️ jq not found, dashboard URLs won't be displayed${NC}"
fi

# Import dashboards
echo -e "\n${BLUE}📊 Importing Dashboards${NC}"
echo "========================"

# Real-time Operations Dashboard
if [ -f "kensington-realtime-operations.json" ]; then
    import_dashboard "kensington-realtime-operations.json" "Real-time Operations Dashboard"
else
    echo -e "${RED}❌ kensington-realtime-operations.json not found${NC}"
fi

echo ""

# Daily Performance Dashboard
if [ -f "kensington-daily-performance.json" ]; then
    import_dashboard "kensington-daily-performance.json" "Daily Performance Analytics Dashboard"
else
    echo -e "${RED}❌ kensington-daily-performance.json not found${NC}"
fi

echo ""
echo -e "${GREEN}🎉 Dashboard import complete!${NC}"
echo ""
echo -e "${BLUE}Next Steps:${NC}"
echo "1. 🔧 Configure Prometheus data source in Grafana"
echo "2. 🚀 Start AWS Connect Exporter with metrics collection"
echo "3. 📊 Open dashboards and verify data is flowing"
echo ""
echo -e "${YELLOW}💡 Tip: Use the enhanced mock API for testing without AWS Connect${NC}"
echo "   ENV=development ./connect-exporter --config.file=config.test.yml"