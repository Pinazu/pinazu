# PyPI Publishing Guide

This guide explains how to publish the `pinazu-py` package to PyPI using GitHub Actions.

## Understanding the Publishing Flow

### When Packages Are Published

| Event | PyPI (Production) | Test PyPI (Testing) |
|-------|------------------|---------------------|
| Pull Request | ❌ Not published | ❌ Not published |
| Push to `main` | ❌ Not published | ✅ Published (optional) |
| Git Tag (e.g., `v0.6.2`) | ✅ Published | ❌ Not published |

**Key Point:** The package only appears on **PyPI.org** when you create a **git tag**, not when you push to main!

## First-Time Setup

### 1. Create PyPI Account

1. Go to https://pypi.org/account/register/
2. Create an account
3. Verify your email address
4. Enable 2FA (recommended)

### 2. Generate PyPI API Token

1. Log in to PyPI.org
2. Go to **Account Settings**
3. Scroll to **API tokens**
4. Click **"Add API token"**
5. Set:
   - **Token name**: `GitHub Actions - Pinazu`
   - **Scope**: Select "Entire account" (initially)
   - After first upload, you can scope it to just `pinazu-py`
6. **Copy the token immediately** (you won't see it again!)
   - Token format: `pypi-AgEIcHlwaS5vcmc...` (very long)
   - ⚠️ **Important**: Copy the entire token cleanly without line breaks or extra characters

### 3. Add Token to GitHub Secrets

1. Go to your GitHub repository
2. Click **Settings** → **Secrets and variables** → **Actions**
3. Click **"New repository secret"**
4. Set:
   - **Name**: `PYPI_API_TOKEN`
   - **Secret**: Paste the ENTIRE token
   - ⚠️ **Make sure there are no line breaks or extra spaces!**
5. Click **"Add secret"**

## Publishing Your First Release

### Step 1: Update Version

Edit `python/pinazu-py/pyproject.toml`:

```toml
[project]
name = "pinazu-py"
version = "0.6.2"  # ← Increment this
```

### Step 2: Commit and Push

```bash
git add python/pinazu-py/pyproject.toml
git commit -m "Bump version to 0.6.2"
git push origin main
```

### Step 3: Create and Push Git Tag

```bash
# Create tag
git tag v0.6.2

# Push tag to GitHub
git push origin v0.6.2
```

### Step 4: Watch the Workflow

1. Go to your repository's **Actions** tab
2. You'll see a workflow running for the tag `v0.6.2`
3. Click on it to watch progress
4. The `build-python` job will:
   - Build the package
   - Publish to PyPI.org
   - Show success message: "✅ Python package published successfully to PyPI"

### Step 5: Verify on PyPI

1. Go to https://pypi.org/project/pinazu-py/
2. Your package should now be live!
3. Anyone can install it: `pip install pinazu-py`

## Test PyPI (Optional Testing)

### Why Use Test PyPI?

Test PyPI is a separate instance for testing package uploads without affecting production PyPI.

### Setup Test PyPI (Optional)

1. Create account at https://test.pypi.org/account/register/
2. Generate API token (same process as PyPI)
3. Add to GitHub Secrets as `TEST_PYPI_API_TOKEN`

### How It Works

- **Push to main** → Publishes to Test PyPI
- **Create tag** → Publishes to production PyPI

### Test PyPI Issues

If you see errors like:

```
UnicodeEncodeError: 'latin-1' codec can't encode character '\u2028'
```

This means the `TEST_PYPI_API_TOKEN` has special characters. To fix:

1. Delete the old secret from GitHub
2. Generate a **new** token on test.pypi.org
3. Copy it carefully (no line breaks!)
4. Add it again to GitHub Secrets

**Or simply skip Test PyPI**: It's optional! Just don't set `TEST_PYPI_API_TOKEN` and the workflow will skip that step.

## Troubleshooting

### Issue: "Package not appearing on PyPI"

**Cause**: You pushed to `main` but didn't create a git tag.

**Solution**: PyPI publishing only happens on git tags. Create and push a tag:

```bash
git tag v0.6.2
git push origin v0.6.2
```

### Issue: "403 Forbidden" when publishing

**Causes:**
1. Token is invalid or expired
2. Token doesn't have permission for this project
3. Someone else owns the package name

**Solutions:**
1. Generate a new token on PyPI.org
2. Set token scope to "Entire account" or scope to `pinazu-py` after first upload
3. Choose a different package name if taken

### Issue: "Version already exists"

**Cause**: You can't re-upload the same version to PyPI.

**Solution**: Increment the version in `pyproject.toml`:

```toml
version = "0.6.3"  # Increment from 0.6.2
```

### Issue: Unicode errors with Test PyPI

**Cause**: Token was copied with special characters or line breaks.

**Solutions:**
1. **Skip Test PyPI**: Don't set `TEST_PYPI_API_TOKEN` secret (recommended for most projects)
2. **Fix token**: Delete and recreate the secret, ensuring clean copy
3. The workflow now handles this gracefully with `continue-on-error: true`

### Issue: "Invalid package name"

**Cause**: Package name `pinazu-py` must be unique on PyPI.

**Check**: Visit https://pypi.org/project/pinazu-py/ to see if it exists.

**Solutions:**
- If you own it: You're good to go!
- If someone else owns it: Choose a different name in `pyproject.toml`

## Version Management

### Semantic Versioning

Follow semantic versioning:

```
MAJOR.MINOR.PATCH

Examples:
- 0.6.1 → 0.6.2  (bug fix)
- 0.6.2 → 0.7.0  (new feature)
- 0.7.0 → 1.0.0  (breaking change)
```

### Release Workflow

1. **Make changes**
   ```bash
   git checkout -b feature/new-feature
   # Make your changes
   git commit -am "Add new feature"
   git push origin feature/new-feature
   ```

2. **Create pull request**
   - Tests will run automatically
   - Package will be built but not published

3. **Merge to main**
   ```bash
   # After PR approval
   git checkout main
   git pull origin main
   ```

4. **Update version**
   ```bash
   # Edit python/pinazu-py/pyproject.toml
   version = "0.6.2"  # New version
   
   git commit -am "Bump version to 0.6.2"
   git push origin main
   ```

5. **Create release**
   ```bash
   git tag v0.6.2
   git push origin v0.6.2
   ```

6. **Verify**
   - Check GitHub Actions for success
   - Visit https://pypi.org/project/pinazu-py/
   - Test: `pip install pinazu-py==0.6.2`

## Advanced: GitHub Releases

You can also create releases through GitHub UI:

1. Go to your repository
2. Click **"Releases"** → **"Create a new release"**
3. Click **"Choose a tag"** → Type `v0.6.2` → **"Create new tag"**
4. Set release title: `v0.6.2`
5. Add release notes (what's new, bug fixes, etc.)
6. Click **"Publish release"**

This automatically creates the tag and triggers the workflow!

## Checking Package Status

### View on PyPI

```bash
# Check if package exists
curl -s https://pypi.org/pypi/pinazu-py/json | jq '.info.version'

# View all versions
curl -s https://pypi.org/pypi/pinazu-py/json | jq '.releases | keys'
```

### Install and Test

```bash
# Install latest
pip install pinazu-py

# Install specific version
pip install pinazu-py==0.6.2

# Upgrade to latest
pip install --upgrade pinazu-py
```

## FAQ

**Q: Do I need to manually create the project on PyPI first?**

A: No! The first time you push a tag and the workflow runs, it will automatically create the project on PyPI.

**Q: Can I delete a version from PyPI?**

A: No, PyPI doesn't allow deleting versions for security/reproducibility. You must increment the version number.

**Q: Should I use Test PyPI?**

A: It's optional. For most projects, testing with pull requests is sufficient. Test PyPI is useful for:
- Testing the upload process before production
- Verifying package metadata
- Testing installation from PyPI

**Q: What if someone already took the name `pinazu-py`?**

A: You'll get a 403 error. You'll need to:
1. Choose a different name in `pyproject.toml`
2. Check availability: https://pypi.org/project/YOUR-NAME-HERE/
3. Update the name and try again

**Q: Can I automate version bumping?**

A: Yes! You can use tools like:
- `bump2version` / `bumpversion`
- `poetry version`
- GitHub Actions with automatic version detection

**Q: How do I unpublish from Test PyPI?**

A: Test PyPI has relaxed deletion rules. You can delete entire projects from your account settings if needed.

## Support

If you encounter issues:

1. Check the GitHub Actions logs for detailed error messages
2. Verify your token is correct and has permissions
3. Ensure the version number is incremented
4. Check https://pypi.org/project/pinazu-py/ for existing versions
5. Review this guide for troubleshooting steps

## Quick Reference

```bash
# Publish new version to PyPI
git add python/pinazu-py/pyproject.toml  # After updating version
git commit -m "Bump version to X.Y.Z"
git push origin main
git tag vX.Y.Z
git push origin vX.Y.Z

# View workflow
# Go to: https://github.com/YOUR-ORG/pinazu/actions

# Verify on PyPI
# Go to: https://pypi.org/project/pinazu-py/

# Test installation
pip install pinazu-py==X.Y.Z
