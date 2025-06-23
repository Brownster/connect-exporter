#!/bin/bash

# Validation script for template setup
# Usage: ./scripts/validate-template.sh [exporter-directory]

set -e

EXPORTER_DIR="${1:-.}"

echo "🔍 Validating template setup in: $EXPORTER_DIR"

# Check if we're in the right directory
if [ ! -f "$EXPORTER_DIR/go.mod" ]; then
    echo "❌ Error: go.mod not found in $EXPORTER_DIR"
    echo "   Make sure you're in the exporter directory or provide the correct path"
    exit 1
fi

# Function to check if file exists and is not empty
check_file() {
    local file="$1"
    local description="$2"
    
    if [ ! -f "$file" ]; then
        echo "❌ Missing: $description ($file)"
        return 1
    elif [ ! -s "$file" ]; then
        echo "⚠️  Empty: $description ($file)"
        return 1
    else
        echo "✅ Found: $description ($file)"
        return 0
    fi
}

# Function to check for template placeholders
check_placeholders() {
    local file="$1"
    local description="$2"
    
    if [ ! -f "$file" ]; then
        return 0  # Skip if file doesn't exist
    fi
    
    local placeholders=$(grep -E "TEMPLATE_(ORG|REPO_NAME|EXPORTER_NAME|GITHUB_ORG)|github\.com/TEMPLATE_" "$file" 2>/dev/null || true)
    
    if [ -n "$placeholders" ]; then
        echo "⚠️  Template placeholders found in $description ($file):"
        echo "$placeholders" | sort | uniq | sed 's/^/   - /'
        return 1
    else
        echo "✅ No template placeholders in $description"
        return 0
    fi
}

# Check required files
echo ""
echo "📁 Checking required files..."

check_file "$EXPORTER_DIR/go.mod" "Go module file"
check_file "$EXPORTER_DIR/main.go" "Main application file" || check_file "$EXPORTER_DIR/cmd/*/main.go" "Main application file in cmd/"

# Find the main.go file
MAIN_GO=$(find "$EXPORTER_DIR" -name "main.go" -path "*/cmd/*" | head -1)
if [ -z "$MAIN_GO" ]; then
    MAIN_GO="$EXPORTER_DIR/main.go"
fi

# Check exporter files
EXPORTER_GO=$(find "$EXPORTER_DIR/collectors" -name "*_exporter.go" 2>/dev/null | head -1)
if [ -n "$EXPORTER_GO" ]; then
    check_file "$EXPORTER_GO" "Exporter constructor"
else
    echo "❌ No exporter.go file found in collectors/"
fi

# Check for at least one collector
COLLECTOR_COUNT=$(find "$EXPORTER_DIR/collectors" -name "*_collector.go" 2>/dev/null | wc -l)
if [ "$COLLECTOR_COUNT" -gt 0 ]; then
    echo "✅ Found $COLLECTOR_COUNT collector(s)"
else
    echo "❌ No collectors found in collectors/"
fi

# Check configuration
check_file "$EXPORTER_DIR/config.yml" "Configuration file" || check_file "$EXPORTER_DIR/config.example.yml" "Example configuration"

# Check Docker files
check_file "$EXPORTER_DIR/Dockerfile" "Docker build file"
check_file "$EXPORTER_DIR/docker-compose.yml" "Docker Compose file"

# Check documentation
echo ""
echo "📚 Checking documentation..."

check_file "$EXPORTER_DIR/README.md" "README file"

# Check for template placeholders
echo ""
echo "🔧 Checking for unresolved template placeholders..."

PLACEHOLDER_ISSUES=0

if [ -f "$MAIN_GO" ]; then
    check_placeholders "$MAIN_GO" "main.go" || ((PLACEHOLDER_ISSUES++))
fi

if [ -n "$EXPORTER_GO" ]; then
    check_placeholders "$EXPORTER_GO" "exporter.go" || ((PLACEHOLDER_ISSUES++))
fi

check_placeholders "$EXPORTER_DIR/go.mod" "go.mod" || ((PLACEHOLDER_ISSUES++))
check_placeholders "$EXPORTER_DIR/README.md" "README.md" || ((PLACEHOLDER_ISSUES++))
check_placeholders "$EXPORTER_DIR/Dockerfile" "Dockerfile" || ((PLACEHOLDER_ISSUES++))

# Check Go compilation
echo ""
echo "🔨 Checking Go compilation..."

cd "$EXPORTER_DIR"

if go mod tidy; then
    echo "✅ go mod tidy successful"
else
    echo "❌ go mod tidy failed"
    exit 1
fi

if go build ./...; then
    echo "✅ Go compilation successful"
else
    echo "❌ Go compilation failed"
    exit 1
fi

# Check tests
echo ""
echo "🧪 Checking tests..."

if go test ./... -v; then
    echo "✅ Tests pass"
else
    echo "❌ Tests failed"
    exit 1
fi

# Final summary
echo ""
echo "📊 Validation Summary:"

if [ "$PLACEHOLDER_ISSUES" -eq 0 ]; then
    echo "✅ No unresolved template placeholders"
else
    echo "⚠️  $PLACEHOLDER_ISSUES files have unresolved template placeholders"
    echo "   Run the setup script to resolve them"
fi

echo ""
echo "🎯 Next steps:"
echo "1. Resolve any template placeholders shown above"
echo "2. Customize your collectors in collectors/"
echo "3. Update configuration in config.yml"
echo "4. Add tests for your specific logic"
echo "5. Update README.md with your exporter details"

echo ""
if [ "$PLACEHOLDER_ISSUES" -eq 0 ]; then
    echo "🎉 Template validation completed successfully!"
    exit 0
else
    echo "⚠️  Template validation completed with warnings"
    exit 1
fi