from pydantic import BaseModel
from uuid import UUID
from datetime import datetime
from typing import Optional


class AddPermissionToAgentRequest(BaseModel):
    assigned_by: Optional[UUID] = None
    permission_id: UUID
    

class AddPermissionToRoleRequest(BaseModel):
    assigned_by: Optional[UUID] = None
    permission_id: UUID
    

class AddRoleToUserRequest(BaseModel):
    assigned_by: Optional[UUID] = None
    role_id: UUID
    

class Agent(BaseModel):
    created_at: datetime
    created_by: UUID
    description: Optional[str] = None
    id: UUID
    name: str
    specs: Optional[str] = None
    updated_at: datetime
    

class AgentList(BaseModel):
    page: int
    per_page: int
    total: int
    total_pages: int
    agents: list[Agent]

class AgentPermissionMapping(BaseModel):
    agent_id: UUID
    assigned_at: datetime
    assigned_by: UUID
    mapping_id: UUID
    permission_id: UUID
    

class AgentPermissionMappingList(BaseModel):
    page: int
    per_page: int
    total: int
    total_pages: int
    permissionMappings: list[AgentPermissionMapping]

class BadRequest(BaseModel):
    message: str
    

class CreateAgentRequest(BaseModel):
    description: Optional[str] = None
    name: str
    specs: Optional[str] = None
    

class CreateFlowRequest(BaseModel):
    additional_info: Optional[dict] = None
    code_location: str
    description: Optional[str] = None
    engine: str
    entrypoint: str
    name: str
    parameters_schema: dict
    tags: Optional[list] = None
    

class CreateMessageRequest(BaseModel):
    message: dict
    recipient_id: UUID
    sender_id: UUID
    

class CreatePermissionRequest(BaseModel):
    content: dict
    description: Optional[str] = None
    name: str
    

class CreateRoleRequest(BaseModel):
    description: Optional[str] = None
    is_system_role: Optional[bool] = None
    name: str
    

class CreateTaskRequest(BaseModel):
    additional_info: Optional[dict] = None
    max_request_loop: Optional[int] = None
    thread_id: UUID
    

class CreateThreadRequest(BaseModel):
    title: str
    user_id: UUID
    

class CreateToolRequest(BaseModel):
    config: dict
    description: Optional[str] = None
    name: str
    

class CreateUserRequest(BaseModel):
    additional_info: Optional[dict] = None
    email: str
    name: str
    password_hash: str
    provider_name: Optional[str] = None
    

class ExecuteFlowRequest(BaseModel):
    parameters: dict
    

class ExecuteTaskRequest(BaseModel):
    agent_id: UUID
    current_loops: Optional[int] = None
    

class Flow(BaseModel):
    additional_info: Optional[dict] = None
    code_location: Optional[str] = None
    created_at: datetime
    description: Optional[str] = None
    engine: str
    entrypoint: Optional[str] = None
    id: UUID
    name: str
    parameters_schema: dict
    tags: list
    updated_at: datetime
    

class FlowList(BaseModel):
    page: int
    per_page: int
    total: int
    total_pages: int
    flows: list[Flow]

class FlowRun(BaseModel):
    created_at: datetime
    engine: str
    error_message: Optional[str] = None
    finished_at: Optional[datetime] = None
    flow_id: UUID
    flow_run_id: UUID
    max_retries: Optional[int] = None
    parameters: dict
    retry_count: Optional[int] = None
    started_at: Optional[datetime] = None
    status: str
    success_task_results: dict
    task_statuses: dict
    updated_at: datetime
    

class MCPTool(BaseModel):
    api_key: Optional[str] = None
    entrypoint: str
    env_vars: Optional[dict] = None
    protocol: str
    type: str
    

class Message(BaseModel):
    citations: Optional[list] = None
    created_at: datetime
    id: UUID
    message: dict
    recipient_id: UUID
    result_type: Optional[str] = None
    sender_id: UUID
    sender_type: str
    stop_reason: Optional[str] = None
    thread_id: UUID
    updated_at: datetime
    

class MessageList(BaseModel):
    page: int
    per_page: int
    total: int
    total_pages: int
    messages: list[Message]

class MockToolRequest(BaseModel):
    input: str
    

class MockToolResponse(BaseModel):
    citation: list
    text: str
    

class MockToolWithDelayRequest(BaseModel):
    input: str
    

class MockToolWithDelayResponse(BaseModel):
    citation: list
    text: str
    

class NotFound(BaseModel):
    id: UUID
    message: str
    resource: str
    

class PaginationMeta(BaseModel):
    page: int
    per_page: int
    total: int
    total_pages: int
    

class Permission(BaseModel):
    content: dict
    created_at: datetime
    description: Optional[str] = None
    id: UUID
    name: str
    updated_at: datetime
    

class PermissionList(BaseModel):
    page: int
    per_page: int
    total: int
    total_pages: int
    permissions: list[Permission]

class ResourceAlreadyExists(BaseModel):
    id: UUID
    message: str
    resource: str
    

class Role(BaseModel):
    created_at: datetime
    description: Optional[str] = None
    id: Optional[UUID] = None
    is_system_role: Optional[bool] = None
    name: Optional[str] = None
    updated_at: datetime
    

class RoleList(BaseModel):
    page: int
    per_page: int
    total: int
    total_pages: int
    roles: list[Role]

class RolePermissionMapping(BaseModel):
    assigned_at: datetime
    assigned_by: UUID
    mapping_id: UUID
    permission_id: UUID
    role_id: UUID
    

class StandaloneTool(BaseModel):
    api_key: Optional[str] = None
    params: dict
    type: str
    url: str
    

class Task(BaseModel):
    additional_info: dict
    created_at: datetime
    created_by: UUID
    id: str
    max_request_loop: int
    parent_task_id: Optional[str] = None
    thread_id: UUID
    updated_at: datetime
    

class TaskList(BaseModel):
    page: int
    per_page: int
    total: int
    total_pages: int
    tasks: list[Task]

class TaskRun(BaseModel):
    created_at: datetime
    current_loops: Optional[int] = None
    finished_at: Optional[datetime] = None
    started_at: Optional[datetime] = None
    status: str
    task_id: str
    task_run_id: UUID
    updated_at: datetime
    

class Thread(BaseModel):
    created_at: datetime
    id: UUID
    title: str
    updated_at: datetime
    user_id: UUID
    

class ThreadList(BaseModel):
    page: int
    per_page: int
    total: int
    total_pages: int
    threads: list[Thread]

class Tool(BaseModel):
    config: dict
    created_at: datetime
    created_by: UUID
    description: Optional[str] = None
    id: UUID
    name: str
    updated_at: datetime
    

class ToolList(BaseModel):
    page: int
    per_page: int
    total: int
    total_pages: int
    tools: list[Tool]

class UpdateAgentRequest(BaseModel):
    description: Optional[str] = None
    name: Optional[str] = None
    specs: Optional[str] = None
    

class UpdateFlowRequest(BaseModel):
    additional_info: Optional[dict] = None
    code_location: Optional[str] = None
    description: Optional[str] = None
    engine: Optional[str] = None
    entrypoint: Optional[str] = None
    name: Optional[str] = None
    tags: Optional[list] = None
    

class UpdateMessageRequest(BaseModel):
    message: dict
    

class UpdatePermissionRequest(BaseModel):
    content: Optional[dict] = None
    description: Optional[str] = None
    name: Optional[str] = None
    

class UpdateRoleRequest(BaseModel):
    description: Optional[str] = None
    is_system_role: Optional[bool] = None
    name: Optional[str] = None
    

class UpdateTaskRequest(BaseModel):
    additional_info: Optional[dict] = None
    max_request_loop: Optional[int] = None
    

class UpdateThreadRequest(BaseModel):
    title: str
    

class UpdateToolRequest(BaseModel):
    config: Optional[dict] = None
    description: Optional[str] = None
    

class UpdateUserRequest(BaseModel):
    additional_info: Optional[dict] = None
    email: Optional[str] = None
    provider_name: Optional[str] = None
    username: Optional[str] = None
    

class User(BaseModel):
    additional_info: Optional[dict] = None
    created_at: datetime
    email: str
    id: UUID
    is_online: Optional[bool] = None
    last_login: Optional[datetime] = None
    name: str
    provider_name: Optional[str] = None
    updated_at: datetime
    

class UserList(BaseModel):
    page: int
    per_page: int
    total: int
    total_pages: int
    users: list[User]

class UserRoleMapping(BaseModel):
    assigned_at: datetime
    assigned_by: UUID
    mapping_id: UUID
    role_id: UUID
    user_id: UUID
    

class WorkflowTool(BaseModel):
    params: dict
    s3_url: str
    type: str
    
