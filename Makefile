

# Setup all toolchains
.PHONY: prepare build clean d-run d-down d-restart clean-tools clean build-fe build-db build-api build-core run test

# Install development tools
prepare:
	@scripts/install-sqlc.sh

# Clean installed tools
clean-tools:
	@echo "Cleaning development tools..."
	rm -f ~/.local/bin/sqlc
	@echo "Tools removed!"

build-fe:
	@echo "Build Frontend..."
	cd web && npm run build

build-api:
	@echo "Building API..."
	@echo "Generate OpenAPI stub..."
	go run scripts/build-openapi.go
	@echo "Generating API code..."
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=oapi-codegen.yaml api/openapi.yaml
	@echo "Generating Event code..."
	go generate ./...

build-db:
	@echo "Generate DB code..."
	sqlc generate

build-core:
	@echo "Building core..."
	CGO_ENABLED=0 go build -o ./dist/pinazu ./cmd/
	sudo mv ./dist/pinazu /usr/local/bin/pinazu
	rm -rf ./dist

# Build the application
build: build-fe build-db build-api build-core

# Run the application
run:
	@echo "Starting application..."
	go run github.com/air-verse/air@latest -c .air.toml

# Build and run the application
build-run:
	@echo "Building and running application..."
	make build && make run

# Clean the build
clean:
	@echo "Cleaning build..."
	rm -rf ./dist
	rm -rf ./bin
	rm -f ~/usr/local/bin/pinazu
	@echo "Build cleaned!"

# Run the development environment
d-up: build
	@echo "Starting development environment..."
	docker compose up -d

# Stop the development environment
d-down:
	@echo "Stopping development environment..."
	docker compose down -v

# Restart the development environment
d-restart: d-down d-up

# Test - Run comprehensive test suite (unit + e2e)
test:
	@echo "Running comprehensive test suite..."
	@scripts/test-all.sh