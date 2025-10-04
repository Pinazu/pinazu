"""Pytest configuration for pinazu-py client tests."""

import uuid
import pytest
import httpx
from uuid import UUID

from pinazu import Client, AsyncClient


@pytest.fixture
def base_url():
    """Base URL for testing."""
    return "http://localhost:8080"


@pytest.fixture
def client(base_url):
    """Sync client instance for testing."""
    return Client(base_url=base_url)


@pytest.fixture
def async_client(base_url):
    """Async client instance for testing."""
    return AsyncClient(base_url=base_url)


@pytest.fixture
def test_agent_data():
    """Test agent data for creating agents."""
    return {
        "name": "Test Agent for Pytest",
        "description": "A test agent created via pytest tests",
        "specs": """
model:
  provider: "anthropic"
  model_id: "claude-3-sonnet"
  max_tokens: 4096
  temperature: 0.7

system: |
  You are a helpful AI assistant for testing purposes.
  Respond clearly and concisely to user queries.

tools:
  - name: "calculator"
    description: "Perform basic arithmetic operations"

parameters:
  thinking_enabled: true
  debug_mode: false
        """,
    }


@pytest.fixture
def test_user_data():
    """Test user data for creating users."""
    return {
        "name": "Test User",
        "email": f"test+{uuid.uuid4().hex[:8]}@example.com",
        "password_hash": "test_password_hash_123",
        "provider_name": "local",
        "additional_info": {"test": True},
    }


@pytest.fixture
def test_role_data():
    """Test role data for creating roles."""
    return {
        "name": f"Test Role {uuid.uuid4().hex[:8]}",
        "description": "A test role for pytest",
        "is_system_role": False,
    }


@pytest.fixture
def test_permission_data():
    """Test permission data for creating permissions."""
    return {
        "name": f"Test Permission {uuid.uuid4().hex[:8]}",
        "description": "A test permission for pytest",
        "content": {
            "action": "test_action",
            "resource": "test_resource",
            "conditions": [],
        },
    }


@pytest.fixture
def test_flow_data():
    """Test flow data for creating flows."""
    return {
        "name": f"Test Flow {uuid.uuid4().hex[:8]}",
        "description": "A test flow created via pytest tests",
        "engine": "python",
        "entrypoint": "main.py",
        "code_location": "s3://test-bucket/flows/test_flow.py",
        "parameters_schema": {
            "type": "object",
            "properties": {
                "input_text": {
                    "type": "string",
                    "description": "Input text to process",
                }
            },
            "required": ["input_text"],
        },
        "tags": ["test", "api", "automation"],
    }


@pytest.fixture
def test_tool_data():
    """Test tool data for creating tools."""
    return {
        "name": f"Test Tool {uuid.uuid4().hex[:8]}",
        "description": "A test tool for pytest",
        "config": {
            "type": "standalone",
            "command": "echo 'hello world'",
            "timeout": 30,
        },
    }


@pytest.fixture
def mock_responses():
    """Mock HTTP responses for testing."""

    class MockResponse:
        def __init__(self, json_data, status_code=200):
            self.json_data = json_data
            self.status_code = status_code
            self.text = str(json_data)
            self.url = httpx.URL("http://localhost:8080/test")

        def json(self):
            return self.json_data

        @property
        def is_success(self):
            return 200 <= self.status_code < 300

        def raise_for_status(self):
            if not self.is_success:
                raise httpx.HTTPStatusError(
                    message=f"HTTP {self.status_code}",
                    request=None,
                    response=self,
                )

    return MockResponse


@pytest.fixture
def sample_uuid():
    """A sample UUID for testing."""
    return UUID("550e8400-c95b-4444-6666-446655440000")


@pytest.fixture
def invalid_uuid():
    """An invalid UUID string for testing."""
    return "invalid-uuid-format"


@pytest.fixture
def nil_uuid():
    """A nil UUID for testing."""
    return UUID("00000000-0000-0000-0000-000000000000")
