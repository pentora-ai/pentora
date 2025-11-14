#!/bin/bash
# rebrand.sh - Automated Pentora → Vulntor rebranding script
#
# This script performs complete rebranding of Pentora OSS to Vulntor.
# It updates Go module paths, import statements, CLI names, environment
# variables, documentation, and all user-facing content.
#
# Usage: ./scripts/rebrand.sh
#
# IMPORTANT: This is a destructive operation. Ensure you're on a feature branch.

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Verify we're in the right directory
if [ ! -f "go.mod" ] || [ ! -d "cmd" ] || [ ! -d "pkg" ]; then
    error "Not in pentora repository root. Please run from repository root."
    exit 1
fi

# Verify we're on a feature branch
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" = "main" ]; then
    error "Cannot run rebrand on main branch. Please create a feature branch first."
    exit 1
fi

info "Starting Pentora → Vulntor rebrand..."
info "Current branch: $CURRENT_BRANCH"
echo ""

# Phase 1: Go module and imports
info "Phase 1: Updating Go module path..."
sed -i '' 's|github.com/pentora-ai/pentora|github.com/vulntor/vulntor|g' go.mod
success "Updated go.mod"

info "Phase 1: Updating import paths in all .go files..."
find . -name "*.go" -type f \
    -not -path "./vendor/*" \
    -not -path "./node_modules/*" \
    -not -path "./.git/*" \
    -exec sed -i '' 's|github.com/pentora-ai/pentora|github.com/vulntor/vulntor|g' {} +
success "Updated import paths in Go files"

# Phase 2: Environment variables
info "Phase 2: Updating environment variables (PENTORA_* → VULNTOR_*)..."
find . \( -name "*.go" -o -name "*.md" -o -name "*.yaml" -o -name "*.yml" \) -type f \
    -not -path "./vendor/*" \
    -not -path "./node_modules/*" \
    -not -path "./.git/*" \
    -not -path "./.claude/*" \
    -exec sed -i '' 's/PENTORA_/VULNTOR_/g' {} +
success "Updated environment variables"

# Phase 3: CLI executable and brand name in code/docs
info "Phase 3: Updating CLI executable name (pentora → vulntor)..."
find . \( -name "*.go" -o -name "*.md" -o -name "*.yaml" -o -name "*.yml" -o -name "*.sh" \) -type f \
    -not -path "./vendor/*" \
    -not -path "./node_modules/*" \
    -not -path "./.git/*" \
    -not -path "./.claude/*" \
    -not -path "./scripts/rebrand.sh" \
    -exec sed -i '' 's/pentora/vulntor/g' {} +
success "Updated CLI executable references (lowercase)"

info "Phase 3: Updating brand name (Pentora → Vulntor)..."
find . \( -name "*.go" -o -name "*.md" -o -name "*.yaml" -o -name "*.yml" -o -name "*.html" \) -type f \
    -not -path "./vendor/*" \
    -not -path "./node_modules/*" \
    -not -path "./.git/*" \
    -not -path "./.claude/*" \
    -exec sed -i '' 's/Pentora/Vulntor/g' {} +
success "Updated brand name (capitalized)"

# Phase 4: Makefile
info "Phase 4: Updating Makefile..."
sed -i '' 's/BIN_NAME=pentora/BIN_NAME=vulntor/g' Makefile
sed -i '' 's/CLI_EXEC=pentora/CLI_EXEC=vulntor/g' Makefile
sed -i '' 's/VULNTOR_CLI_EXECUTABLE/VULNTOR_CLI_EXECUTABLE/g' Makefile
success "Updated Makefile"

# Phase 5: Storage paths (pkg/storage/config.go)
info "Phase 5: Updating storage workspace paths..."
if [ -f "pkg/storage/config.go" ]; then
    # macOS path
    sed -i '' 's|Application Support/Pentora|Application Support/Vulntor|g' pkg/storage/config.go
    # Linux path
    sed -i '' 's|\.local/share/pentora|.local/share/vulntor|g' pkg/storage/config.go
    # Windows path (if present)
    sed -i '' 's|AppData\\Pentora|AppData\\Vulntor|g' pkg/storage/config.go
    success "Updated storage paths"
else
    warning "pkg/storage/config.go not found, skipping storage path updates"
fi

# Phase 6: Directory rename (cmd/pentora → cmd/vulntor)
info "Phase 6: Renaming cmd/pentora directory to cmd/vulntor..."
if [ -d "cmd/pentora" ]; then
    git mv cmd/pentora cmd/vulntor
    success "Renamed cmd/pentora → cmd/vulntor"
else
    warning "cmd/pentora not found, skipping directory rename"
fi

# Phase 7: Format and verify
info "Phase 7: Running gofmt and go mod tidy..."
gofmt -w .
go mod tidy
success "Code formatted and go.mod cleaned up"

echo ""
info "Rebrand complete! Running verification..."
echo ""

# Verification
info "Building to verify changes..."
if go build ./...; then
    success "Build successful!"
else
    error "Build failed! Please review changes."
    exit 1
fi

echo ""
success "=========================================="
success "Rebrand completed successfully!"
success "=========================================="
echo ""
info "Next steps:"
echo "  1. Review changes: git diff"
echo "  2. Run tests: make test"
echo "  3. Run validation: make validate"
echo "  4. Commit changes: git add -A && git commit"
echo ""
warning "Note: Some manual verification may be needed for:"
echo "  - UI files (if any)"
echo "  - Plugin YAML metadata"
echo "  - External references"
echo ""
