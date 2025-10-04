#!/bin/bash

# test-all.sh - Comprehensive test script for Pinazu Core
# Runs unit tests and e2e integration tests

set -e

echo "🧪 Starting comprehensive test suite..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Cleanup function
cleanup() {
    print_status "Cleaning up test environment..."
    # Stop the Pinazu server if it's running
    if [ ! -z "$PINAZU_PID" ] && kill -0 "$PINAZU_PID" 2>/dev/null; then
        print_status "Stopping Pinazu server (PID: $PINAZU_PID)..."
        kill "$PINAZU_PID" && wait "$PINAZU_PID" 2>/dev/null || true
    fi
    docker compose down -v > /dev/null 2>&1 || true
}

# Set up cleanup trap
trap cleanup EXIT

print_status "Step 1: Setting up test environment..."
print_status "Stopping any existing containers..."
docker compose down -v

print_status "Starting fresh test environment..."
docker compose up -d

# Wait for services to be ready
print_status "Waiting for services to be ready..."
sleep 3

# Check if services are healthy
print_status "Checking service health..."
if ! docker compose ps | grep -q "Up"; then
    print_error "Some services failed to start properly"
    docker compose logs
    exit 1
fi

print_status "Step 2: Building and starting Pinazu server..."
echo "================================================="

# Build the Pinazu server
print_status "Building Pinazu server..."
make build-db build-api build-core

# Load environment variables
if [ -f .env ]; then
    set -a
    source .env
    set +a
fi

# Start the Pinazu server in the background
print_status "Starting Pinazu server..."
pinazu serve all -c configs/config.yaml &
PINAZU_PID=$!

# Wait for the server to be ready
print_status "Waiting for Pinazu server to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:8080/swagger/openapi.yaml > /dev/null 2>&1; then
        print_status "Pinazu server is ready!"
        break
    fi
    if [ $i -eq 30 ]; then
        print_error "Pinazu server failed to start within 30 seconds"
        exit 1
    fi
    sleep 1
done

print_status "Step 3: Running unit tests..."
echo "================================================="

# Run tests and capture the exit code
go test -v ./...
TEST_EXIT_CODE=$?

if [ $TEST_EXIT_CODE -eq 0 ]; then
    print_status "✅ Unit tests passed!"
else
    print_warning "⚠️  Some unit tests failed (exit code: $TEST_EXIT_CODE)"
    print_status "Continuing with e2e tests..."
fi

print_status "Step 4: Running e2e integration tests..."
echo "================================================="

# Check if e2e directory exists
if [ ! -d "e2e" ]; then
    print_error "e2e directory not found, skipping integration tests"
    exit 1
fi

cd e2e

# Install npm dependencies if needed
if [ ! -d "node_modules" ]; then
    print_status "Installing e2e test dependencies..."
    npm install
fi

# Run e2e tests
print_status "Running Playwright e2e tests..."
npm run test
E2E_EXIT_CODE=$?

if [ $E2E_EXIT_CODE -eq 0 ]; then
    print_status "✅ E2E tests passed!"
else
    print_error "❌ E2E tests failed!"
fi

cd ..

# Final summary
echo "================================================="
print_status "🎯 Test Suite Summary:"

if [ $TEST_EXIT_CODE -eq 0 ]; then
    print_status "  ✅ Unit tests: PASSED"
else
    print_warning "  ⚠️  Unit tests: FAILED (exit code: $TEST_EXIT_CODE)"
fi

if [ $E2E_EXIT_CODE -eq 0 ]; then
    print_status "  ✅ E2E tests: PASSED"
else
    print_warning "  ❌ E2E tests: FAILED (exit code: $E2E_EXIT_CODE)"
fi

echo "================================================="

# Exit with failure if any tests failed
if [ $TEST_EXIT_CODE -ne 0 ] || [ $E2E_EXIT_CODE -ne 0 ]; then
    print_error "Some tests failed. Please review the output above."
    exit 1
else
    print_status "🎉 All tests completed successfully!"
    print_status "Cleaning up containers after successful test run..."
    docker compose down
fi