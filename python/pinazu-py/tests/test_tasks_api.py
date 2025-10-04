"""Test suite for Tasks API client methods."""

import pytest
from uuid import UUID
from unittest.mock import patch
from pinazu import PinazuAPIError
from pinazu.api.models_generated import Task, TaskList, TaskRun


class TestTasksAPI:
    """Test class for Tasks API methods."""

    def test_create_task(self, client, sample_uuid, mock_responses):
        """Test creating a new task."""
        expected_task = Task(
            id=UUID("12345678-1234-1234-1234-123456789012"),
            thread_id=sample_uuid,
            created_by=sample_uuid,
            additional_info={"test": "data"},
            max_request_loop=5,
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:00:00Z",
        )

        mock_response = mock_responses(
            expected_task.model_dump(mode="json"),
            201,
        )

        with patch.object(
            client,
            "post",
            return_value=mock_response,
        ) as mock_post:  # noqa: E501
            result = client.create_task(
                thread_id=sample_uuid,
                additional_info={"test": "data"},
                max_request_loop=5,
            )

            mock_post.assert_called_once()
            call_args = mock_post.call_args
            assert call_args[1]["url"] == "/v1/tasks"

            assert result.id == expected_task.id
            assert result.thread_id == sample_uuid
            assert result.additional_info == {"test": "data"}
            assert result.max_request_loop == 5

    def test_get_task(self, client, sample_uuid, mock_responses):
        """Test getting a task by ID."""
        expected_task = Task(
            id=sample_uuid,
            thread_id=UUID("12345678-1234-1234-1234-123456789013"),
            created_by=sample_uuid,
            additional_info={},
            max_request_loop=3,
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:01:00Z",
        )

        mock_response = mock_responses(expected_task.model_dump(mode="json"))

        with patch.object(
            client, "get", return_value=mock_response
        ) as mock_get:  # noqa: E501
            result = client.get_task(sample_uuid)

            mock_get.assert_called_once_with(f"/v1/tasks/{sample_uuid}")
            assert result.id == sample_uuid
            assert result.max_request_loop == 3

    def test_list_tasks(self, client, sample_uuid, mock_responses):
        """Test listing tasks with pagination."""
        tasks_data = TaskList(
            tasks=[
                Task(
                    id=UUID("12345678-1234-1234-1234-123456789012"),
                    thread_id=UUID("12345678-1234-1234-1234-123456789013"),
                    created_by=sample_uuid,
                    additional_info={},
                    max_request_loop=5,
                    created_at="2025-01-01T00:00:00Z",
                    updated_at="2025-01-01T00:00:00Z",
                ),
                Task(
                    id=UUID("12345678-1234-1234-1234-123456789014"),
                    thread_id=UUID("12345678-1234-1234-1234-123456789015"),
                    created_by=sample_uuid,
                    additional_info={"priority": "high"},
                    max_request_loop=10,
                    created_at="2025-01-01T00:00:00Z",
                    updated_at="2025-01-01T00:05:00Z",
                ),
            ],
            total=2,
            page=1,
            per_page=10,
            total_pages=1,
        )

        mock_response = mock_responses(tasks_data.model_dump(mode="json"))

        with patch.object(
            client, "get", return_value=mock_response
        ) as mock_get:  # noqa: E501
            result = client.list_tasks(page=1, per_page=10)

            mock_get.assert_called_once_with(
                "/v1/tasks", params={"page": 1, "per_page": 10}
            )
            assert len(result.tasks) == 2

    def test_update_task(self, client, sample_uuid, mock_responses):
        """Test updating an existing task."""
        updated_task = Task(
            id=sample_uuid,
            thread_id=UUID("12345678-1234-1234-1234-123456789013"),
            created_by=sample_uuid,
            additional_info={"updated": True, "priority": "high"},
            max_request_loop=10,
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T01:00:00Z",
        )

        mock_response = mock_responses(updated_task.model_dump(mode="json"))

        with patch.object(
            client, "put", return_value=mock_response
        ) as mock_put:  # noqa: E501
            result = client.update_task(
                task_id=sample_uuid,
                additional_info={"updated": True, "priority": "high"},
                max_request_loop=10,
            )

            mock_put.assert_called_once()
            call_args = mock_put.call_args
            assert call_args[1]["url"] == f"/v1/tasks/{sample_uuid}"
            assert result.additional_info == {
                "updated": True,
                "priority": "high",
            }
            assert result.max_request_loop == 10

    def test_delete_task(self, client, sample_uuid, mock_responses):
        """Test deleting a task."""
        mock_response = mock_responses(None, 204)

        with patch.object(
            client, "delete", return_value=mock_response
        ) as mock_delete:  # noqa: E501
            client.delete_task(sample_uuid)

            mock_delete.assert_called_once_with(f"/v1/tasks/{sample_uuid}")

    def test_execute_task(self, client, sample_uuid, mock_responses):
        """Test executing a task (non-streaming)."""
        agent_id = UUID("12345678-1234-1234-1234-123456789020")
        expected_task_run = TaskRun(
            task_run_id=UUID("12345678-1234-1234-1234-123456789016"),
            task_id=sample_uuid,
            status="pending",
            current_loops=1,
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:00:00Z",
        )

        mock_response = mock_responses(
            expected_task_run.model_dump(mode="json"),
            200,
        )

        with patch.object(
            client, "post", return_value=mock_response
        ) as mock_post:  # noqa: E501
            result = client.execute_task(
                task_id=sample_uuid,
                agent_id=agent_id,
                current_loops="1",
                stream=False,
            )

            mock_post.assert_called_once()
            call_args = mock_post.call_args
            assert call_args[1]["url"] == f"/v1/tasks/{sample_uuid}/execute"
            assert result.task_id == sample_uuid
            assert result.current_loops == 1
            assert result.status == "pending"

    def test_execute_task_streaming(self, client, sample_uuid):
        """Test executing a task with streaming."""
        agent_id = UUID("12345678-1234-1234-1234-123456789020")

        # Mock SSE response data
        sse_events = [
            {"type": "message_start", "message": {"role": "assistant"}},
            {"type": "content_block_delta", "delta": {"text": "Hello"}},
            {"type": "content_block_delta", "delta": {"text": " World"}},
            {"type": "message_stop"},
        ]

        with patch.object(client, "_execute_task_stream") as mock_stream:
            mock_stream.return_value = iter(sse_events)

            result = client.execute_task(
                task_id=sample_uuid,
                agent_id=agent_id,
                current_loops="1",
                stream=True,
            )

            # Collect all events from the stream
            events = list(result)

            mock_stream.assert_called_once()
            assert len(events) == 4
            assert events[0]["type"] == "message_start"
            assert events[1]["delta"]["text"] == "Hello"
            assert events[2]["delta"]["text"] == " World"
            assert events[3]["type"] == "message_stop"

    def test_list_task_runs(self, client, sample_uuid, mock_responses):
        """Test listing task runs for a task."""
        task_runs_data = [
            TaskRun(
                task_run_id=UUID("12345678-1234-1234-1234-123456789016"),
                task_id=sample_uuid,
                status="completed",
                current_loops=1,
                created_at="2025-01-01T00:00:00Z",
                updated_at="2025-01-01T00:01:00Z",
            ),
            TaskRun(
                task_run_id=UUID("12345678-1234-1234-1234-123456789017"),
                task_id=sample_uuid,
                status="failed",
                current_loops="2",
                created_at="2025-01-01T00:01:00Z",
                updated_at="2025-01-01T00:02:00Z",
            ),
        ]

        mock_response = mock_responses(
            [run.model_dump(mode="json") for run in task_runs_data]
        )

        with patch.object(
            client, "get", return_value=mock_response
        ) as mock_get:  # noqa: E501
            result = client.list_task_runs(sample_uuid)

            mock_get.assert_called_once_with(f"/v1/tasks/{sample_uuid}/runs")
            assert len(result) == 2
            assert result[0].status == "completed"
            assert result[1].status == "failed"

    def test_get_task_run(self, client, sample_uuid, mock_responses):
        """Test getting task run status."""
        task_run = TaskRun(
            task_run_id=sample_uuid,
            task_id=UUID("12345678-1234-1234-1234-123456789018"),
            status="completed",
            current_loops=3,
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:03:00Z",
        )

        mock_response = mock_responses(task_run.model_dump(mode="json"))

        with patch.object(
            client, "get", return_value=mock_response
        ) as mock_get:  # noqa: E501
            result = client.get_task_run(sample_uuid)

            mock_get.assert_called_once_with(f"/v1/tasks/{sample_uuid}/status")
            assert result.task_run_id == sample_uuid
            assert result.status == "completed"
            assert result.current_loops == 3

    def test_get_nonexistent_task(self, client, sample_uuid, mock_responses):
        """Test getting a non-existent task returns 404."""
        error_data = {
            "resource": "Task",
            "id": str(sample_uuid),
            "error": "Task not found",
        }
        mock_response = mock_responses(error_data, 404)

        with patch.object(client, "get", return_value=mock_response):
            with pytest.raises(PinazuAPIError) as exc_info:
                client.get_task(sample_uuid)

            assert exc_info.value.status_code == 404
            assert "Task not found" in str(exc_info.value)

    def test_create_task_validation_error(self, client, mock_responses):
        """Test creating task with missing required fields."""
        error_data = {"error": "Validation failed: thread_id is required"}
        mock_response = mock_responses(error_data, 400)

        with patch.object(client, "post", return_value=mock_response):
            # This should raise a pydantic validation error, not API error
            with pytest.raises(Exception):  # Pydantic validation error
                client.create_task(thread_id=None, additional_info={})

    def test_execute_nonexistent_task(
        self,
        client,
        sample_uuid,
        mock_responses,
    ):
        """Test executing a non-existent task returns 404."""
        agent_id = UUID("12345678-1234-1234-1234-123456789020")
        error_data = {
            "resource": "Task",
            "id": str(sample_uuid),
            "error": "Task not found",
        }
        mock_response = mock_responses(error_data, 404)

        with patch.object(client, "post", return_value=mock_response):
            with pytest.raises(PinazuAPIError) as exc_info:
                client.execute_task(
                    task_id=sample_uuid,
                    agent_id=agent_id,
                    current_loops="1",
                    stream=False,
                )

            assert exc_info.value.status_code == 404


class TestTasksAPIAsync:
    """Test class for async Tasks API methods."""

    @pytest.mark.asyncio
    async def test_async_create_task(
        self,
        async_client,
        sample_uuid,
        mock_responses,
    ):
        """Test creating task with async client."""
        expected_task = Task(
            id=UUID("12345678-1234-1234-1234-123456789012"),
            thread_id=sample_uuid,
            created_by=sample_uuid,
            additional_info={"async": True},
            max_request_loop=5,
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:00:00Z",
        )

        mock_response = mock_responses(
            expected_task.model_dump(mode="json"),
            201,
        )

        with patch.object(async_client, "post", return_value=mock_response):
            result = await async_client.create_task(
                thread_id=sample_uuid,
                additional_info={"async": "test"},
                max_request_loop=5,
            )

            assert result.thread_id == sample_uuid

    @pytest.mark.asyncio
    async def test_async_execute_task(
        self,
        async_client,
        sample_uuid,
        mock_responses,
    ):
        """Test executing task with async client (non-streaming)."""
        agent_id = UUID("12345678-1234-1234-1234-123456789020")
        expected_task_run = TaskRun(
            task_run_id=UUID("12345678-1234-1234-1234-123456789016"),
            task_id=sample_uuid,
            status="pending",
            current_loops=1,
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:00:00Z",
        )

        mock_response = mock_responses(
            expected_task_run.model_dump(mode="json"),
            200,
        )

        with patch.object(async_client, "post", return_value=mock_response):
            result = await async_client.execute_task(
                task_id=sample_uuid,
                agent_id=agent_id,
                current_loops="1",
                stream=False,
            )

            assert result.task_id == sample_uuid
            assert result.status == "pending"

    @pytest.mark.asyncio
    async def test_async_execute_task_streaming(
        self,
        async_client,
        sample_uuid,
    ):
        """Test executing task with async client (streaming)."""
        agent_id = UUID("12345678-1234-1234-1234-123456789020")

        # Mock SSE response data
        async def mock_async_stream():
            events = [
                {"type": "message_start", "message": {"role": "assistant"}},
                {"type": "content_block_delta", "delta": {"text": "Hello"}},
                {"type": "content_block_delta", "delta": {"text": " Async"}},
                {"type": "message_stop"},
            ]
            for event in events:
                yield event

        with patch.object(async_client, "_execute_task_stream") as mock_stream:
            mock_stream.return_value = mock_async_stream()

            result = await async_client.execute_task(
                task_id=sample_uuid,
                agent_id=agent_id,
                current_loops="1",
                stream=True,
            )

            # Collect all events from the async stream
            events = []
            async for event in result:
                events.append(event)

            mock_stream.assert_called_once()
            assert len(events) == 4
            assert events[0]["type"] == "message_start"
            assert events[1]["delta"]["text"] == "Hello"
            assert events[2]["delta"]["text"] == " Async"
            assert events[3]["type"] == "message_stop"
