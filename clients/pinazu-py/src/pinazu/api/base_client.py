import httpx
from uuid import UUID
from typing import Optional, Dict, Any, List, Union, Iterator, AsyncIterator
import json
import codecs
import asyncio
import time

from .models_generated import (
    Agent,
    CreateAgentRequest,
    AgentList,
    AgentPermissionMapping,
    AgentPermissionMappingList,
    AddPermissionToAgentRequest,
    UpdateAgentRequest,
    CreateFlowRequest,
    Flow,
    FlowList,
    UpdateFlowRequest,
    ExecuteFlowRequest,
    FlowRun,
    CreateTaskRequest,
    Task,
    TaskList,
    UpdateTaskRequest,
    ExecuteTaskRequest,
    TaskRun,
    CreateToolRequest,
    Tool,
    ToolList,
    UpdateToolRequest,
    CreateMessageRequest,
    Message,
    MessageList,
    UpdateMessageRequest,
    CreateThreadRequest,
    Thread,
    ThreadList,
    UpdateThreadRequest,
    CreatePermissionRequest,
    Permission,
    PermissionList,
    UpdatePermissionRequest,
    CreateRoleRequest,
    Role,
    RoleList,
    UpdateRoleRequest,
    AddPermissionToRoleRequest,
    RolePermissionMapping,
    CreateUserRequest,
    User,
    UserList,
    UpdateUserRequest,
    AddRoleToUserRequest,
    UserRoleMapping,
)


class PinazuAPIError(Exception):
    """Custom exception for Pinazu API errors"""

    def __init__(self, status_code: int, message: str, url: str = None):
        self.status_code = status_code
        self.message = message
        self.url = url
        super().__init__(f"API Error {status_code}: {message}")


def _handle_error_response(response: httpx.Response) -> None:
    """Handle HTTP error responses by raising PinazuAPIError with details"""
    if response.is_success:
        return

    try:
        error_data = response.json()
        if isinstance(error_data, dict):
            message = error_data.get(
                "error",
                error_data.get("message", f"HTTP {response.status_code}"),  # noqa: E501
            )
            if isinstance(message, dict):
                message = str(message)
        else:
            message = str(error_data)
    except (ValueError, KeyError):
        message = (
            response.text
            if response.text
            else f"HTTP {response.status_code}"  # noqa: E501
        )

    raise PinazuAPIError(
        status_code=response.status_code,
        message=message,
        url=str(response.url),
    )


__all__ = [
    "Client",
    "AsyncClient",
    "PinazuAPIError",
]


class Client(httpx.Client):
    def __init__(
        self,
        base_url: Optional[str] = "http://localhost:8080",
        headers: Optional[Dict[str, str]] = None,
        timeout: Optional[float] = None,
        max_connections: int = 20,
        max_keepalive_connections: int = 10,
        keepalive_expiry: float = 30.0,
        **kwargs: Any,
    ):
        # Configure connection limits for better performance
        limits = httpx.Limits(
            max_connections=max_connections,
            max_keepalive_connections=max_keepalive_connections,
            keepalive_expiry=keepalive_expiry,
        )

        # Configure transport with TCP keep-alive
        transport = httpx.HTTPTransport(
            limits=limits,
            retries=3,
        )

        # Set default timeout if not provided
        if timeout is not None:
            kwargs["timeout"] = timeout

        # Enhanced headers for better connection management
        default_headers = {
            "Connection": "keep-alive",
            "Keep-Alive": "timeout=300, max=1000",
            "User-Agent": "pinazu-py/1.0",
        }
        if headers:
            default_headers.update(headers)

        # Connection health monitoring
        self._last_heartbeat = {}
        self._connection_stats = {
            "total_requests": 0,
            "streaming_requests": 0,
            "failed_connections": 0,
            "heartbeats_received": 0,
        }

        super().__init__(
            base_url=base_url,
            headers=default_headers,
            transport=transport,
            limits=limits,
            **kwargs,
        )

    # Agent methods
    def create_agent(
        self,
        name: str,
        description: Optional[str] = None,
        specs: Optional[str] = None,
    ) -> Agent:
        request = CreateAgentRequest(
            name=name,
            description=description,
            specs=specs,
        )
        response = self.post(
            url="/v1/agents",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Agent.model_validate(response.json())

    def get_agent(self, agent_id: UUID) -> Agent:
        response = self.get(f"/v1/agents/{agent_id}")
        _handle_error_response(response)
        return Agent.model_validate(response.json())

    def list_agents(self) -> AgentList:
        response = self.get("/v1/agents")
        _handle_error_response(response)
        return AgentList.model_validate(response.json())

    def update_agent(
        self,
        agent_id: UUID,
        name: Optional[str] = None,
        description: Optional[str] = None,
        specs: Optional[str] = None,
    ) -> Agent:
        request = UpdateAgentRequest(
            name=name,
            description=description,
            specs=specs,
        )
        response = self.put(
            url=f"/v1/agents/{agent_id}",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Agent.model_validate(response.json())

    def delete_agent(self, agent_id: UUID) -> None:
        response = self.delete(f"/v1/agents/{agent_id}")
        _handle_error_response(response)

    def add_permission_to_agent(
        self,
        agent_id: UUID,
        permission_id: UUID,
        assigned_by: Optional[UUID] = None,
    ) -> AgentPermissionMapping:
        request = AddPermissionToAgentRequest(
            permission_id=permission_id,
            assigned_by=assigned_by,
        )
        response = self.post(
            url=f"/v1/agents/{agent_id}/permissions",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return AgentPermissionMapping.model_validate(response.json())

    def list_agent_permissions(
        self,
        agent_id: UUID,
    ) -> AgentPermissionMappingList:
        response = self.get(f"/v1/agents/{agent_id}/permissions")
        _handle_error_response(response)
        return AgentPermissionMappingList.model_validate(response.json())

    def remove_permission_from_agent(
        self,
        agent_id: UUID,
        permission_id: UUID,
    ) -> None:
        response = self.delete(
            url=f"/v1/agents/{agent_id}/permissions/{permission_id}",
        )
        _handle_error_response(response)

    # Flow methods
    def create_flow(
        self,
        name: str,
        engine: str,
        entrypoint: str,
        code_location: str,
        parameters_schema: dict,
        tags: Optional[str] = None,
        description: Optional[str] = None,
        additional_info: Optional[dict] = None,
    ) -> Flow:
        request = CreateFlowRequest(
            name=name,
            description=description,
            engine=engine,
            entrypoint=entrypoint,
            code_location=code_location,
            parameters_schema=parameters_schema,
            additional_info=additional_info,
            tags=tags,
        )
        response = self.post(
            url="/v1/flows",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Flow.model_validate(response.json())

    def get_flow(self, flow_id: UUID) -> Flow:
        response = self.get(f"/v1/flows/{flow_id}")
        _handle_error_response(response)
        return Flow.model_validate(response.json())

    def list_flows(self, page: int = 1, per_page: int = 10) -> FlowList:
        params = {"page": page, "per_page": per_page}
        response = self.get("/v1/flows", params=params)
        _handle_error_response(response)
        return FlowList.model_validate(response.json())

    def update_flow(
        self,
        flow_id: UUID,
        name: Optional[str] = None,
        tags: Optional[str] = None,
        description: Optional[str] = None,
        engine: Optional[str] = None,
        entrypoint: Optional[str] = None,
        code_location: Optional[str] = None,
        additional_info: Optional[dict] = None,
    ) -> Flow:
        request = UpdateFlowRequest(
            name=name,
            description=description,
            engine=engine,
            entrypoint=entrypoint,
            code_location=code_location,
            additional_info=additional_info,
            tags=tags,
        )
        response = self.put(
            url=f"/v1/flows/{flow_id}",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Flow.model_validate(response.json())

    def delete_flow(self, flow_id: UUID) -> None:
        response = self.delete(f"/v1/flows/{flow_id}")
        _handle_error_response(response)

    def execute_flow(
        self,
        flow_id: UUID,
        parameters: dict,
    ) -> FlowRun:
        request = ExecuteFlowRequest(parameters=parameters)
        response = self.post(
            url=f"/v1/flows/{flow_id}/execute",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return FlowRun.model_validate(response.json())

    def get_flow_run(self, flow_run_id: UUID) -> FlowRun:
        response = self.get(f"/v1/flows/{flow_run_id}/status")
        _handle_error_response(response)
        return FlowRun.model_validate(response.json())

    # Task methods
    def create_task(
        self,
        thread_id: UUID,
        additional_info: Optional[dict] = None,
        max_request_loop: Optional[str] = None,
    ) -> Task:
        request = CreateTaskRequest(
            thread_id=thread_id,
            additional_info=additional_info or {},
            max_request_loop=max_request_loop,
        )
        response = self.post(
            url="/v1/tasks",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        task_data = response.json()
        if task_data.get("additional_info") is None:
            task_data["additional_info"] = {}
        return Task.model_validate(task_data)

    def get_task(self, task_id: str) -> Task:
        response = self.get(f"/v1/tasks/{task_id}")
        _handle_error_response(response)
        return Task.model_validate(response.json())

    def list_tasks(self, page: int = 1, per_page: int = 10) -> TaskList:
        params = {"page": page, "per_page": per_page}
        response = self.get("/v1/tasks", params=params)
        _handle_error_response(response)
        return TaskList.model_validate(response.json())

    def update_task(
        self,
        task_id: str,
        additional_info: Optional[dict] = None,
        max_request_loop: Optional[str] = None,
    ) -> Task:
        request = UpdateTaskRequest(
            additional_info=additional_info,
            max_request_loop=max_request_loop,
        )
        response = self.put(
            url=f"/v1/tasks/{task_id}",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Task.model_validate(response.json())

    def delete_task(self, task_id: str) -> None:
        response = self.delete(f"/v1/tasks/{task_id}")
        _handle_error_response(response)

    def execute_task(
        self,
        task_id: UUID,
        agent_id: UUID,
        current_loops: Optional[str] = None,
        stream: bool = False,
    ) -> Union[TaskRun, Iterator[Dict[str, Any]]]:
        request = ExecuteTaskRequest(
            agent_id=agent_id,
            current_loops=current_loops,
        )

        headers = {}
        if stream:
            headers["Accept"] = "text/event-stream"
            return self._execute_task_stream(task_id, request, headers)
        else:
            response = self.post(
                url=f"/v1/tasks/{task_id}/execute",
                json=request.model_dump(mode="json"),
                headers=headers,
                follow_redirects=True,
            )
            _handle_error_response(response)
            return TaskRun.model_validate(response.json())

    def _execute_task_stream(
        self,
        task_id: UUID,
        request: ExecuteTaskRequest,
        headers: Dict[str, str],
    ) -> Iterator[Dict[str, Any]]:
        """Handle streaming execution as a generator function."""
        with self.stream(
            method="POST",
            url=f"/v1/tasks/{task_id}/execute",
            json=request.model_dump(mode="json"),
            headers=headers,
            follow_redirects=True,
            timeout=60.0,  # 60 second timeout for streaming AI responses
        ) as response:
            _handle_error_response(response)
            buffer = ""
            decoder = codecs.getincrementaldecoder("utf-8")(errors="ignore")

            for chunk in response.iter_bytes(chunk_size=1024):
                if chunk:
                    # Use incremental decoder
                    # to handle partial UTF-8 characters
                    text = decoder.decode(chunk, False)
                    buffer += text

                    while "\n" in buffer:
                        line, buffer = buffer.split("\n", 1)
                        line = line.strip()
                        if line.startswith("event: "):
                            # Handle SSE event types (like heartbeat)
                            continue
                        elif line.startswith("data: "):
                            try:
                                json_data = line[6:]  # Remove "data: " prefix
                                if json_data.strip():  # Skip empty data lines
                                    data = json.loads(json_data)
                                    # Filter out heartbeat events
                                    if (
                                        isinstance(data, dict)
                                        and data.get("type") == "heartbeat"
                                    ):
                                        # Record heartbeat for connection
                                        self.record_heartbeat()
                                        continue
                                    yield data
                            except json.JSONDecodeError:
                                # Skip malformed JSON lines
                                continue

            # Handle any remaining data in decoder
            final_text = decoder.decode(b"", True)
            if final_text:
                buffer += final_text

    def list_task_runs(self, task_id: UUID) -> List[TaskRun]:
        response = self.get(f"/v1/tasks/{task_id}/runs")
        _handle_error_response(response)
        return [TaskRun.model_validate(run) for run in response.json()]

    def get_task_run(self, task_run_id: UUID) -> TaskRun:
        response = self.get(f"/v1/tasks/{task_run_id}/status")
        _handle_error_response(response)
        return TaskRun.model_validate(response.json())

    # Tool methods
    def create_tool(
        self,
        name: str,
        config: dict,
        description: Optional[str] = None,
    ) -> Tool:
        request = CreateToolRequest(
            name=name,
            description=description,
            config=config,
        )
        response = self.post(
            url="/v1/tools",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Tool.model_validate(response.json())

    def get_tool(self, tool_id: UUID) -> Tool:
        response = self.get(f"/v1/tools/{tool_id}")
        _handle_error_response(response)
        return Tool.model_validate(response.json())

    def list_tools(self) -> ToolList:
        response = self.get("/v1/tools")
        _handle_error_response(response)
        return ToolList.model_validate(response.json())

    def update_tool(
        self,
        tool_id: UUID,
        description: Optional[str] = None,
        config: Optional[dict] = None,
    ) -> Tool:
        request = UpdateToolRequest(
            description=description,
            config=config,
        )
        response = self.put(
            url=f"/v1/tools/{tool_id}",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Tool.model_validate(response.json())

    def delete_tool(self, tool_id: UUID) -> None:
        response = self.delete(f"/v1/tools/{tool_id}")
        _handle_error_response(response)

    # Message methods
    def create_message(
        self,
        thread_id: UUID,
        message: dict,
        recipient_id: UUID,
        sender_id: UUID,
    ) -> Message:
        request = CreateMessageRequest(
            message=message,
            recipient_id=recipient_id,
            sender_id=sender_id,
        )
        response = self.post(
            url=f"/v1/threads/{thread_id}/messages",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Message.model_validate(response.json())

    def get_message(self, thread_id: UUID, message_id: UUID) -> Message:
        response = self.get(f"/v1/threads/{thread_id}/messages/{message_id}")
        _handle_error_response(response)
        return Message.model_validate(response.json())

    def list_messages(self, thread_id: UUID) -> MessageList:
        response = self.get(f"/v1/threads/{thread_id}/messages")
        _handle_error_response(response)
        return MessageList.model_validate(response.json())

    def update_message(
        self,
        thread_id: UUID,
        message_id: UUID,
        message: dict,
    ) -> Message:
        request = UpdateMessageRequest(message=message)
        response = self.put(
            url=f"/v1/threads/{thread_id}/messages/{message_id}",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Message.model_validate(response.json())

    def delete_message(self, thread_id: UUID, message_id: UUID) -> None:
        response = self.delete(
            url=f"/v1/threads/{thread_id}/messages/{message_id}",
        )
        _handle_error_response(response)

    # Thread methods
    def create_thread(
        self,
        title: str,
        user_id: UUID,
    ) -> Thread:
        request = CreateThreadRequest(
            title=title,
            user_id=user_id,
        )
        response = self.post(
            url="/v1/threads",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Thread.model_validate(response.json())

    def get_thread(self, thread_id: UUID) -> Thread:
        response = self.get(f"/v1/threads/{thread_id}")
        _handle_error_response(response)
        return Thread.model_validate(response.json())

    def list_threads(self) -> ThreadList:
        response = self.get("/v1/threads")
        _handle_error_response(response)
        return ThreadList.model_validate(response.json())

    def update_thread(
        self,
        thread_id: UUID,
        title: str,
    ) -> Thread:
        request = UpdateThreadRequest(title=title)
        response = self.put(
            url=f"/v1/threads/{thread_id}",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Thread.model_validate(response.json())

    def delete_thread(self, thread_id: UUID) -> None:
        response = self.delete(f"/v1/threads/{thread_id}")
        _handle_error_response(response)

    # Permission methods
    def create_permission(
        self,
        name: str,
        content: dict,
        description: Optional[str] = None,
    ) -> Permission:
        request = CreatePermissionRequest(
            name=name,
            description=description,
            content=content,
        )
        response = self.post(
            url="/v1/permissions",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Permission.model_validate(response.json())

    def get_permission(self, permission_id: UUID) -> Permission:
        response = self.get(f"/v1/permissions/{permission_id}")
        _handle_error_response(response)
        return Permission.model_validate(response.json())

    def list_permissions(self) -> PermissionList:
        response = self.get("/v1/permissions")
        _handle_error_response(response)
        return PermissionList.model_validate(response.json())

    def update_permission(
        self,
        permission_id: UUID,
        name: Optional[str] = None,
        description: Optional[str] = None,
        content: Optional[dict] = None,
    ) -> Permission:
        request = UpdatePermissionRequest(
            name=name,
            description=description,
            content=content,
        )
        response = self.put(
            url=f"/v1/permissions/{permission_id}",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Permission.model_validate(response.json())

    def delete_permission(self, permission_id: UUID) -> None:
        response = self.delete(f"/v1/permissions/{permission_id}")
        _handle_error_response(response)

    # Role methods
    def create_role(
        self,
        name: str,
        description: Optional[str] = None,
        is_system_role: Optional[bool] = None,
    ) -> Role:
        request = CreateRoleRequest(
            name=name,
            description=description,
            is_system_role=is_system_role,
        )
        response = self.post(
            url="/v1/roles",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Role.model_validate(response.json())

    def get_role(self, role_id: UUID) -> Role:
        response = self.get(f"/v1/roles/{role_id}")
        _handle_error_response(response)
        return Role.model_validate(response.json())

    def list_roles(self) -> RoleList:
        response = self.get("/v1/roles")
        _handle_error_response(response)
        return RoleList.model_validate(response.json())

    def update_role(
        self,
        role_id: UUID,
        name: Optional[str] = None,
        description: Optional[str] = None,
        is_system_role: Optional[bool] = None,
    ) -> Role:
        request = UpdateRoleRequest(
            name=name,
            description=description,
            is_system_role=is_system_role,
        )
        response = self.put(
            url=f"/v1/roles/{role_id}",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Role.model_validate(response.json())

    def delete_role(self, role_id: UUID) -> None:
        response = self.delete(f"/v1/roles/{role_id}")
        _handle_error_response(response)

    def add_permission_to_role(
        self,
        role_id: UUID,
        permission_id: UUID,
        assigned_by: Optional[UUID] = None,
    ) -> RolePermissionMapping:
        request = AddPermissionToRoleRequest(
            permission_id=permission_id,
            assigned_by=assigned_by,
        )
        response = self.post(
            url=f"/v1/roles/{role_id}/permissions",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return RolePermissionMapping.model_validate(response.json())

    def list_permissions_for_role(
        self,
        role_id: UUID,
    ) -> List[RolePermissionMapping]:
        response = self.get(f"/v1/roles/{role_id}/permissions")
        _handle_error_response(response)
        return [
            RolePermissionMapping.model_validate(mapping)
            for mapping in response.json()  # noqa: E501
        ]

    def remove_permission_from_role(
        self,
        role_id: UUID,
        permission_id: UUID,
    ) -> None:
        response = self.delete(
            url=f"/v1/roles/{role_id}/permissions/{permission_id}",
        )
        _handle_error_response(response)

    # User methods
    def create_user(
        self,
        name: str,
        email: str,
        password_hash: str,
        provider_name: Optional[str] = None,
        additional_info: Optional[dict] = None,
    ) -> User:
        request = CreateUserRequest(
            name=name,
            email=email,
            password_hash=password_hash,
            provider_name=provider_name,
            additional_info=additional_info,
        )
        response = self.post(
            url="/v1/users",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return User.model_validate(response.json())

    def get_user(self, user_id: UUID) -> User:
        response = self.get(f"/v1/users/{user_id}")
        _handle_error_response(response)
        return User.model_validate(response.json())

    def list_users(self, page: int = 1, per_page: int = 10) -> UserList:
        params = {"page": page, "per_page": per_page}
        response = self.get("/v1/users", params=params)
        _handle_error_response(response)
        return UserList.model_validate(response.json())

    def update_user(
        self,
        user_id: UUID,
        username: Optional[str] = None,
        email: Optional[str] = None,
        provider_name: Optional[str] = None,
        additional_info: Optional[dict] = None,
    ) -> User:
        request = UpdateUserRequest(
            username=username,
            email=email,
            provider_name=provider_name,
            additional_info=additional_info,
        )
        response = self.put(
            url=f"/v1/users/{user_id}",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return User.model_validate(response.json())

    def delete_user(self, user_id: UUID) -> None:
        response = self.delete(f"/v1/users/{user_id}")
        _handle_error_response(response)

    def add_role_to_user(
        self,
        user_id: UUID,
        role_id: UUID,
        assigned_by: Optional[UUID] = None,
    ) -> UserRoleMapping:
        request = AddRoleToUserRequest(
            role_id=role_id,
            assigned_by=assigned_by,
        )
        response = self.post(
            url=f"/v1/users/{user_id}/roles",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return UserRoleMapping.model_validate(response.json())

    def list_roles_for_user(self, user_id: UUID) -> List[UserRoleMapping]:
        response = self.get(f"/v1/users/{user_id}/roles")
        _handle_error_response(response)
        return [
            UserRoleMapping.model_validate(mapping)
            for mapping in response.json()  # noqa: E501
        ]

    def remove_role_from_user(
        self,
        user_id: UUID,
        role_id: UUID,
    ) -> None:
        response = self.delete(f"/v1/users/{user_id}/roles/{role_id}")
        _handle_error_response(response)

    def record_heartbeat(self, connection_id: str = "default"):
        """Record heartbeat for connection health monitoring."""
        self._last_heartbeat[connection_id] = time.time()
        self._connection_stats["heartbeats_received"] += 1


class AsyncClient(httpx.AsyncClient):
    def __init__(
        self,
        base_url: Optional[str] = "http://localhost:8080",
        headers: Optional[Dict[str, str]] = None,
        timeout: Optional[float] = None,
        streaming_timeout: Optional[float] = None,
        max_connections: int = 20,
        max_keepalive_connections: int = 10,
        keepalive_expiry: float = 30.0,
        **kwargs: Any,
    ):
        # Configure connection limits for better streaming performance
        limits = httpx.Limits(
            max_connections=max_connections,
            max_keepalive_connections=max_keepalive_connections,
            keepalive_expiry=keepalive_expiry,
        )

        # Configure transport with TCP keep-alive and retries
        transport = httpx.AsyncHTTPTransport(
            limits=limits,
            retries=3,  # Automatic retries for failed connections
        )

        # Set default timeout if not provided
        if timeout is not None:
            kwargs["timeout"] = timeout

        # Store streaming timeout for SSE operations
        self._streaming_timeout = streaming_timeout or 120.0

        # Enhanced headers for better streaming
        default_headers = {
            "Connection": "keep-alive",
            "Keep-Alive": "timeout=300, max=1000",
            "User-Agent": "pinazu-py/1.0",
        }
        if headers:
            default_headers.update(headers)

        # Connection health monitoring
        self._last_heartbeat = {}
        self._connection_stats = {
            "total_requests": 0,
            "streaming_requests": 0,
            "failed_connections": 0,
            "heartbeats_received": 0,
        }

        super().__init__(
            base_url=base_url,
            headers=default_headers,
            transport=transport,
            limits=limits,
            **kwargs,
        )

    # Agent methods
    async def create_agent(
        self,
        name: str,
        description: Optional[str] = None,
        specs: Optional[str] = None,
    ) -> Agent:
        request = CreateAgentRequest(
            name=name,
            description=description,
            specs=specs,
        )
        response = await self.post(
            url="/v1/agents",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Agent.model_validate(response.json())

    async def get_agent(self, agent_id: UUID) -> Agent:
        response = await self.get(f"/v1/agents/{agent_id}")
        _handle_error_response(response)
        return Agent.model_validate(response.json())

    async def list_agents(self) -> AgentList:
        response = await self.get("/v1/agents")
        _handle_error_response(response)
        return AgentList.model_validate(response.json())

    async def update_agent(
        self,
        agent_id: UUID,
        name: Optional[str] = None,
        description: Optional[str] = None,
        specs: Optional[str] = None,
    ) -> Agent:
        request = UpdateAgentRequest(
            name=name,
            description=description,
            specs=specs,
        )
        response = await self.put(
            url=f"/v1/agents/{agent_id}",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Agent.model_validate(response.json())

    async def delete_agent(self, agent_id: UUID) -> None:
        response = await self.delete(f"/v1/agents/{agent_id}")
        _handle_error_response(response)

    async def add_permission_to_agent(
        self,
        agent_id: UUID,
        permission_id: UUID,
        assigned_by: Optional[UUID] = None,
    ) -> AgentPermissionMapping:
        request = AddPermissionToAgentRequest(
            permission_id=permission_id,
            assigned_by=assigned_by,
        )
        response = await self.post(
            url=f"/v1/agents/{agent_id}/permissions",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return AgentPermissionMapping.model_validate(response.json())

    async def list_agent_permissions(
        self,
        agent_id: UUID,
    ) -> AgentPermissionMappingList:
        response = await self.get(f"/v1/agents/{agent_id}/permissions")
        _handle_error_response(response)
        return AgentPermissionMappingList.model_validate(response.json())

    async def remove_permission_from_agent(
        self,
        agent_id: UUID,
        permission_id: UUID,
    ) -> None:
        response = await self.delete(
            url=f"/v1/agents/{agent_id}/permissions/{permission_id}",
        )
        _handle_error_response(response)

    # Flow methods
    async def create_flow(
        self,
        name: str,
        engine: str,
        entrypoint: str,
        code_location: str,
        parameters_schema: dict,
        description: Optional[str] = None,
        additional_info: Optional[dict] = None,
        tags: Optional[str] = None,
    ) -> Flow:
        request = CreateFlowRequest(
            name=name,
            description=description,
            engine=engine,
            entrypoint=entrypoint,
            code_location=code_location,
            parameters_schema=parameters_schema,
            additional_info=additional_info,
            tags=tags,
        )
        response = await self.post(
            url="/v1/flows",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Flow.model_validate(response.json())

    async def get_flow(self, flow_id: UUID) -> Flow:
        response = await self.get(f"/v1/flows/{flow_id}")
        _handle_error_response(response)
        return Flow.model_validate(response.json())

    async def list_flows(self, page: int = 1, per_page: int = 10) -> FlowList:
        params = {"page": page, "per_page": per_page}
        response = await self.get("/v1/flows", params=params)
        _handle_error_response(response)
        return FlowList.model_validate(response.json())

    async def update_flow(
        self,
        flow_id: UUID,
        name: Optional[str] = None,
        description: Optional[str] = None,
        engine: Optional[str] = None,
        entrypoint: Optional[str] = None,
        code_location: Optional[str] = None,
        additional_info: Optional[dict] = None,
        tags: Optional[str] = None,
    ) -> Flow:
        request = UpdateFlowRequest(
            name=name,
            description=description,
            engine=engine,
            entrypoint=entrypoint,
            code_location=code_location,
            additional_info=additional_info,
            tags=tags,
        )
        response = await self.put(
            url=f"/v1/flows/{flow_id}",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Flow.model_validate(response.json())

    async def delete_flow(self, flow_id: UUID) -> None:
        response = await self.delete(f"/v1/flows/{flow_id}")
        _handle_error_response(response)

    async def execute_flow(
        self,
        flow_id: UUID,
        parameters: dict,
    ) -> FlowRun:
        request = ExecuteFlowRequest(parameters=parameters)
        response = await self.post(
            url=f"/v1/flows/{flow_id}/execute",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return FlowRun.model_validate(response.json())

    async def get_flow_run(self, flow_run_id: UUID) -> FlowRun:
        response = await self.get(f"/v1/flows/{flow_run_id}/status")
        _handle_error_response(response)
        return FlowRun.model_validate(response.json())

    # Task methods
    async def create_task(
        self,
        thread_id: UUID,
        additional_info: Optional[dict] = None,
        max_request_loop: Optional[int] = None,
    ) -> Task:
        request = CreateTaskRequest(
            thread_id=thread_id,
            additional_info=additional_info or {},
            max_request_loop=(
                f"{max_request_loop}" if max_request_loop is not None else None
            ),
        )
        response = await self.post(
            url="/v1/tasks",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        task_data = response.json()
        if task_data.get("additional_info") is None:
            task_data["additional_info"] = {}
        return Task.model_validate(task_data)

    async def get_task(self, task_id: UUID) -> Task:
        response = await self.get(f"/v1/tasks/{task_id}")
        _handle_error_response(response)
        return Task.model_validate(response.json())

    async def list_tasks(self, page: int = 1, per_page: int = 10) -> TaskList:
        params = {"page": page, "per_page": per_page}
        response = await self.get("/v1/tasks", params=params)
        _handle_error_response(response)
        return TaskList.model_validate(response.json())

    async def update_task(
        self,
        task_id: UUID,
        additional_info: Optional[str] = None,
        max_request_loop: Optional[int] = None,
    ) -> Task:
        request = UpdateTaskRequest(
            additional_info=additional_info,
            max_request_loop=f"{max_request_loop}",
        )
        response = await self.put(
            url=f"/v1/tasks/{task_id}",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Task.model_validate(response.json())

    async def delete_task(self, task_id: UUID) -> None:
        response = await self.delete(f"/v1/tasks/{task_id}")
        _handle_error_response(response)

    async def execute_task(
        self,
        task_id: UUID,
        agent_id: UUID,
        current_loops: Optional[str] = None,
        stream: bool = False,
        streaming_timeout: Optional[float] = None,
    ) -> Union[TaskRun, AsyncIterator[Dict[str, Any]]]:
        request = ExecuteTaskRequest(
            agent_id=agent_id,
            current_loops=current_loops,
        )

        headers = {}
        if stream:
            headers["Accept"] = "text/event-stream"

        if stream:
            return self._execute_task_stream(
                task_id, request, headers, streaming_timeout
            )
        else:
            response = await self.post(
                url=f"/v1/tasks/{task_id}/execute",
                json=request.model_dump(mode="json"),
                headers=headers,
                follow_redirects=True,
            )
            _handle_error_response(response)
            return TaskRun.model_validate(response.json())

    async def _execute_task_stream(
        self,
        task_id: UUID,
        request: ExecuteTaskRequest,
        headers: Dict[str, str],
        streaming_timeout: Optional[float] = None,
    ) -> AsyncIterator[Dict[str, Any]]:
        """Handle streaming execution as an async generator function."""
        # Use provided timeout, instance timeout, or default
        timeout_value = streaming_timeout or self._streaming_timeout
        async with self.stream(
            method="POST",
            url=f"/v1/tasks/{task_id}/execute",
            json=request.model_dump(mode="json"),
            headers=headers,
            follow_redirects=True,
            timeout=timeout_value,
        ) as response:
            _handle_error_response(response)
            buffer = ""
            decoder = codecs.getincrementaldecoder("utf-8")(errors="ignore")

            async for chunk in response.aiter_bytes(chunk_size=1024):
                if chunk:
                    # Use incremental decoder
                    # to handle partial UTF-8 characters
                    text = decoder.decode(chunk, False)
                    buffer += text

                    while "\n" in buffer:
                        line, buffer = buffer.split("\n", 1)
                        line = line.strip()
                        if line.startswith("event: "):
                            continue
                        elif line.startswith("data: "):
                            try:
                                json_data = line[6:]  # Remove "data: " prefix
                                if json_data.strip():  # Skip empty data lines
                                    data = json.loads(json_data)
                                    # Filter out heartbeat events
                                    if (
                                        isinstance(data, dict)
                                        and data.get("type") == "heartbeat"
                                    ):
                                        # Record heartbeat for connection
                                        self.record_heartbeat()
                                        continue
                                    yield data
                            except json.JSONDecodeError:
                                # Skip malformed JSON lines
                                continue

            # Handle any remaining data in decoder
            final_text = decoder.decode(b"", True)
            if final_text:
                buffer += final_text

    async def list_task_runs(self, task_id: UUID) -> List[TaskRun]:
        response = await self.get(f"/v1/tasks/{task_id}/runs")
        _handle_error_response(response)
        return [TaskRun.model_validate(run) for run in response.json()]

    async def get_task_run(self, task_run_id: UUID) -> TaskRun:
        response = await self.get(f"/v1/tasks/{task_run_id}/status")
        _handle_error_response(response)
        return TaskRun.model_validate(response.json())

    # Advanced streaming methods
    async def stream_with_retry(
        self,
        method: str,
        url: str,
        max_retries: int = 3,
        retry_delay: float = 1.0,
        **kwargs,
    ) -> httpx.Response:
        """
        Stream with automatic retry logic for failed connections.

        Args:
            method: HTTP method
            url: Request URL
            max_retries: Maximum number of retries
            retry_delay: Delay between retries
            **kwargs: Additional request arguments

        Returns:
            Response object for streaming
        """

        for attempt in range(max_retries + 1):
            try:
                response = await self.stream(method, url, **kwargs)
                self._connection_stats["streaming_requests"] += 1
                return response

            except (httpx.ConnectError, httpx.TimeoutException) as e:
                self._connection_stats["failed_connections"] += 1

                if attempt == max_retries:
                    raise e

                # Exponential backoff
                await asyncio.sleep(retry_delay * (2**attempt))

        raise RuntimeError("Unexpected error in stream_with_retry")

    def record_heartbeat(self, connection_id: str = "default"):
        """Record heartbeat for connection health monitoring."""
        self._last_heartbeat[connection_id] = time.time()
        self._connection_stats["heartbeats_received"] += 1

    def get_connection_health(
        self,
        connection_id: str = "default",
    ) -> Dict[str, Any]:
        """Get connection health status."""
        last_heartbeat = self._last_heartbeat.get(connection_id)
        current_time = time.time()

        if last_heartbeat:
            seconds_since_heartbeat = current_time - last_heartbeat
            is_healthy = (
                seconds_since_heartbeat < 60
            )  # Consider unhealthy if no heartbeat for 60s
        else:
            seconds_since_heartbeat = None
            is_healthy = False

        return {
            "connection_id": connection_id,
            "is_healthy": is_healthy,
            "last_heartbeat": last_heartbeat,
            "seconds_since_heartbeat": seconds_since_heartbeat,
            "stats": self._connection_stats.copy(),
        }

    async def request(self, method: str, url: str, **kwargs) -> httpx.Response:
        """Override request to track connection statistics."""
        self._connection_stats["total_requests"] += 1
        return await super().request(method, url, **kwargs)

    # Tool methods
    async def create_tool(
        self,
        name: str,
        config: dict,
        description: Optional[str] = None,
    ) -> Tool:
        request = CreateToolRequest(
            name=name,
            description=description,
            config=config,
        )
        response = await self.post(
            url="/v1/tools",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Tool.model_validate(response.json())

    async def get_tool(self, tool_id: UUID) -> Tool:
        response = await self.get(f"/v1/tools/{tool_id}")
        _handle_error_response(response)
        return Tool.model_validate(response.json())

    async def list_tools(self) -> ToolList:
        response = await self.get("/v1/tools")
        _handle_error_response(response)
        return ToolList.model_validate(response.json())

    async def update_tool(
        self,
        tool_id: UUID,
        description: Optional[str] = None,
        config: Optional[dict] = None,
    ) -> Tool:
        request = UpdateToolRequest(
            description=description,
            config=config,
        )
        response = await self.put(
            url=f"/v1/tools/{tool_id}",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Tool.model_validate(response.json())

    async def delete_tool(self, tool_id: UUID) -> None:
        response = await self.delete(f"/v1/tools/{tool_id}")
        _handle_error_response(response)

    # Message methods
    async def create_message(
        self,
        thread_id: UUID,
        message: dict,
        recipient_id: UUID,
        sender_id: UUID,
    ) -> Message:
        request = CreateMessageRequest(
            message=message,
            recipient_id=recipient_id,
            sender_id=sender_id,
        )
        response = await self.post(
            url=f"/v1/threads/{thread_id}/messages",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Message.model_validate(response.json())

    async def get_message(self, thread_id: UUID, message_id: UUID) -> Message:
        response = await self.get(
            url=f"/v1/threads/{thread_id}/messages/{message_id}",
        )
        _handle_error_response(response)
        return Message.model_validate(response.json())

    async def list_messages(self, thread_id: UUID) -> MessageList:
        response = await self.get(f"/v1/threads/{thread_id}/messages")
        _handle_error_response(response)
        return MessageList.model_validate(response.json())

    async def update_message(
        self,
        thread_id: UUID,
        message_id: UUID,
        message: dict,
    ) -> Message:
        request = UpdateMessageRequest(message=message)
        response = await self.put(
            url=f"/v1/threads/{thread_id}/messages/{message_id}",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Message.model_validate(response.json())

    async def delete_message(self, thread_id: UUID, message_id: UUID) -> None:
        response = await self.delete(
            url=f"/v1/threads/{thread_id}/messages/{message_id}",
        )
        _handle_error_response(response)

    # Thread methods
    async def create_thread(self, title: str, user_id: UUID) -> Thread:
        request = CreateThreadRequest(
            title=title,
            user_id=user_id,
        )
        response = await self.post(
            url="/v1/threads",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Thread.model_validate(response.json())

    async def get_thread(self, thread_id: UUID) -> Thread:
        response = await self.get(f"/v1/threads/{thread_id}")
        _handle_error_response(response)
        return Thread.model_validate(response.json())

    async def list_threads(self) -> ThreadList:
        response = await self.get("/v1/threads")
        _handle_error_response(response)
        return ThreadList.model_validate(response.json())

    async def update_thread(self, thread_id: UUID, title: str) -> Thread:
        request = UpdateThreadRequest(title=title)
        response = await self.put(
            url=f"/v1/threads/{thread_id}",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Thread.model_validate(response.json())

    async def delete_thread(self, thread_id: UUID) -> None:
        response = await self.delete(f"/v1/threads/{thread_id}")
        _handle_error_response(response)

    # Permission methods
    async def create_permission(
        self,
        name: str,
        content: dict,
        description: Optional[str] = None,
    ) -> Permission:
        request = CreatePermissionRequest(
            name=name,
            description=description,
            content=content,
        )
        response = await self.post(
            url="/v1/permissions",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Permission.model_validate(response.json())

    async def get_permission(self, permission_id: UUID) -> Permission:
        response = await self.get(f"/v1/permissions/{permission_id}")
        _handle_error_response(response)
        return Permission.model_validate(response.json())

    async def list_permissions(self) -> PermissionList:
        response = await self.get("/v1/permissions")
        _handle_error_response(response)
        return PermissionList.model_validate(response.json())

    async def update_permission(
        self,
        id: UUID,
        name: Optional[str] = None,
        description: Optional[str] = None,
        content: Optional[dict] = None,
    ) -> Permission:
        request = UpdatePermissionRequest(
            name=name,
            description=description,
            content=content,
        )
        response = await self.put(
            url=f"/v1/permissions/{id}",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Permission.model_validate(response.json())

    async def delete_permission(self, permission_id: UUID) -> None:
        response = await self.delete(f"/v1/permissions/{permission_id}")
        _handle_error_response(response)

    # Role methods
    async def create_role(
        self,
        name: str,
        description: Optional[str] = None,
        is_system_role: Optional[bool] = None,
    ) -> Role:
        request = CreateRoleRequest(
            name=name,
            description=description,
            is_system_role=is_system_role,
        )
        response = await self.post(
            url="/v1/roles",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Role.model_validate(response.json())

    async def get_role(self, role_id: UUID) -> Role:
        response = await self.get(f"/v1/roles/{role_id}")
        _handle_error_response(response)
        return Role.model_validate(response.json())

    async def list_roles(self) -> RoleList:
        response = await self.get("/v1/roles")
        _handle_error_response(response)
        return RoleList.model_validate(response.json())

    async def update_role(
        self,
        role_id: UUID,
        name: Optional[str] = None,
        description: Optional[str] = None,
        is_system_role: Optional[bool] = None,
    ) -> Role:
        request = UpdateRoleRequest(
            name=name,
            description=description,
            is_system_role=is_system_role,
        )
        response = await self.put(
            url=f"/v1/roles/{role_id}",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return Role.model_validate(response.json())

    async def delete_role(self, role_id: UUID) -> None:
        response = await self.delete(f"/v1/roles/{role_id}")
        _handle_error_response(response)

    async def add_permission_to_role(
        self,
        role_id: UUID,
        permission_id: UUID,
        assigned_by: Optional[UUID] = None,
    ) -> RolePermissionMapping:
        request = AddPermissionToRoleRequest(
            permission_id=permission_id,
            assigned_by=assigned_by,
        )
        response = await self.post(
            url=f"/v1/roles/{role_id}/permissions",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return RolePermissionMapping.model_validate(response.json())

    async def list_permissions_for_role(
        self, role_id: UUID
    ) -> List[RolePermissionMapping]:
        response = await self.get(f"/v1/roles/{role_id}/permissions")
        _handle_error_response(response)
        return [
            RolePermissionMapping.model_validate(mapping)
            for mapping in response.json()  # noqa: E501
        ]

    async def remove_permission_from_role(
        self,
        role_id: UUID,
        permission_id: UUID,
    ) -> None:
        response = await self.delete(
            url=f"/v1/roles/{role_id}/permissions/{permission_id}",
        )
        _handle_error_response(response)

    # User methods
    async def create_user(
        self,
        name: str,
        email: str,
        password_hash: str,
        provider_name: Optional[str] = None,
        additional_info: Optional[dict] = None,
    ) -> User:
        request = CreateUserRequest(
            name=name,
            email=email,
            password_hash=password_hash,
            provider_name=provider_name,
            additional_info=additional_info,
        )
        response = await self.post(
            url="/v1/users",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return User.model_validate(response.json())

    async def get_user(self, user_id: UUID) -> User:
        response = await self.get(f"/v1/users/{user_id}")
        _handle_error_response(response)
        return User.model_validate(response.json())

    async def list_users(self, page: int = 1, per_page: int = 10) -> UserList:
        params = {"page": page, "per_page": per_page}
        response = await self.get("/v1/users", params=params)
        _handle_error_response(response)
        return UserList.model_validate(response.json())

    async def update_user(
        self,
        user_id: UUID,
        username: Optional[str] = None,
        email: Optional[str] = None,
        provider_name: Optional[str] = None,
        additional_info: Optional[dict] = None,
    ) -> User:
        request = UpdateUserRequest(
            username=username,
            email=email,
            provider_name=provider_name,
            additional_info=additional_info,
        )
        response = await self.put(
            url=f"/v1/users/{user_id}",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return User.model_validate(response.json())

    async def delete_user(self, user_id: UUID) -> None:
        response = await self.delete(f"/v1/users/{user_id}")
        _handle_error_response(response)

    async def add_role_to_user(
        self,
        user_id: UUID,
        role_id: UUID,
        assigned_by: Optional[UUID] = None,
    ) -> UserRoleMapping:
        request = AddRoleToUserRequest(
            role_id=role_id,
            assigned_by=assigned_by,
        )
        response = await self.post(
            url=f"/v1/users/{user_id}/roles",
            json=request.model_dump(mode="json"),
        )
        _handle_error_response(response)
        return UserRoleMapping.model_validate(response.json())

    async def list_roles_for_user(
        self,
        user_id: UUID,
    ) -> List[UserRoleMapping]:
        response = await self.get(f"/v1/users/{user_id}/roles")
        _handle_error_response(response)
        return [
            UserRoleMapping.model_validate(mapping)
            for mapping in response.json()  # noqa: E501
        ]

    async def remove_role_from_user(
        self,
        user_id: UUID,
        role_id: UUID,
    ) -> None:
        response = await self.delete(f"/v1/users/{user_id}/roles/{role_id}")
        _handle_error_response(response)
