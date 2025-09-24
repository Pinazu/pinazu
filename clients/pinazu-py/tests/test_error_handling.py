"""Test suite for error handling in the pinazu-py client."""

import pytest
from unittest.mock import patch, MagicMock
import httpx
from pinazu import Client, AsyncClient, PinazuAPIError
from pinazu.api.base_client import _handle_error_response


class TestErrorHandling:
    """Test class for error handling functionality."""

    def test_pinazu_api_error_creation(self):
        """Test PinazuAPIError creation with different parameters."""
        # Basic error
        error = PinazuAPIError(404, "Not found")
        assert error.status_code == 404
        assert error.message == "Not found"
        assert error.url is None
        assert str(error) == "API Error 404: Not found"

        # Error with URL
        error_with_url = PinazuAPIError(
            500,
            "Internal error",
            "http://test.com/api",
        )
        assert error_with_url.url == "http://test.com/api"
        assert str(error_with_url) == "API Error 500: Internal error"

    def test_handle_error_response_success(self):
        """Test _handle_error_response with successful response."""
        mock_response = MagicMock()
        mock_response.is_success = True
        mock_response.status_code = 200

        # Should not raise any exception
        _handle_error_response(mock_response)

    def test_handle_error_response_json_error(self):
        """Test _handle_error_response with JSON error response."""
        mock_response = MagicMock()
        mock_response.is_success = False
        mock_response.status_code = 400
        mock_response.json.return_value = {"error": "Validation failed"}
        mock_response.url = httpx.URL("http://test.com/api")

        with pytest.raises(PinazuAPIError) as exc_info:
            _handle_error_response(mock_response)

        assert exc_info.value.status_code == 400
        assert exc_info.value.message == "Validation failed"
        assert exc_info.value.url == "http://test.com/api"

    def test_handle_error_response_json_with_message_field(self):
        """Test error handling when JSON has 'message' field."""
        mock_response = MagicMock()
        mock_response.is_success = False
        mock_response.status_code = 404
        mock_response.json.return_value = {"message": "Resource not found"}
        mock_response.url = httpx.URL("http://test.com/api")

        with pytest.raises(PinazuAPIError) as exc_info:
            _handle_error_response(mock_response)

        assert exc_info.value.status_code == 404
        assert exc_info.value.message == "Resource not found"

    def test_handle_error_response_nested_error_dict(self):
        """Test error handling when error field is a dict."""
        mock_response = MagicMock()
        mock_response.is_success = False
        mock_response.status_code = 422
        mock_response.json.return_value = {
            "error": {"field": "name", "reason": "required"}
        }
        mock_response.url = httpx.URL("http://test.com/api")

        with pytest.raises(PinazuAPIError) as exc_info:
            _handle_error_response(mock_response)

        assert exc_info.value.status_code == 422
        assert (
            "{'field': 'name', 'reason': 'required'}" in exc_info.value.message
        )  # noqa: E501

    def test_handle_error_response_invalid_json(self):
        """Test error handling when response is not valid JSON."""
        mock_response = MagicMock()
        mock_response.is_success = False
        mock_response.status_code = 500
        mock_response.json.side_effect = ValueError("Invalid JSON")
        mock_response.text = "Internal Server Error"
        mock_response.url = httpx.URL("http://test.com/api")

        with pytest.raises(PinazuAPIError) as exc_info:
            _handle_error_response(mock_response)

        assert exc_info.value.status_code == 500
        assert exc_info.value.message == "Internal Server Error"

    def test_handle_error_response_empty_text(self):
        """Test error handling when response has no text."""
        mock_response = MagicMock()
        mock_response.is_success = False
        mock_response.status_code = 503
        mock_response.json.side_effect = ValueError("Invalid JSON")
        mock_response.text = ""
        mock_response.url = httpx.URL("http://test.com/api")

        with pytest.raises(PinazuAPIError) as exc_info:
            _handle_error_response(mock_response)

        assert exc_info.value.status_code == 503
        assert exc_info.value.message == "HTTP 503"

    def test_handle_error_response_non_dict_json(self):
        """Test error handling when JSON response is not a dict."""
        mock_response = MagicMock()
        mock_response.is_success = False
        mock_response.status_code = 400
        mock_response.json.return_value = ["error", "list"]
        mock_response.url = httpx.URL("http://test.com/api")

        with pytest.raises(PinazuAPIError) as exc_info:
            _handle_error_response(mock_response)

        assert exc_info.value.status_code == 400
        assert "['error', 'list']" in exc_info.value.message

    def test_client_network_error(self, client):
        """Test client behavior with network errors."""
        with patch.object(
            client, "get", side_effect=httpx.ConnectError("Connection failed")
        ):
            with pytest.raises(httpx.ConnectError):
                client.list_agents()

    def test_client_timeout_error(self, client):
        """Test client behavior with timeout errors."""
        with patch.object(
            client,
            "get",
            side_effect=httpx.TimeoutException("Request timed out"),  # noqa: E501
        ):
            with pytest.raises(httpx.TimeoutException):
                client.list_agents()

    def test_client_unauthorized_error(self, client, mock_responses):
        """Test client behavior with 401 Unauthorized."""
        error_data = {"error": "Authentication required"}
        mock_response = mock_responses(error_data, 401)

        with patch.object(client, "get", return_value=mock_response):
            with pytest.raises(PinazuAPIError) as exc_info:
                client.list_agents()

            assert exc_info.value.status_code == 401
            assert "Authentication required" in str(exc_info.value)

    def test_client_forbidden_error(self, client, mock_responses):
        """Test client behavior with 403 Forbidden."""
        error_data = {"error": "Insufficient permissions"}
        mock_response = mock_responses(error_data, 403)

        with patch.object(client, "get", return_value=mock_response):
            with pytest.raises(PinazuAPIError) as exc_info:
                client.list_agents()

            assert exc_info.value.status_code == 403
            assert "Insufficient permissions" in str(exc_info.value)

    def test_client_server_error(self, client, mock_responses):
        """Test client behavior with 500 Internal Server Error."""
        error_data = {"error": "Internal server error"}
        mock_response = mock_responses(error_data, 500)

        with patch.object(client, "get", return_value=mock_response):
            with pytest.raises(PinazuAPIError) as exc_info:
                client.list_agents()

            assert exc_info.value.status_code == 500
            assert "Internal server error" in str(exc_info.value)

    def test_client_bad_gateway_error(self, client, mock_responses):
        """Test client behavior with 502 Bad Gateway."""
        mock_response = mock_responses("Bad Gateway", 502)

        with patch.object(client, "get", return_value=mock_response):
            with pytest.raises(PinazuAPIError) as exc_info:
                client.list_agents()

            assert exc_info.value.status_code == 502

    def test_client_service_unavailable_error(self, client, mock_responses):
        """Test client behavior with 503 Service Unavailable."""
        mock_response = mock_responses("Service Unavailable", 503)

        with patch.object(client, "get", return_value=mock_response):
            with pytest.raises(PinazuAPIError) as exc_info:
                client.list_agents()

            assert exc_info.value.status_code == 503


class TestAsyncErrorHandling:
    """Test class for async error handling functionality."""

    @pytest.mark.asyncio
    async def test_async_client_network_error(self, async_client):
        """Test async client behavior with network errors."""
        with patch.object(
            async_client,
            "get",
            side_effect=httpx.ConnectError("Async connection failed"),
        ):
            with pytest.raises(httpx.ConnectError):
                await async_client.list_agents()

    @pytest.mark.asyncio
    async def test_async_client_api_error(self, async_client, mock_responses):
        """Test async client behavior with API errors."""
        error_data = {"error": "Async validation failed"}
        mock_response = mock_responses(error_data, 422)

        with patch.object(async_client, "post", return_value=mock_response):
            with pytest.raises(PinazuAPIError) as exc_info:
                await async_client.create_agent(name="test")

            assert exc_info.value.status_code == 422
            assert "Async validation failed" in str(exc_info.value)

    @pytest.mark.asyncio
    async def test_async_client_timeout_error(self, async_client):
        """Test async client behavior with timeout errors."""
        with patch.object(
            async_client,
            "get",
            side_effect=httpx.TimeoutException("Async timeout"),  # noqa: E501
        ):
            with pytest.raises(httpx.TimeoutException):
                await async_client.list_agents()


class TestClientInitialization:
    """Test class for client initialization and configuration."""

    def test_client_default_base_url(self):
        """Test client with default base URL."""
        client = Client()
        assert str(client.base_url) == "http://localhost:8080"

    def test_client_custom_base_url(self):
        """Test client with custom base URL."""
        client = Client(base_url="https://api.example.com")
        assert str(client.base_url) == "https://api.example.com"

    def test_client_custom_headers(self):
        """Test client with custom headers."""
        custom_headers = {"Authorization": "Bearer token123"}
        client = Client(headers=custom_headers)
        assert client.headers.get("Authorization") == "Bearer token123"

    def test_async_client_default_base_url(self):
        """Test async client with default base URL."""
        client = AsyncClient()
        assert str(client.base_url) == "http://localhost:8080"

    def test_async_client_custom_base_url(self):
        """Test async client with custom base URL."""
        client = AsyncClient(base_url="https://api.example.com")
        assert str(client.base_url) == "https://api.example.com"

    def test_async_client_custom_headers(self):
        """Test async client with custom headers."""
        custom_headers = {"Authorization": "Bearer token456"}
        client = AsyncClient(headers=custom_headers)
        assert client.headers.get("Authorization") == "Bearer token456"

    def test_client_additional_kwargs(self):
        """Test client with additional httpx kwargs."""
        client = Client(timeout=30.0)
        # Test that timeout was passed correctly
        assert client.timeout.read == 30.0

    def test_async_client_additional_kwargs(self):
        """Test async client with additional httpx kwargs."""
        client = AsyncClient(timeout=45.0)
        # Test that timeout was passed correctly
        assert client.timeout.read == 45.0
