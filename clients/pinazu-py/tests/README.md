# Pinazu-Py Test Suite

This directory contains comprehensive tests for the pinazu-py Python client library.

## Test Structure

### Unit Tests
- `test_agents_api.py` - Tests for Agent API methods (create, read, update, delete, permissions)
- `test_flows_api.py` - Tests for Flow API methods (create, execute, monitor)
- `test_tasks_api.py` - Tests for Task API methods (create, execute, monitor)
- `test_additional_apis.py` - Tests for Users, Roles, Permissions, Tools, Threads, Messages APIs
- `test_error_handling.py` - Tests for error handling, HTTP errors, and client initialization

### Integration Tests
- `test_integration_e2e.py` - End-to-end integration tests requiring a running Pinazu server

### Configuration Files
- `conftest.py` - Pytest fixtures and test configuration
- `__init__.py` - Makes tests directory a Python package
- `pytest.ini` - Pytest configuration
- `tox.ini` - Tox configuration for testing across Python versions

## Running Tests

### Prerequisites
Install test dependencies:
```bash
pip install pytest pytest-asyncio pytest-mock pytest-cov
```

### Unit Tests Only
Run unit tests (no external dependencies required):
```bash
# Run all unit tests
pytest tests/ -m "not integration"

# Run specific test file
pytest tests/test_agents_api.py

# Run with coverage
pytest tests/ -m "not integration" --cov=pinazu --cov-report=html
```

### Integration Tests
Run integration tests (requires running Pinazu server):
```bash
# Start Pinazu server first (in separate terminal)
pinazu serve all

# Then run integration tests
pytest tests/ -m integration
```

### All Tests
Run all tests including integration:
```bash
pytest tests/
```

### Using Tox
Run tests across multiple Python versions:
```bash
# Install tox
pip install tox

# Run unit tests on all Python versions
tox

# Run specific environment
tox -e py311

# Run integration tests
tox -e integration

# Run linting
tox -e lint
```

## Test Markers

Tests are marked with the following markers:

- `unit` - Unit tests that don't require external services (default)
- `integration` - Integration tests that require a running Pinazu server
- `slow` - Tests that take a long time to run
- `network` - Tests that require network connectivity

Filter tests by marker:
```bash
pytest -m "unit"           # Unit tests only
pytest -m "integration"    # Integration tests only
pytest -m "not slow"       # Exclude slow tests
```

## Test Features

### Comprehensive API Coverage
- **Agents API**: Create, read, update, delete agents; manage permissions
- **Flows API**: Create, execute, monitor workflows
- **Tasks API**: Create, execute, monitor tasks
- **Users API**: User management and authentication
- **Roles & Permissions**: Authorization and access control
- **Tools API**: Tool definitions and configurations
- **Threads & Messages**: Conversation management

### Async Support
All API methods are tested with both synchronous and asynchronous clients:
```python
# Sync client tests
def test_create_agent(client):
    agent = client.create_agent(name="Test Agent")

# Async client tests  
@pytest.mark.asyncio
async def test_async_create_agent(async_client):
    agent = await async_client.create_agent(name="Test Agent")
```

### Error Handling
Comprehensive error handling tests covering:
- HTTP status codes (400, 401, 403, 404, 409, 500, etc.)
- Network errors (connection, timeout)
- Validation errors
- JSON parsing errors
- Client initialization errors

### Mock Responses
Tests use mock HTTP responses to avoid external dependencies:
```python
def test_api_method(client, mock_responses):
    mock_response = mock_responses({"id": "123", "name": "test"}, 201)
    with patch.object(client, 'post', return_value=mock_response):
        result = client.create_resource(name="test")
```

### Fixtures
Reusable test fixtures for common test data:
- `client` - Synchronous HTTP client
- `async_client` - Asynchronous HTTP client  
- `test_agent_data` - Sample agent data
- `test_flow_data` - Sample flow data
- `sample_uuid` - Sample UUID for testing
- `mock_responses` - Mock HTTP response factory

## Integration Test Requirements

Integration tests require:
1. Running Pinazu server (typically `pinazu serve all`)
2. Database connectivity (PostgreSQL)
3. Message broker (NATS)
4. Network connectivity

Set up integration environment:
```bash
# Start dependencies with Docker Compose
docker-compose up -d

# Start Pinazu server
pinazu serve all -c configs/config.yaml

# Run integration tests
pytest tests/ -m integration
```

## CI/CD Integration

The test suite is designed for CI/CD pipelines:

```yaml
# GitHub Actions example
- name: Run unit tests
  run: pytest tests/ -m "not integration" --cov=pinazu
  
- name: Run integration tests
  run: |
    docker-compose up -d
    pinazu serve all &
    sleep 10
    pytest tests/ -m integration
```

## Writing New Tests

### Unit Test Template
```python
def test_new_api_method(client, mock_responses):
    """Test description."""
    expected_result = ExpectedModel(id=UUID("..."), name="test")
    mock_response = mock_responses(expected_result.model_dump(mode="json"), 201)
    
    with patch.object(client, 'post', return_value=mock_response):
        result = client.new_api_method(name="test")
        
        assert result.name == "test"
        assert result.id is not None
```

### Integration Test Template
```python
@pytest.mark.integration
def test_new_integration_workflow(client):
    """Test complete workflow integration."""
    # Create resource
    resource = client.create_resource(name="integration test")
    
    try:
        # Test operations
        assert resource.name == "integration test"
        
        # Update resource
        updated = client.update_resource(resource.id, name="updated")
        assert updated.name == "updated"
        
    finally:
        # Cleanup
        client.delete_resource(resource.id)
```

### Async Test Template
```python
@pytest.mark.asyncio
async def test_async_method(async_client, mock_responses):
    """Test async client method."""
    mock_response = mock_responses({"id": "123"}, 201)
    
    with patch.object(async_client, 'post', return_value=mock_response):
        result = await async_client.async_method()
        assert result.id == "123"
```

## Best Practices

1. **Use descriptive test names** that explain what is being tested
2. **Test both success and error scenarios** for each API method
3. **Use fixtures** for common test data to avoid duplication
4. **Mock external dependencies** in unit tests
5. **Clean up resources** in integration tests using try/finally blocks
6. **Test async variants** of API methods separately
7. **Use appropriate markers** to categorize tests
8. **Include edge cases** like invalid UUIDs, empty responses, etc.
9. **Verify request parameters** in addition to response validation
10. **Test error message content** not just status codes