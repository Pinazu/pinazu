# GitHub Workflows Documentation

This document explains the CI/CD pipeline structure for the Pinazu project.

## Overview

The project uses a branch-based deployment strategy with three main workflows:

1. **Pull Request Checks** - Runs tests without any deployments
2. **Docker Push & Python Publish** - Handles branch-specific deployments
3. **Tagged Release** - Handles versioned releases via git tags

## Workflow Triggers and Behavior

### 1. Pull Requests (pr-checks.yml)

**Trigger:** Pull requests to `main` or `release` branches

**Actions:**
- ✅ Run unit tests
- ✅ Run E2E tests
- ✅ Build Docker image (no push)
- ✅ Run SonarQube SAST scan
- ❌ NO uploads to Docker Hub
- ❌ NO uploads to PyPI

**Purpose:** Ensure code quality and functionality before merging.

### 2. Main Branch (docker-push.yml + python-publish.yml)

**Trigger:** Push to `main` branch

**Actions:**
- 🐳 Docker: Build and push `test` tag to Docker Hub
  - Platform: `linux/amd64` only
  - Tag: `<dockerhub-username>/pinazu:test`
- 🐍 Python: Publish to TestPyPI
  - Repository: https://test.pypi.org
  - Install: `pip install --index-url https://test.pypi.org/simple/ pinazu-py`

**Purpose:** Create test versions for development and staging environments.

### 3. Release Branch (docker-push.yml + python-publish.yml)

**Trigger:** Push to `release` branch

**Actions:**
- 🐳 Docker: Build and push production images to Docker Hub
  - Platforms: `linux/amd64`, `linux/arm64` (multi-arch)
  - Tags: `<dockerhub-username>/pinazu:latest`, `<dockerhub-username>/pinazu:stable`
- 🐍 Python: Publish to production PyPI
  - Repository: https://pypi.org
  - Install: `pip install pinazu-py`

**Purpose:** Create stable production releases.

### 4. Tagged Releases (release.yml)

**Trigger:** Push git tags matching `v*` (e.g., `v1.0.0`, `v2.1.3`)

**Actions:**
- 🐳 Docker: Build and push versioned multi-arch images
  - Platforms: `linux/amd64`, `linux/arm64`
  - Tags: `latest`, version tags (e.g., `1.0.0`, `1.0`, `1`)
- 🐍 Python: Publish to production PyPI
- 📝 Create GitHub Release with changelog

**Purpose:** Create immutable versioned releases for distribution.

## Branch Strategy

```
main (development)
  ↓
  └─ Deploys to: TestPyPI + Docker Hub (test tag)
  
release (production)
  ↓
  └─ Deploys to: PyPI + Docker Hub (latest, stable tags)

v* tags (versioned releases)
  ↓
  └─ Deploys to: PyPI + Docker Hub (versioned tags) + GitHub Release
```

## Required Secrets

Configure these secrets in GitHub repository settings:

### Docker Hub
- `DOCKERHUB_USERNAME` - Your Docker Hub username
- `DOCKERHUB_TOKEN` - Docker Hub access token

### PyPI
- `PYPI_API_TOKEN` - Production PyPI API token (for `release` branch and tags)
- `TEST_PYPI_API_TOKEN` - TestPyPI API token (for `main` branch)

### Optional
- `CODECOV_TOKEN` - Codecov integration token
- `SONAR_TOKEN` - SonarCloud integration token

## Workflow Execution Summary

| Event | Docker Build | Docker Push | PyPI Publish | Tests |
|-------|-------------|------------|--------------|-------|
| **Pull Request** | ✅ AMD64 only | ❌ No | ❌ No | ✅ Full suite |
| **Push to main** | ✅ AMD64 only | ✅ test tag | ✅ TestPyPI | ✅ Via PR |
| **Push to release** | ✅ Multi-arch | ✅ latest, stable | ✅ Production PyPI | ✅ Via PR |
| **Tag push (v*)** | ✅ Multi-arch | ✅ Versioned tags | ✅ Production PyPI | ✅ Via PR |

## Usage Examples

### Testing Changes
```bash
# Create a feature branch
git checkout -b feature/my-feature

# Make changes and commit
git add .
git commit -m "Add new feature"

# Push and create PR
git push origin feature/my-feature
# Create PR to main branch
# ✅ Tests run, no deployments
```

### Releasing to Test Environment
```bash
# Merge PR to main
git checkout main
git pull origin main
# ✅ Automatic deployment to TestPyPI and Docker Hub (test tag)
```

### Releasing to Production
```bash
# Merge main to release branch
git checkout release
git merge main
git push origin release
# ✅ Automatic deployment to PyPI and Docker Hub (latest, stable tags)
```

### Creating Versioned Release
```bash
# Tag a commit
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
# ✅ Creates versioned Docker images, PyPI package, and GitHub Release
```

## Docker Image Tags

### Test Images (main branch)
- `pinazu:test` - Latest test build from main branch

### Production Images (release branch)
- `pinazu:latest` - Latest stable release
- `pinazu:stable` - Alias for latest stable release

### Versioned Images (git tags)
- `pinazu:1.0.0` - Specific version
- `pinazu:1.0` - Minor version
- `pinazu:1` - Major version
- `pinazu:latest` - Latest tagged release