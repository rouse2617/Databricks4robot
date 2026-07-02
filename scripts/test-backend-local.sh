#!/bin/bash
# Local backend test runner — quick feedback before commit

set -e

echo "🧪 Running Backend Tests..."
cd "$(dirname "$0")/../backend"

# Quick vet check
echo "  → Running go vet..."
go vet ./...

# Run tests (parallel, 300s timeout like CI)
echo "  → Running go test..."
go test -p 4 -timeout 300s ./...

echo ""
echo "✅ All backend tests passed!"
