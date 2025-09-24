"""Example invoke agent + tools through pinazu client with SSE"""

import pinazu
import yaml

client = pinazu.Client()

# Use the default user_id
user_id = "550e8400-c95b-4444-6666-446655440000"
agent, thread, task, first_tool, second_tool, third_tool = (
    None,
    None,
    None,
    None,
    None,
    None,
)

try:
    # Create a new first tool
    first_tool = client.create_tool(
        name="first_test_tool",
        description="This is a test tool",
        config={
            "type": "standalone",
            "url": "http://localhost:8080/v1/mock/tool",
            "params": {
                "type": "object",
                "properties": {
                    "input": {
                        "type": "string",
                        "description": "Mock input for the tool",
                    }
                },
                "required": ["input"],
            },
        },
    )

    # Create a new second tool
    second_tool = client.create_tool(
        name="second_test_tool",
        description="This is a test tool",
        config={
            "type": "standalone",
            "url": "http://localhost:8080/v1/mock/tool_with_delay",
            "params": {
                "type": "object",
                "properties": {
                    "input": {
                        "type": "string",
                        "description": "Mock input for the tool",
                    }
                },
                "required": ["input"],
            },
        },
    )

    # Create a new third tool
    third_tool = client.create_tool(
        name="third_test_tool",
        description="This is a test tool",
        config={
            "type": "standalone",
            "url": "http://localhost:6060/v1/mock/tool",
            "params": {
                "type": "object",
                "properties": {
                    "input": {
                        "type": "string",
                        "description": "Mock input for the tool",
                    }
                },
                "required": ["input"],
            },
        },
    )

    # Create a new agent
    agent = client.create_agent(
        name="test_agent",
        description="This is a test description",
        specs=yaml.safe_dump(
            {
                "model": {
                    "provider": "bedrock/anthropic",
                    "model_id": "anthropic.claude-3-haiku-20240307-v1:0",
                    # "model_id": "apac.anthropic.claude-sonnet-4-20250514-v1:0", # noqa: E501
                    "max_tokens": 8192,
                    "thinking": {"enabled": False},
                    "stream": True,  # Enable streaming
                },
                "system": "You are a helpful assistant.",
                "tool_refs": [
                    first_tool.id.hex,
                    second_tool.id.hex,
                    third_tool.id.hex,
                ],
            }
        ),
    )

    # Create a new thread
    thread = client.create_thread(title="example_thread", user_id=user_id)

    # Create a new message
    message = client.create_message(
        thread_id=thread.id,
        message={
            "role": "user",
            "content": [
                {
                    "type": "text",
                    "text": "Use all the curren tools for demonstrate please. No use parallel. Each tool must be executed once at a time. All the tools must be use twice event if the frist call is error or not. ALL two must use second time.",  # noqa: E501
                }
            ],
        },
        recipient_id=agent.id,
        sender_id=user_id,
    )

    # Create a new task
    task = client.create_task(thread_id=thread.id, max_request_loop=20)

    # Execute the task with streaming
    stream = client.execute_task(
        task_id=task.id,
        agent_id=agent.id,
        stream=True,
    )

    # Handle the SSE response
    for event in stream:
        event_type = event.get("type", "")
        if event_type == "content_block_start":
            if event.get("content_block", {}).get("type", "") == "tool_use":
                print("\n\nTool use started\n\n")
            elif event.get("content_block", {}).get("type", "") == "text":
                print("\n\nText started\n\n")
        if event_type == "content_block_delta":
            delta = event.get("delta", {})
            text = delta.get("text", "")
            partial_json = delta.get("partial_json", "")
            if text:
                print(text, end="", flush=True)
            if partial_json:
                print(partial_json, end="", flush=True)


finally:
    # Clean up the agent
    if agent:
        client.delete_agent(agent.id)

    # Clean up the tool
    if first_tool:
        client.delete_tool(first_tool.id)
    if second_tool:
        client.delete_tool(second_tool.id)
    if third_tool:
        client.delete_tool(third_tool.id)

    # Delete thread will also delete messgae CASCADE
    if thread:
        client.delete_thread(thread.id)
