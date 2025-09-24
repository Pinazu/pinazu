"""End-to-end integration tests for the pinazu-py client.

These tests require a running Pinazu server instance and
will make actual HTTP requests.
They should be run separately from unit tests and
only in integration test environments.
"""

import pytest
import uuid
from uuid import UUID
from pinazu import PinazuAPIError


@pytest.mark.integration
class TestE2EAgentsWorkflow:
    """End-to-end test class for Agents workflow."""

    def test_complete_agent_lifecycle(self, client):
        """Test complete agent lifecycle: create, read, update, delete."""
        agent_name = f"E2E Test Agent {uuid.uuid4().hex[:8]}"

        # Create agent
        agent = client.create_agent(
            name=agent_name,
            description="E2E test agent for integration testing",
            specs="""
model:
  provider: "anthropic"
  model_id: "claude-3-sonnet"
  max_tokens: 4096
  temperature: 0.7

system: |
  You are a test AI assistant for integration testing.
  Respond clearly and concisely.

tools:
  - name: "test_calculator"
    description: "Perform test calculations"

parameters:
  thinking_enabled: false
  debug_mode: true
            """,
        )

        assert agent.name == agent_name
        assert agent.id is not None
        created_agent_id = agent.id

        try:
            # Read agent
            retrieved_agent = client.get_agent(created_agent_id)
            assert retrieved_agent.id == created_agent_id
            assert retrieved_agent.name == agent_name
            assert "claude-3-sonnet" in retrieved_agent.specs

            # List agents (should include our created agent)
            agents_list = client.list_agents()
            agent_ids = [a.id for a in agents_list.agents]
            assert created_agent_id in agent_ids

            # Update agent
            updated_name = f"Updated {agent_name}"
            updated_agent = client.update_agent(
                agent_id=created_agent_id,
                name=updated_name,
                description="Updated description for E2E test",
            )
            assert updated_agent.name == updated_name
            assert (
                updated_agent.description == "Updated description for E2E test"
            )  # noqa: E501

            # Verify update persisted
            retrieved_updated = client.get_agent(created_agent_id)
            assert retrieved_updated.name == updated_name

        finally:
            # Cleanup: Delete agent
            client.delete_agent(created_agent_id)

            # Verify deletion
            with pytest.raises(PinazuAPIError) as exc_info:
                client.get_agent(created_agent_id)
            assert exc_info.value.status_code == 404

    def test_agent_permissions_workflow(self, client):
        """Test agent permissions assignment workflow."""
        # Create test agent
        agent_name = f"E2E Permissions Agent {uuid.uuid4().hex[:8]}"
        agent = client.create_agent(
            name=agent_name, description="Agent for testing permissions"
        )

        # Create test permission
        permission_name = f"E2E Test Permission {uuid.uuid4().hex[:8]}"
        permission = client.create_permission(
            name=permission_name,
            description="Permission for E2E testing",
            content={
                "action": "test_action",
                "resource": "test_resource",
                "conditions": [],
            },
        )

        try:
            # Initially no permissions
            initial_permissions = client.list_agent_permissions(agent.id)
            assert len(initial_permissions.permissionMappings) == 0

            # Add permission to agent
            mapping = client.add_permission_to_agent(
                agent_id=agent.id, permission_id=permission.id
            )
            assert mapping.agent_id == agent.id
            assert mapping.permission_id == permission.id

            # Verify permission was added
            updated_permissions = client.list_agent_permissions(agent.id)
            assert len(updated_permissions.permissionMappings) == 1
            assert (
                updated_permissions.permissionMappings[0].permission_id
                == permission.id  # noqa: E501
            )

            # Try to add duplicate permission (should fail)
            with pytest.raises(PinazuAPIError) as exc_info:
                client.add_permission_to_agent(
                    agent_id=agent.id, permission_id=permission.id
                )
            assert exc_info.value.status_code == 409

            # Remove permission from agent
            client.remove_permission_from_agent(
                agent_id=agent.id, permission_id=permission.id
            )

            # Verify permission was removed
            final_permissions = client.list_agent_permissions(agent.id)
            assert len(final_permissions.permissionMappings) == 0

        finally:
            # Cleanup
            client.delete_agent(agent.id)
            client.delete_permission(permission.id)


@pytest.mark.integration
class TestE2EFlowsWorkflow:
    """End-to-end test class for Flows workflow."""

    def test_complete_flow_lifecycle(self, client):
        """Test complete flow lifecycle: create, read,
        update, execute, delete."""
        flow_name = f"E2E Test Flow {uuid.uuid4().hex[:8]}"

        # Create flow
        flow = client.create_flow(
            name=flow_name,
            description="E2E test flow for integration testing",
            engine="python",
            entrypoint="main.py",
            code_location="s3://test-bucket/e2e_flow.py",
            parameters_schema={
                "type": "object",
                "properties": {
                    "input_message": {
                        "type": "string",
                        "description": "Input message for processing",
                    }
                },
                "required": ["input_message"],
            },
            tags=["e2e", "test", "integration"],
        )

        assert flow.name == flow_name
        assert flow.id is not None
        created_flow_id = flow.id

        try:
            # Read flow
            retrieved_flow = client.get_flow(created_flow_id)
            assert retrieved_flow.id == created_flow_id
            assert retrieved_flow.name == flow_name
            assert retrieved_flow.engine == "python"

            # List flows (should include our created flow)
            flows_list = client.list_flows(
                page=1, per_page=100
            )  # Use large per_page to catch our flow
            flow_ids = [f.id for f in flows_list.flows]
            assert created_flow_id in flow_ids

            # Update flow
            updated_name = f"Updated {flow_name}"
            updated_flow = client.update_flow(
                flow_id=created_flow_id,
                name=updated_name,
                description="Updated description for E2E test",
                tags=["updated", "e2e", "test"],
            )
            assert updated_flow.name == updated_name
            assert "updated" in updated_flow.tags

            # Execute flow
            try:
                flow_run = client.execute_flow(
                    flow_id=created_flow_id,
                    parameters={"input_message": "E2E test execution"},
                )
                assert flow_run.flow_id == created_flow_id
                assert flow_run.parameters == {
                    "input_message": "E2E test execution",
                }

                # Try to get flow run status
                flow_run_status = client.get_flow_run(flow_run.flow_run_id)
                assert flow_run_status.flow_run_id == flow_run.flow_run_id

            except PinazuAPIError as e:
                # Execution might fail if no workers available
                if e.status_code not in [
                    404,
                    503,
                ]:  # 404: no workers, 503: service unavailable
                    raise

        finally:
            # Cleanup: Delete flow
            client.delete_flow(created_flow_id)

            # Verify deletion
            with pytest.raises(PinazuAPIError) as exc_info:
                client.get_flow(created_flow_id)
            assert exc_info.value.status_code == 404


@pytest.mark.integration
class TestE2EUsersWorkflow:
    """End-to-end test class for Users workflow."""

    def test_complete_user_lifecycle(self, client):
        """Test complete user lifecycle: create, read, update, delete."""
        user_email = f"e2e+{uuid.uuid4().hex[:8]}@test.com"

        # Create user
        user = client.create_user(
            name="E2E Test User",
            email=user_email,
            password_hash="test_hash_123",
            provider_name="local",
            additional_info={"test_type": "e2e", "automated": True},
        )

        assert user.email == user_email
        assert user.id is not None
        created_user_id = user.id

        try:
            # Read user
            retrieved_user = client.get_user(created_user_id)
            assert retrieved_user.id == created_user_id
            assert retrieved_user.email == user_email
            assert retrieved_user.additional_info["test_type"] == "e2e"

            # List users (should include our created user)
            users_list = client.list_users(page=1, per_page=100)
            user_ids = [u.id for u in users_list.users]
            assert created_user_id in user_ids

            # Update user
            updated_user = client.update_user(
                user_id=created_user_id,
                username="updated_e2e_user",
                additional_info={"test_type": "e2e", "updated": True},
            )
            assert updated_user.name == "updated_e2e_user"
            assert updated_user.additional_info["updated"] is True

        finally:
            # Cleanup: Delete user
            client.delete_user(created_user_id)

            # Verify deletion
            with pytest.raises(PinazuAPIError) as exc_info:
                client.get_user(created_user_id)
            assert exc_info.value.status_code == 404


@pytest.mark.integration
class TestE2EUserRolesWorkflow:
    """End-to-end test class for User Roles workflow."""

    def test_user_roles_assignment(self, client):
        """Test user roles assignment workflow."""
        # Create test user
        user_email = f"e2e+roles+{uuid.uuid4().hex[:8]}@test.com"
        user = client.create_user(
            name="E2E Roles User",
            email=user_email,
            password_hash="test_hash_456",
        )

        # Create test role
        role_name = f"E2E Test Role {uuid.uuid4().hex[:8]}"
        role = client.create_role(
            name=role_name,
            description="Role for E2E testing",
            is_system_role=False,
        )

        try:
            # Get initial roles (may include default system role)
            initial_roles = client.list_roles_for_user(user.id)
            initial_count = len(initial_roles)

            # Add role to user
            mapping = client.add_role_to_user(user_id=user.id, role_id=role.id)
            assert mapping.user_id == user.id
            assert mapping.role_id == role.id

            # Verify role was added (should have one more role than initial)
            updated_roles = client.list_roles_for_user(user.id)
            assert len(updated_roles) == initial_count + 1
            
            # Verify our test role is in the list
            role_ids = [r.role_id for r in updated_roles]
            assert role.id in role_ids

            # Remove role from user
            client.remove_role_from_user(user_id=user.id, role_id=role.id)

            # Verify role was removed (should be back to initial count)
            final_roles = client.list_roles_for_user(user.id)
            assert len(final_roles) == initial_count
            
            # Verify our test role is no longer in the list
            final_role_ids = [r.role_id for r in final_roles]
            assert role.id not in final_role_ids

        finally:
            # Cleanup
            client.delete_user(user.id)
            client.delete_role(role.id)


@pytest.mark.integration
@pytest.mark.asyncio
class TestE2EAsyncClient:
    """End-to-end test class for Async Client."""

    async def test_async_agent_lifecycle(self, async_client):
        """Test async client with agent operations."""
        agent_name = f"E2E Async Agent {uuid.uuid4().hex[:8]}"

        # Create agent
        agent = await async_client.create_agent(
            name=agent_name, description="Async E2E test agent"
        )

        assert agent.name == agent_name
        created_agent_id = agent.id

        try:
            # Read agent
            retrieved_agent = await async_client.get_agent(created_agent_id)
            assert retrieved_agent.id == created_agent_id

            # List agents
            agents_list = await async_client.list_agents()
            agent_ids = [a.id for a in agents_list.agents]
            assert created_agent_id in agent_ids

            # Update agent
            updated_agent = await async_client.update_agent(
                agent_id=created_agent_id, name=f"Updated {agent_name}"
            )
            assert f"Updated {agent_name}" in updated_agent.name

        finally:
            # Cleanup
            await async_client.delete_agent(created_agent_id)

            # Verify deletion
            with pytest.raises(PinazuAPIError) as exc_info:
                await async_client.get_agent(created_agent_id)
            assert exc_info.value.status_code == 404


@pytest.mark.integration
class TestE2EErrorScenarios:
    """End-to-end test class for error scenarios."""

    def test_nonexistent_resource_errors(self, client):
        """Test various 404 scenarios with non-existent resources."""
        fake_uuid = UUID("12345678-1234-1234-1234-123456789999")

        # Test all GET endpoints with non-existent UUIDs
        with pytest.raises(PinazuAPIError) as exc_info:
            client.get_agent(fake_uuid)
        assert exc_info.value.status_code == 404

        with pytest.raises(PinazuAPIError) as exc_info:
            client.get_flow(fake_uuid)
        assert exc_info.value.status_code == 404

        with pytest.raises(PinazuAPIError) as exc_info:
            client.get_user(fake_uuid)
        assert exc_info.value.status_code == 404

        with pytest.raises(PinazuAPIError) as exc_info:
            client.get_role(fake_uuid)
        assert exc_info.value.status_code == 404

    def test_validation_errors(self, client):
        """Test various validation error scenarios."""
        # Test agent creation with invalid data
        with pytest.raises(PinazuAPIError) as exc_info:
            client.create_agent(name="")  # Empty name should fail
        assert exc_info.value.status_code == 400

        # Test user creation with invalid email
        with pytest.raises(PinazuAPIError) as exc_info:
            client.create_user(
                name="Test",
                email="invalid-email",  # Invalid email format
                password_hash="hash",
            )
        assert exc_info.value.status_code == 400

    def test_nil_uuid_handling(self, client, nil_uuid):
        """Test handling of nil UUID (all zeros)."""
        with pytest.raises(PinazuAPIError) as exc_info:
            client.get_agent(nil_uuid)
        assert exc_info.value.status_code == 404
