"""Test suite for Agents API client methods."""

import pytest
from uuid import UUID
from unittest.mock import patch
from pinazu import PinazuAPIError
from pinazu.api.models_generated import Agent, AgentList


class TestAgentsAPI:
    """Test class for Agents API methods."""

    def test_create_agent(self, client, test_agent_data, mock_responses):
        """Test creating a new agent."""
        expected_agent = Agent(
            id=UUID("12345678-1234-1234-1234-123456789012"),
            name=test_agent_data["name"],
            description=test_agent_data["description"],
            specs=test_agent_data["specs"],
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:00:00Z",
            created_by=UUID("12345678-1234-1234-1234-123456789012"),
        )

        mock_response = mock_responses(
            expected_agent.model_dump(mode="json"),
            201,
        )

        with patch.object(
            client, "post", return_value=mock_response
        ) as mock_post:  # noqa: E501
            result = client.create_agent(**test_agent_data)

            # Verify the request was made correctly
            mock_post.assert_called_once()
            call_args = mock_post.call_args
            assert call_args[1]["url"] == "/v1/agents"

            # Verify the result
            assert result.id == expected_agent.id
            assert result.name == test_agent_data["name"]
            assert result.description == test_agent_data["description"]
            assert result.specs == test_agent_data["specs"]

    def test_get_agent(self, client, sample_uuid, mock_responses):
        """Test getting an agent by ID."""
        expected_agent = Agent(
            id=sample_uuid,
            name="Test Agent",
            description="Test Description",
            specs="test specs",
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:00:00Z",
            created_by=sample_uuid,
        )

        mock_response = mock_responses(expected_agent.model_dump(mode="json"))

        with patch.object(
            client, "get", return_value=mock_response
        ) as mock_get:  # noqa: E501
            result = client.get_agent(sample_uuid)

            mock_get.assert_called_once_with(f"/v1/agents/{sample_uuid}")
            assert result.id == sample_uuid
            assert result.name == "Test Agent"

    def test_list_agents(self, client, mock_responses):
        """Test listing all agents."""
        agents_data = AgentList(
            agents=[
                Agent(
                    id=UUID("12345678-1234-1234-1234-123456789012"),
                    name="Agent 1",
                    description="First agent",
                    specs="specs1",
                    created_at="2025-01-01T00:00:00Z",
                    updated_at="2025-01-01T00:00:00Z",
                    created_by=UUID("12345678-1234-1234-1234-123456789012"),
                ),
                Agent(
                    id=UUID("12345678-1234-1234-1234-123456789013"),
                    name="Agent 2",
                    description="Second agent",
                    specs="specs2",
                    created_at="2025-01-01T00:00:00Z",
                    updated_at="2025-01-01T00:00:00Z",
                    created_by=UUID("12345678-1234-1234-1234-123456789012"),
                ),
            ],
            total=2,
            page=1,
            per_page=10,
            total_pages=1
        )

        mock_response = mock_responses(agents_data.model_dump(mode="json"))

        with patch.object(
            client, "get", return_value=mock_response
        ) as mock_get:  # noqa: E501
            result = client.list_agents()

            mock_get.assert_called_once_with("/v1/agents")
            assert len(result.agents) == 2
            assert result.agents[0].name == "Agent 1"
            assert result.agents[1].name == "Agent 2"

    def test_update_agent(self, client, sample_uuid, mock_responses):
        """Test updating an existing agent."""
        updated_agent = Agent(
            id=sample_uuid,
            name="Updated Agent",
            description="Updated Description",
            specs="updated specs",
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T01:00:00Z",
            created_by=sample_uuid,
        )

        mock_response = mock_responses(updated_agent.model_dump(mode="json"))

        with patch.object(
            client, "put", return_value=mock_response
        ) as mock_put:  # noqa: E501
            result = client.update_agent(
                agent_id=sample_uuid,
                name="Updated Agent",
                description="Updated Description",
                specs="updated specs",
            )

            mock_put.assert_called_once()
            call_args = mock_put.call_args
            assert call_args[1]["url"] == f"/v1/agents/{sample_uuid}"
            assert result.name == "Updated Agent"
            assert result.description == "Updated Description"

    def test_delete_agent(self, client, sample_uuid, mock_responses):
        """Test deleting an agent."""
        mock_response = mock_responses(None, 204)

        with patch.object(
            client, "delete", return_value=mock_response
        ) as mock_delete:  # noqa: E501
            # Should not raise an exception
            client.delete_agent(sample_uuid)

            mock_delete.assert_called_once_with(f"/v1/agents/{sample_uuid}")

    def test_get_nonexistent_agent(self, client, sample_uuid, mock_responses):
        """Test getting a non-existent agent returns 404."""
        error_data = {
            "resource": "Agent",
            "id": str(sample_uuid),
            "error": "Agent not found",
        }
        mock_response = mock_responses(error_data, 404)

        with patch.object(client, "get", return_value=mock_response):
            with pytest.raises(PinazuAPIError) as exc_info:
                client.get_agent(sample_uuid)

            assert exc_info.value.status_code == 404
            assert "Agent not found" in str(exc_info.value)

    def test_create_agent_validation_error(self, client, mock_responses):
        """Test creating agent with validation errors."""
        error_data = {"error": "Validation failed: name is required"}
        mock_response = mock_responses(error_data, 400)

        with patch.object(client, "post", return_value=mock_response):
            with pytest.raises(PinazuAPIError) as exc_info:
                client.create_agent(name="")  # Invalid name

            assert exc_info.value.status_code == 400
            assert "Validation failed" in str(exc_info.value)

    def test_invalid_uuid_format(self, client, invalid_uuid, mock_responses):
        """Test API calls with invalid UUID format."""
        error_data = {"error": "Invalid UUID format"}
        mock_response = mock_responses(error_data, 400)

        with patch.object(client, "get", return_value=mock_response):
            with pytest.raises(PinazuAPIError) as exc_info:
                client.get_agent(invalid_uuid)

            assert exc_info.value.status_code == 400


class TestAgentsAPIAsync:
    """Test class for async Agents API methods."""

    @pytest.mark.asyncio
    async def test_async_create_agent(
        self, async_client, test_agent_data, mock_responses
    ):
        """Test creating agent with async client."""
        expected_agent = Agent(
            id=UUID("12345678-1234-1234-1234-123456789012"),
            name=test_agent_data["name"],
            description=test_agent_data["description"],
            specs=test_agent_data["specs"],
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:00:00Z",
            created_by=UUID("12345678-1234-1234-1234-123456789012"),
        )

        mock_response = mock_responses(
            expected_agent.model_dump(mode="json"),
            201,
        )

        with patch.object(
            async_client, "post", return_value=mock_response
        ) as mock_post:
            result = await async_client.create_agent(**test_agent_data)

            mock_post.assert_called_once()
            assert result.name == test_agent_data["name"]
            assert result.description == test_agent_data["description"]

    @pytest.mark.asyncio
    async def test_async_list_agents(self, async_client, mock_responses):
        """Test listing agents with async client."""
        agents_data = AgentList(
            agents=[
                Agent(
                    id=UUID("12345678-1234-1234-1234-123456789012"),
                    name="Async Agent",
                    description="Async test agent",
                    specs="async specs",
                    created_at="2025-01-01T00:00:00Z",
                    updated_at="2025-01-01T00:00:00Z",
                    created_by=UUID("12345678-1234-1234-1234-123456789012"),
                )
            ],
            total=1,
            page=1,
            per_page=10,
            total_pages=1
        )

        mock_response = mock_responses(agents_data.model_dump(mode="json"))

        with patch.object(async_client, "get", return_value=mock_response):
            result = await async_client.list_agents()

            assert len(result.agents) == 1
            assert result.agents[0].name == "Async Agent"


class TestAgentPermissions:
    """Test class for Agent Permissions API methods."""

    def test_add_permission_to_agent(
        self,
        client,
        sample_uuid,
        mock_responses,
    ):
        """Test adding a permission to an agent."""
        from pinazu.api.models_generated import AgentPermissionMapping

        expected_mapping = AgentPermissionMapping(
            mapping_id=UUID("12345678-1234-1234-1234-123456789014"),
            agent_id=sample_uuid,
            permission_id=sample_uuid,
            assigned_at="2025-01-01T00:00:00Z",
            assigned_by=sample_uuid,
        )

        mock_response = mock_responses(
            expected_mapping.model_dump(mode="json"),
            201,
        )

        with patch.object(
            client, "post", return_value=mock_response
        ) as mock_post:  # noqa: E501
            result = client.add_permission_to_agent(
                agent_id=sample_uuid, permission_id=sample_uuid
            )

            mock_post.assert_called_once()
            call_args = mock_post.call_args
            assert (
                call_args[1]["url"] == f"/v1/agents/{sample_uuid}/permissions"
            )  # noqa E501
            assert result.agent_id == sample_uuid
            assert result.permission_id == sample_uuid

    def test_list_agent_permissions(self, client, sample_uuid, mock_responses):
        """Test listing permissions for an agent."""
        from pinazu.api.models_generated import (
            AgentPermissionMappingList,
            AgentPermissionMapping,
        )

        permission_mappings = AgentPermissionMappingList(
            permissionMappings=[
                AgentPermissionMapping(
                    mapping_id=UUID("12345678-1234-1234-1234-123456789014"),
                    agent_id=sample_uuid,
                    permission_id=sample_uuid,
                    assigned_at="2025-01-01T00:00:00Z",
                    assigned_by=sample_uuid,
                )
            ],
            total=1,
            page=1,
            per_page=10,
            total_pages=1
        )

        mock_response = mock_responses(
            permission_mappings.model_dump(mode="json"),
        )

        with patch.object(client, "get", return_value=mock_response):
            result = client.list_agent_permissions(sample_uuid)

            assert len(result.permissionMappings) == 1
            assert result.permissionMappings[0].agent_id == sample_uuid

    def test_remove_permission_from_agent(
        self,
        client,
        sample_uuid,
        mock_responses,
    ):
        """Test removing a permission from an agent."""
        mock_response = mock_responses(None, 204)

        with patch.object(
            client, "delete", return_value=mock_response
        ) as mock_delete:  # noqa: E501
            client.remove_permission_from_agent(
                agent_id=sample_uuid, permission_id=sample_uuid
            )

            mock_delete.assert_called_once_with(
                url=f"/v1/agents/{sample_uuid}/permissions/{sample_uuid}"
            )

    def test_duplicate_permission_assignment(
        self,
        client,
        sample_uuid,
        mock_responses,
    ):
        """Test adding duplicate permission returns 409."""
        error_data = {"error": "Permission already assigned to agent"}
        mock_response = mock_responses(error_data, 409)

        with patch.object(client, "post", return_value=mock_response):
            with pytest.raises(PinazuAPIError) as exc_info:
                client.add_permission_to_agent(
                    agent_id=sample_uuid, permission_id=sample_uuid
                )

            assert exc_info.value.status_code == 409
