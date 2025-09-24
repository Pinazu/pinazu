# Makefile for pinazu-py development and testing

.PHONY: help install install-dev test test-unit test-integration test-all lint format type-check clean build upload

# Default target
help:
	@echo "Available targets:"
	@echo "  install          Install package dependencies"
	@echo "  install-dev      Install development dependencies"
	@echo "  test             Run unit tests only"
	@echo "  test-unit        Run unit tests only"
	@echo "  test-integration Run integration tests (requires running server)"
	@echo "  test-all         Run all tests including integration"
	@echo "  test-cov         Run tests with coverage report"
	@echo "  lint             Run code linting with ruff"
	@echo "  format           Format code with ruff"
	@echo "  type-check       Run type checking with mypy"
	@echo "  clean            Clean build artifacts"
	@echo "  build            Build package"
	@echo "  upload           Upload package to PyPI"
	@echo "  tox              Run tests with tox across Python versions"

# Installation targets
install:
	pip install -e .

install-dev:
	pip install -e ".[dev]"

install-test:
	pip install -e ".[test]"

# Testing targets
test: test-unit

test-unit:
	pytest tests/ -m "not integration" -v

test-integration:
	@echo "Make sure Pinazu server is running before running integration tests"
	@echo "Run: pinazu serve all"
	pytest tests/ -m "integration" -v

test-all:
	pytest tests/ -v

test-cov:
	pytest tests/ -m "not integration" --cov=pinazu --cov-report=html --cov-report=term --cov-report=xml

test-parallel:
	pytest tests/ -m "not integration" -n auto

# Code quality targets
lint:
	ruff check src/ tests/
	ruff format --check src/ tests/

format:
	ruff format src/ tests/
	ruff check --fix src/ tests/

type-check:
	mypy src/pinazu

# Tox targets
tox:
	tox

tox-unit:
	tox -e py310,py311,py312

tox-integration:
	tox -e integration

# Build and distribution targets
clean:
	rm -rf build/
	rm -rf dist/
	rm -rf *.egg-info/
	rm -rf .pytest_cache/
	rm -rf .coverage
	rm -rf htmlcov/
	rm -rf .mypy_cache/
	rm -rf .tox/
	find . -type d -name __pycache__ -delete
	find . -type f -name "*.pyc" -delete

build: clean
	python -m build

upload: build
	python -m twine upload dist/*

upload-test: build
	python -m twine upload --repository testpypi dist/*

# Development targets
pre-commit:
	pre-commit run --all-files

setup-pre-commit:
	pre-commit install

# Documentation targets (for future use)
docs:
	@echo "Documentation generation not yet implemented"

# Quick development workflow
dev-setup: install-dev setup-pre-commit
	@echo "Development environment setup complete"

dev-test: format lint type-check test-unit
	@echo "All development checks passed"

# CI targets
ci-test: install-test test-cov lint type-check
	@echo "CI tests completed"

# Integration test setup helper
setup-integration:
	@echo "Setting up integration test environment..."
	@echo "1. Start Docker services: docker-compose up -d"
	@echo "2. Start Pinazu server: pinazu serve all"
	@echo "3. Wait for services to be ready"
	@echo "4. Run: make test-integration"