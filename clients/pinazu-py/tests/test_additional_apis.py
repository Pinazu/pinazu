"""
Test suite for additional API client methods
(Users, Roles, Permissions, Tools, etc.).
"""

from uuid import UUID
from unittest.mock import patch
from pinazu.api.models_generated import (
    User,
    UserList,
    Role,
    RoleList,
    Permission,
    PermissionList,
    Tool,
    ToolList,
    Thread,
    ThreadList,
    Message,
    MessageList,
    UserRoleMapping,
    RolePermissionMapping,
)


class TestUsersAPI:
    """Test class for Users API methods."""

    def test_create_user(self, client, test_user_data, mock_responses):
        """Test creating a new user."""
        expected_user = User(
            id=UUID("12345678-1234-1234-1234-123456789012"),
            name=test_user_data["name"],
            email=test_user_data["email"],
            password_hash=test_user_data["password_hash"],
            provider_name=test_user_data["provider_name"],
            additional_info=test_user_data["additional_info"],
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:00:00Z",
        )

        mock_response = mock_responses(
            expected_user.model_dump(mode="json"),
            201,
        )

        with patch.object(
            client,
            "post",
            return_value=mock_response,
        ) as mock_post:  # noqa: E501
            result = client.create_user(**test_user_data)

            mock_post.assert_called_once()
            call_args = mock_post.call_args
            assert call_args[1]["url"] == "/v1/users"

            assert result.id == expected_user.id
            assert result.email == test_user_data["email"]
            assert result.name == test_user_data["name"]

    def test_list_users_with_pagination(self, client, mock_responses):
        """Test listing users with pagination."""
        users_data = UserList(
            users=[
                User(
                    id=UUID("12345678-1234-1234-1234-123456789012"),
                    name="User 1",
                    email="user1@test.com",
                    created_at="2025-01-01T00:00:00Z",
                    updated_at="2025-01-01T00:00:00Z",
                )
            ],
            total=1,
            page=1,
            per_page=10,
            total_pages=1,
        )

        mock_response = mock_responses(users_data.model_dump(mode="json"))

        with patch.object(
            client,
            "get",
            return_value=mock_response,
        ) as mock_get:  # noqa: E501
            result = client.list_users(page=1, per_page=10)

            mock_get.assert_called_once_with(
                "/v1/users", params={"page": 1, "per_page": 10}
            )
            assert result.total == 1
            assert result.page == 1
            assert result.per_page == 10
            assert len(result.users) == 1

    def test_update_user(self, client, sample_uuid, mock_responses):
        """Test updating a user."""
        updated_user = User(
            id=sample_uuid,
            name="Updated User",
            email="updated@test.com",
            password_hash="updated_hash",
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T01:00:00Z",
        )

        mock_response = mock_responses(updated_user.model_dump(mode="json"))

        with patch.object(client, "put", return_value=mock_response) as _:
            result = client.update_user(
                user_id=sample_uuid,
                email="updated@test.com",
            )

            assert result.name == "Updated User"
            assert result.email == "updated@test.com"


class TestRolesAPI:
    """Test class for Roles API methods."""

    def test_create_role(self, client, test_role_data, mock_responses):
        """Test creating a new role."""
        expected_role = Role(
            id=UUID("12345678-1234-1234-1234-123456789012"),
            name=test_role_data["name"],
            description=test_role_data["description"],
            is_system_role=test_role_data["is_system_role"],
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:00:00Z",
        )

        mock_response = mock_responses(
            expected_role.model_dump(mode="json"),
            201,
        )

        with patch.object(client, "post", return_value=mock_response):
            result = client.create_role(**test_role_data)

            assert result.name == test_role_data["name"]
            assert result.is_system_role == test_role_data["is_system_role"]

    def test_list_roles(self, client, mock_responses):
        """Test listing roles."""
        roles_data = RoleList(
            roles=[
                Role(
                    id=UUID("12345678-1234-1234-1234-123456789012"),
                    name="Admin Role",
                    description="Administrator role",
                    is_system_role=True,
                    created_at="2025-01-01T00:00:00Z",
                    updated_at="2025-01-01T00:00:00Z",
                ),
                Role(
                    id=UUID("12345678-1234-1234-1234-123456789013"),
                    name="User Role",
                    description="Regular user role",
                    is_system_role=False,
                    created_at="2025-01-01T00:00:00Z",
                    updated_at="2025-01-01T00:00:00Z",
                ),
            ],
            total=2,
            page=1,
            per_page=10,
            total_pages=1,
        )

        mock_response = mock_responses(roles_data.model_dump(mode="json"))

        with patch.object(client, "get", return_value=mock_response):
            result = client.list_roles()

            assert len(result.roles) == 2
            assert result.roles[0].is_system_role is True
            assert result.roles[1].is_system_role is False

    def test_add_permission_to_role(self, client, sample_uuid, mock_responses):
        """Test adding permission to role."""
        expected_mapping = RolePermissionMapping(
            mapping_id=UUID("12345678-1234-1234-1234-123456789014"),
            role_id=sample_uuid,
            permission_id=sample_uuid,
            assigned_at="2025-01-01T00:00:00Z",
            assigned_by=sample_uuid,
        )

        mock_response = mock_responses(
            expected_mapping.model_dump(mode="json"),
            201,
        )

        with patch.object(client, "post", return_value=mock_response):
            result = client.add_permission_to_role(
                role_id=sample_uuid, permission_id=sample_uuid
            )

            assert result.role_id == sample_uuid
            assert result.permission_id == sample_uuid


class TestPermissionsAPI:
    """Test class for Permissions API methods."""

    def test_create_permission(
        self,
        client,
        test_permission_data,
        mock_responses,
    ):
        """Test creating a new permission."""
        expected_permission = Permission(
            id=UUID("12345678-1234-1234-1234-123456789012"),
            name=test_permission_data["name"],
            description=test_permission_data["description"],
            content=test_permission_data["content"],
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:00:00Z",
        )

        mock_response = mock_responses(
            expected_permission.model_dump(mode="json"),
            201,
        )

        with patch.object(client, "post", return_value=mock_response):
            result = client.create_permission(**test_permission_data)

            assert result.name == test_permission_data["name"]
            assert result.content == test_permission_data["content"]

    def test_list_permissions(self, client, mock_responses):
        """Test listing permissions."""
        permissions_data = PermissionList(
            permissions=[
                Permission(
                    id=UUID("12345678-1234-1234-1234-123456789012"),
                    name="Read Permission",
                    description="Permission to read resources",
                    content={"action": "read", "resource": "*"},
                    created_at="2025-01-01T00:00:00Z",
                    updated_at="2025-01-01T00:00:00Z",
                )
            ],
            total=1,
            page=1,
            per_page=10,
            total_pages=1,
        )

        mock_response = mock_responses(
            permissions_data.model_dump(mode="json"),
        )

        with patch.object(client, "get", return_value=mock_response):
            result = client.list_permissions()

            assert len(result.permissions) == 1
            assert result.permissions[0].name == "Read Permission"


class TestToolsAPI:
    """Test class for Tools API methods."""

    def test_create_tool(self, client, test_tool_data, mock_responses):
        """Test creating a new tool."""
        expected_tool = Tool(
            id=UUID("12345678-1234-1234-1234-123456789012"),
            name=test_tool_data["name"],
            description=test_tool_data["description"],
            config='{"type": "standalone", "url": "http://localhost:9999/mock", "params": {"type": "object", "properties": {"query": {"type": "string", "description": "The mock params"}}, "required": ["query"]}}',  # noqa: E501
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:00:00Z",
            created_by=UUID("12345678-1234-1234-1234-123456789012"),
        )

        mock_response = mock_responses(
            expected_tool.model_dump(mode="json"),
            201,
        )

        with patch.object(client, "post", return_value=mock_response):
            result = client.create_tool(**test_tool_data)

            assert result.name == test_tool_data["name"]
            assert result.description == test_tool_data["description"]

    def test_list_tools(self, client, mock_responses):
        """Test listing tools."""
        tools_data = ToolList(
            tools=[
                Tool(
                    id=UUID("12345678-1234-1234-1234-123456789012"),
                    name="Calculator Tool",
                    description="A calculator tool",
                    config='{"type": "standalone", "url": "http://localhost:9999/mock", "params": {"type": "object", "properties": {"query": {"type": "string", "description": "The mock params"}}, "required": ["query"]}}',  # noqa: E501
                    created_at="2025-01-01T00:00:00Z",
                    updated_at="2025-01-01T00:00:00Z",
                    created_by=UUID("12345678-1234-1234-1234-123456789012"),
                )
            ],
            total=1,
            page=1,
            per_page=10,
            total_pages=1,
        )

        mock_response = mock_responses(tools_data.model_dump(mode="json"))

        with patch.object(client, "get", return_value=mock_response):
            result = client.list_tools()

            assert len(result.tools) == 1
            assert result.tools[0].name == "Calculator Tool"

    def test_update_tool(self, client, sample_uuid, mock_responses):
        """Test updating a tool."""
        updated_tool = Tool(
            id=sample_uuid,
            name="Updated Tool",
            description="Updated description",
            config='{"type": "standalone", "url": "http://localhost:9999/mock", "params": {"type": "object", "properties": {"query": {"type": "string", "description": "The mock params"}}, "required": ["query"]}}',  # noqa: E501
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T01:00:00Z",
            created_by=UUID("12345678-1234-1234-1234-123456789012"),
        )

        mock_response = mock_responses(updated_tool.model_dump(mode="json"))

        with patch.object(client, "put", return_value=mock_response):
            result = client.update_tool(
                tool_id=sample_uuid,
                description="Updated description",
                config={"type": "updated", "version": "2.0"},
            )

            assert result.description == "Updated description"
            assert result.name == "Updated Tool"


class TestThreadsAndMessagesAPI:
    """Test class for Threads and Messages API methods."""

    def test_create_thread(self, client, sample_uuid, mock_responses):
        """Test creating a new thread."""
        expected_thread = Thread(
            id=UUID("12345678-1234-1234-1234-123456789012"),
            title="Test Thread",
            user_id=sample_uuid,
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:00:00Z",
        )

        mock_response = mock_responses(
            expected_thread.model_dump(mode="json"),
            201,
        )

        with patch.object(client, "post", return_value=mock_response):
            result = client.create_thread(
                title="Test Thread",
                user_id=sample_uuid,
            )

            assert result.title == "Test Thread"
            assert result.user_id == sample_uuid

    def test_list_threads(self, client, mock_responses):
        """Test listing threads."""
        threads_data = ThreadList(
            threads=[
                Thread(
                    id=UUID("12345678-1234-1234-1234-123456789012"),
                    title="Thread 1",
                    user_id=UUID("12345678-1234-1234-1234-123456789013"),
                    created_at="2025-01-01T00:00:00Z",
                    updated_at="2025-01-01T00:00:00Z",
                )
            ],
            total=1,
            page=1,
            per_page=10,
            total_pages=1,
        )

        mock_response = mock_responses(threads_data.model_dump(mode="json"))

        with patch.object(client, "get", return_value=mock_response):
            result = client.list_threads()

            assert len(result.threads) == 1
            assert result.threads[0].title == "Thread 1"

    def test_create_message(self, client, sample_uuid, mock_responses):
        """Test creating a message in a thread."""
        expected_message = Message(
            id=UUID("12345678-1234-1234-1234-123456789012"),
            message={"role": "user", "content": "Hello"},
            recipient_id=sample_uuid,
            sender_id=sample_uuid,
            sender_type="user",
            thread_id=sample_uuid,
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:00:00Z",
        )

        mock_response = mock_responses(
            expected_message.model_dump(mode="json"),
            201,
        )

        with patch.object(client, "post", return_value=mock_response):
            result = client.create_message(
                thread_id=sample_uuid,
                message={"role": "user", "content": "Hello"},
                recipient_id=sample_uuid,
                sender_id=sample_uuid,
            )

            assert result.message == {"role": "user", "content": "Hello"}
            assert result.sender_id == sample_uuid

    def test_list_messages(self, client, sample_uuid, mock_responses):
        """Test listing messages in a thread."""
        messages_data = MessageList(
            messages=[
                Message(
                    id=UUID("12345678-1234-1234-1234-123456789012"),
                    message={"role": "user", "content": "Hello"},
                    recipient_id=sample_uuid,
                    sender_id=sample_uuid,
                    sender_type="user",
                    thread_id=sample_uuid,
                    created_at="2025-01-01T00:00:00Z",
                    updated_at="2025-01-01T00:00:00Z",
                ),
                Message(
                    id=UUID("12345678-1234-1234-1234-123456789013"),
                    message={"role": "assistant", "content": "Hi there!"},
                    recipient_id=sample_uuid,
                    sender_id=sample_uuid,
                    sender_type="assistant",
                    thread_id=sample_uuid,
                    created_at="2025-01-01T00:00:01Z",
                    updated_at="2025-01-01T00:00:01Z",
                ),
            ],
            total=2,
            page=1,
            per_page=10,
            total_pages=1,
        )

        mock_response = mock_responses(messages_data.model_dump(mode="json"))

        with patch.object(client, "get", return_value=mock_response):
            result = client.list_messages(sample_uuid)

            assert len(result.messages) == 2
            assert result.messages[0].message["role"] == "user"
            assert result.messages[1].message["role"] == "assistant"


class TestUserRolesMappingAPI:
    """Test class for User-Roles mapping API methods."""

    def test_add_role_to_user(self, client, sample_uuid, mock_responses):
        """Test adding a role to a user."""
        expected_mapping = UserRoleMapping(
            mapping_id=UUID("12345678-1234-1234-1234-123456789016"),
            user_id=sample_uuid,
            role_id=sample_uuid,
            assigned_at="2025-01-01T00:00:00Z",
            assigned_by=sample_uuid,
        )

        mock_response = mock_responses(
            expected_mapping.model_dump(mode="json"),
            201,
        )

        with patch.object(client, "post", return_value=mock_response):
            result = client.add_role_to_user(
                user_id=sample_uuid,
                role_id=sample_uuid,
            )

            assert result.user_id == sample_uuid
            assert result.role_id == sample_uuid

    def test_list_roles_for_user(self, client, sample_uuid, mock_responses):
        """Test listing roles for a user."""
        roles_mappings = [
            UserRoleMapping(
                mapping_id=UUID("12345678-1234-1234-1234-123456789016"),
                user_id=sample_uuid,
                role_id=UUID("12345678-1234-1234-1234-123456789017"),
                assigned_at="2025-01-01T00:00:00Z",
                assigned_by=sample_uuid,
            )
        ]

        mock_response = mock_responses(
            [mapping.model_dump(mode="json") for mapping in roles_mappings]
        )

        with patch.object(client, "get", return_value=mock_response):
            result = client.list_roles_for_user(sample_uuid)

            assert len(result) == 1
            assert result[0].user_id == sample_uuid

    def test_remove_role_from_user(self, client, sample_uuid, mock_responses):
        """Test removing a role from a user."""
        mock_response = mock_responses(None, 204)

        with patch.object(client, "delete", return_value=mock_response):
            # Should not raise an exception
            client.remove_role_from_user(
                user_id=sample_uuid,
                role_id=sample_uuid,
            )
