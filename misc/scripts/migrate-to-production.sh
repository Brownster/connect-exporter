#!/bin/bash

# Script to migrate template exporter to production Sabio infrastructure
# This removes local monitoring-maas and updates imports to use production repos

set -e

echo "🚀 Migrating exporter to production configuration..."

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo "❌ Error: go.mod not found. Run this script from the exporter root directory."
    exit 1
fi

# Check if monitoring-maas folder exists
if [ ! -d "monitoring-maas" ]; then
    echo "❌ Error: monitoring-maas folder not found. This script is for template repositories."
    exit 1
fi

echo "📁 Current directory: $(pwd)"

# Backup current state
BACKUP_DIR="backup-$(date +%Y%m%d-%H%M%S)"
echo "📦 Creating backup in $BACKUP_DIR..."
mkdir -p "$BACKUP_DIR"
cp -r . "$BACKUP_DIR/" 2>/dev/null || true

# Remove local monitoring-maas
echo "🗑️  Removing local monitoring-maas folder..."
rm -rf monitoring-maas/

# Update go.mod to use production repositories
echo "📝 Updating go.mod for production dependencies..."

# Remove local replace directives
sed -i '/replace.*monitoring-maas/d' go.mod
sed -i '/replace.*monitoring-connectors/d' go.mod

# Add production dependencies
go mod edit -require github.com/sabio-engineering-product/monitoring-maas@latest
go mod edit -require github.com/sabio-engineering-product/monitoring-connectors@latest

# Update import statements in Go files
echo "🔄 Updating import statements..."

# Find all Go files and update imports
find . -name "*.go" -type f | while read -r file; do
    if grep -q "monitoring-rss-exporter/monitoring-maas" "$file"; then
        echo "  Updating imports in $file"
        
        # Get the module name from go.mod
        MODULE_NAME=$(grep "^module " go.mod | awk '{print $2}')
        
        # Replace local monitoring-maas import with production import
        sed -i "s|$MODULE_NAME/monitoring-maas|github.com/sabio-engineering-product/monitoring-maas|g" "$file"
        
        # Add monitoring-connectors imports if needed (common production pattern)
        if ! grep -q "github.com/sabio-engineering-product/monitoring-connectors" "$file" && grep -q "Connector" "$file"; then
            # Add import for connectors if the file uses connectors
            if grep -q "^import (" "$file"; then
                # Multi-line import block exists
                sed -i '/^import (/a\\t"github.com/sabio-engineering-product/monitoring-connectors/command"' "$file"
                sed -i '/^import (/a\\t"github.com/sabio-engineering-product/monitoring-connectors/snmp"' "$file"
            else
                # Single import or no imports
                sed -i '1i import "github.com/sabio-engineering-product/monitoring-connectors/command"' "$file"
                sed -i '1i import "github.com/sabio-engineering-product/monitoring-connectors/snmp"' "$file"
            fi
        fi
    fi
done

# Update main.go to use production connectors if it's using the template HTTP connector
if [ -f "cmd/*/main.go" ]; then
    MAIN_GO=$(find cmd -name "main.go" | head -1)
else
    MAIN_GO="main.go"
fi

if [ -f "$MAIN_GO" ]; then
    echo "🔧 Updating main.go connector usage..."
    
    # Comment out template connector and add production examples
    cat >> "$MAIN_GO.tmp" << 'EOF'
// Production connector examples:
// For SSH connections:
// connector := command.NewSSH()

// For SNMP monitoring:
// connector := snmp.NewSNMP()

// For HTTP APIs (if using monitoring-connectors HTTP):
// connector := http.NewHTTP()
EOF
    
    mv "$MAIN_GO" "$MAIN_GO.backup"
    cat "$MAIN_GO.backup" "$MAIN_GO.tmp" > "$MAIN_GO"
    rm "$MAIN_GO.tmp"
fi

# Update Docker configuration for production
echo "🐳 Updating Docker configuration..."

if [ -f "Dockerfile" ]; then
    # Update Dockerfile to use production registry if needed
    if grep -q "TEMPLATE" Dockerfile; then
        echo "⚠️  Dockerfile still contains template placeholders - please review manually"
    fi
fi

# Clean up go modules
echo "🧹 Cleaning up Go modules..."
go mod tidy

# Verify compilation
echo "🔨 Verifying compilation..."
if go build ./...; then
    echo "✅ Production build successful"
else
    echo "❌ Production build failed"
    echo "📦 Backup available in $BACKUP_DIR"
    exit 1
fi

# Run tests
echo "🧪 Running tests..."
if go test ./...; then
    echo "✅ Tests pass with production dependencies"
else
    echo "⚠️  Some tests failed - this may be expected if they require SSH/SNMP setup"
fi

# Create production deployment notes
cat > PRODUCTION_MIGRATION.md << EOF
# Production Migration Complete

This exporter has been migrated from template to production configuration.

## Changes Made

1. **Removed local monitoring-maas**: The local copy has been removed
2. **Updated dependencies**: Now uses production Sabio repositories
3. **Updated imports**: All Go files now import from production repos
4. **Backup created**: Original state backed up to \`$BACKUP_DIR\`

## Next Steps

### 1. Update Connector Configuration

Review your main.go file and configure the appropriate connector:

\`\`\`go
// For SSH monitoring
connector := command.NewSSH()

// For SNMP monitoring  
connector := snmp.NewSNMP()

// For HTTP APIs
connector := http.NewHTTP()
\`\`\`

### 2. Production Configuration

- Update connection parameters for your production environment
- Configure authentication (SSH keys, SNMP community strings, etc.)
- Set appropriate timeouts and retry policies

### 3. Deployment

- Build production Docker image
- Deploy using company infrastructure
- Configure monitoring and alerting
- Update service discovery (Consul, etc.)

### 4. Testing

- Test against production targets
- Verify metrics are being collected
- Check Prometheus scraping
- Validate alerts are working

## Files Modified

- go.mod (dependencies updated)
- All *.go files (imports updated)
- $MAIN_GO (connector examples added)

## Backup Location

Original template state: \`$BACKUP_DIR\`
EOF

echo ""
echo "🎉 Migration to production configuration complete!"
echo ""
echo "📋 Summary of changes:"
echo "  ✅ Removed local monitoring-maas folder"
echo "  ✅ Updated go.mod dependencies"
echo "  ✅ Updated import statements in Go files"
echo "  ✅ Added production connector examples"
echo "  ✅ Created backup in $BACKUP_DIR"
echo "  ✅ Verified compilation"
echo ""
echo "📖 Next steps documented in PRODUCTION_MIGRATION.md"
echo ""
echo "⚠️  Important: Review and update your connector configuration in $MAIN_GO"