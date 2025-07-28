#!/bin/bash

# E2E Test Runner for kmcp
# This script runs the end-to-end tests using the Go test library

set -e

echo "Starting kmcp e2e test suite..."

# Ensure we're in the right directory
cd "$(dirname "$0")/../.."

# Check if kind cluster exists
if ! kind get clusters | grep -q "kind"; then
    echo "Kind cluster not found. Please run setup-kind-demo.sh first."
    exit 1
fi

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "kubectl is not installed or not in PATH"
    exit 1
fi

# Check if helm is available
if ! command -v helm &> /dev/null; then
    echo "helm is not installed or not in PATH"
    exit 1
fi

# Run the tests
echo "Running e2e tests..."
go test -v ./test/e2e -timeout=30m

echo "E2E test suite completed." 