"""Test suite for Flows API client methods."""

import pytest
from uuid import UUID
from unittest.mock import patch
from pinazu import PinazuAPIError
from pinazu.api.models_generated import Flow, FlowList, FlowRun


class TestFlowsAPI:
    """Test class for Flows API methods."""

    def test_create_flow(self, client, test_flow_data, mock_responses):
        """Test creating a new flow."""
        expected_flow = Flow(
            id=UUID("12345678-1234-1234-1234-123456789012"),
            name=test_flow_data["name"],
            description=test_flow_data["description"],
            engine=test_flow_data["engine"],
            entrypoint=test_flow_data["entrypoint"],
            code_location=test_flow_data["code_location"],
            parameters_schema=test_flow_data["parameters_schema"],
            tags=test_flow_data["tags"],
            additional_info={},
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:00:00Z",
        )

        mock_response = mock_responses(
            expected_flow.model_dump(mode="json"),
            201,
        )

        with patch.object(
            client, "post", return_value=mock_response
        ) as mock_post:  # noqa: E501
            result = client.create_flow(**test_flow_data)

            mock_post.assert_called_once()
            call_args = mock_post.call_args
            assert call_args[1]["url"] == "/v1/flows"

            assert result.id == expected_flow.id
            assert result.name == test_flow_data["name"]
            assert result.engine == test_flow_data["engine"]
            assert (
                result.parameters_schema == test_flow_data["parameters_schema"]
            )  # noqa: E501

    def test_get_flow(self, client, sample_uuid, mock_responses):
        """Test getting a flow by ID."""
        expected_flow = Flow(
            id=sample_uuid,
            name="Test Flow",
            description="Test Description",
            engine="python",
            entrypoint="main.py",
            code_location="s3://test-bucket/flow.py",
            parameters_schema={"type": "object"},
            tags=["test"],
            additional_info={},
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:00:00Z",
        )

        mock_response = mock_responses(expected_flow.model_dump(mode="json"))

        with patch.object(
            client, "get", return_value=mock_response
        ) as mock_get:  # noqa: E501
            result = client.get_flow(sample_uuid)

            mock_get.assert_called_once_with(f"/v1/flows/{sample_uuid}")
            assert result.id == sample_uuid
            assert result.name == "Test Flow"
            assert result.engine == "python"

    def test_list_flows(self, client, mock_responses):
        """Test listing flows with pagination."""
        flows_data = FlowList(
            flows=[
                Flow(
                    id=UUID("12345678-1234-1234-1234-123456789012"),
                    name="Flow 1",
                    description="First flow",
                    engine="python",
                    entrypoint="main1.py",
                    code_location="s3://test/flow1.py",
                    parameters_schema={"type": "object"},
                    tags=["test"],
                    additional_info={},
                    created_at="2025-01-01T00:00:00Z",
                    updated_at="2025-01-01T00:00:00Z",
                ),
                Flow(
                    id=UUID("12345678-1234-1234-1234-123456789013"),
                    name="Flow 2",
                    description="Second flow",
                    engine="process",
                    entrypoint="main2.py",
                    code_location="s3://test/flow2.py",
                    parameters_schema={"type": "object"},
                    tags=["test"],
                    additional_info={},
                    created_at="2025-01-01T00:00:00Z",
                    updated_at="2025-01-01T00:00:00Z",
                ),
            ],
            total=2,
            page=1,
            per_page=10,
            total_pages=1,
        )

        mock_response = mock_responses(flows_data.model_dump(mode="json"))

        with patch.object(
            client, "get", return_value=mock_response
        ) as mock_get:  # noqa: E501
            result = client.list_flows(page=1, per_page=10)

            mock_get.assert_called_once_with(
                "/v1/flows", params={"page": 1, "per_page": 10}
            )
            assert len(result.flows) == 2

    def test_update_flow(self, client, sample_uuid, mock_responses):
        """Test updating an existing flow."""
        updated_flow = Flow(
            id=sample_uuid,
            name="Updated Flow",
            description="Updated Description",
            engine="python",
            entrypoint="updated_main.py",
            code_location="s3://test/updated_flow.py",
            parameters_schema={"type": "object", "updated": True},
            tags=["updated", "test"],
            additional_info={},
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T01:00:00Z",
        )

        mock_response = mock_responses(updated_flow.model_dump(mode="json"))

        with patch.object(
            client, "put", return_value=mock_response
        ) as mock_put:  # noqa: E501
            result = client.update_flow(
                flow_id=sample_uuid,
                name="Updated Flow",
                description="Updated Description",
                entrypoint="updated_main.py",
                code_location="s3://test/updated_flow.py",
                tags=["updated", "test"],
            )

            mock_put.assert_called_once()
            call_args = mock_put.call_args
            assert call_args[1]["url"] == f"/v1/flows/{sample_uuid}"
            assert result.name == "Updated Flow"
            assert result.description == "Updated Description"
            assert result.tags == ["updated", "test"]

    def test_delete_flow(self, client, sample_uuid, mock_responses):
        """Test deleting a flow."""
        mock_response = mock_responses(None, 204)

        with patch.object(
            client, "delete", return_value=mock_response
        ) as mock_delete:  # noqa: E501
            client.delete_flow(sample_uuid)

            mock_delete.assert_called_once_with(f"/v1/flows/{sample_uuid}")

    def test_execute_flow(self, client, sample_uuid, mock_responses):
        """Test executing a flow."""
        expected_flow_run = FlowRun(
            flow_run_id=UUID("12345678-1234-1234-1234-123456789014"),
            flow_id=sample_uuid,
            engine="python",
            status="pending",
            parameters={"input_text": "test input"},
            success_task_results={},
            task_statuses={},
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:00:00Z",
        )

        mock_response = mock_responses(
            expected_flow_run.model_dump(mode="json"),
            201,
        )

        with patch.object(
            client, "post", return_value=mock_response
        ) as mock_post:  # noqa: E501
            result = client.execute_flow(
                flow_id=sample_uuid, parameters={"input_text": "test input"}
            )

            mock_post.assert_called_once()
            call_args = mock_post.call_args
            assert call_args[1]["url"] == f"/v1/flows/{sample_uuid}/execute"
            assert result.flow_id == sample_uuid
            assert result.parameters == {"input_text": "test input"}
            assert result.status == "pending"

    def test_get_flow_run(self, client, sample_uuid, mock_responses):
        """Test getting flow run status."""
        flow_run = FlowRun(
            flow_run_id=sample_uuid,
            flow_id=UUID("12345678-1234-1234-1234-123456789015"),
            engine="python",
            status="completed",
            parameters={"input_text": "test"},
            success_task_results={"output_text": "processed test"},
            task_statuses={},
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:05:00Z",
        )

        mock_response = mock_responses(flow_run.model_dump(mode="json"))

        with patch.object(
            client, "get", return_value=mock_response
        ) as mock_get:  # noqa: E501
            result = client.get_flow_run(sample_uuid)

            mock_get.assert_called_once_with(f"/v1/flows/{sample_uuid}/status")
            assert result.flow_run_id == sample_uuid
            assert result.status == "completed"
            assert result.success_task_results == {
                "output_text": "processed test",
            }

    def test_get_nonexistent_flow(self, client, sample_uuid, mock_responses):
        """Test getting a non-existent flow returns 404."""
        error_data = {
            "resource": "Flow",
            "id": str(sample_uuid),
            "error": "Flow not found",
        }
        mock_response = mock_responses(error_data, 404)

        with patch.object(client, "get", return_value=mock_response):
            with pytest.raises(PinazuAPIError) as exc_info:
                client.get_flow(sample_uuid)

            assert exc_info.value.status_code == 404
            assert "Flow not found" in str(exc_info.value)

    def test_create_flow_validation_error(self, client, mock_responses):
        """Test creating flow with missing required fields."""
        error_data = {"error": "Validation failed: name is required"}
        mock_response = mock_responses(error_data, 400)

        with patch.object(client, "post", return_value=mock_response):
            with pytest.raises(PinazuAPIError) as exc_info:
                client.create_flow(
                    name="",  # Invalid empty name
                    engine="python",
                    entrypoint="main.py",
                    code_location="s3://test/flow.py",
                    parameters_schema={},
                )

            assert exc_info.value.status_code == 400
            assert "Validation failed" in str(exc_info.value)

    def test_execute_nonexistent_flow(
        self,
        client,
        sample_uuid,
        mock_responses,
    ):
        """Test executing a non-existent flow returns 404."""
        error_data = {
            "resource": "Flow",
            "id": str(sample_uuid),
            "error": "Flow not found",
        }
        mock_response = mock_responses(error_data, 404)

        with patch.object(client, "post", return_value=mock_response):
            with pytest.raises(PinazuAPIError) as exc_info:
                client.execute_flow(
                    flow_id=sample_uuid,
                    parameters={"test": "data"},
                )

            assert exc_info.value.status_code == 404

    def test_get_nonexistent_flow_run(
        self,
        client,
        sample_uuid,
        mock_responses,
    ):
        """Test getting status of non-existent flow run."""
        error_data = {
            "resource": "FlowRun",
            "id": str(sample_uuid),
            "error": "Flow run not found",
        }
        mock_response = mock_responses(error_data, 404)

        with patch.object(client, "get", return_value=mock_response):
            with pytest.raises(PinazuAPIError) as exc_info:
                client.get_flow_run(sample_uuid)

            assert exc_info.value.status_code == 404


class TestFlowsAPIAsync:
    """Test class for async Flows API methods."""

    @pytest.mark.asyncio
    async def test_async_create_flow(
        self, async_client, test_flow_data, mock_responses
    ):
        """Test creating flow with async client."""
        expected_flow = Flow(
            id=UUID("12345678-1234-1234-1234-123456789012"),
            name=test_flow_data["name"],
            description=test_flow_data["description"],
            engine=test_flow_data["engine"],
            entrypoint=test_flow_data["entrypoint"],
            code_location=test_flow_data["code_location"],
            parameters_schema=test_flow_data["parameters_schema"],
            tags=test_flow_data["tags"],
            additional_info={},
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:00:00Z",
        )

        mock_response = mock_responses(
            expected_flow.model_dump(mode="json"),
            201,
        )

        with patch.object(async_client, "post", return_value=mock_response):
            result = await async_client.create_flow(**test_flow_data)

            assert result.name == test_flow_data["name"]
            assert result.engine == test_flow_data["engine"]

    @pytest.mark.asyncio
    async def test_async_execute_flow(
        self,
        async_client,
        sample_uuid,
        mock_responses,
    ):
        """Test executing flow with async client."""
        expected_flow_run = FlowRun(
            flow_run_id=UUID("12345678-1234-1234-1234-123456789014"),
            flow_id=sample_uuid,
            engine="python",
            status="pending",
            parameters={"async": "test"},
            success_task_results={},
            task_statuses={},
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:00:00Z",
        )

        mock_response = mock_responses(
            expected_flow_run.model_dump(mode="json"),
            201,
        )

        with patch.object(async_client, "post", return_value=mock_response):
            result = await async_client.execute_flow(
                flow_id=sample_uuid, parameters={"async": "test"}
            )

            assert result.flow_id == sample_uuid
            assert result.parameters == {"async": "test"}
            assert result.status == "pending"
