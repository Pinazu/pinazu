"""Example invoke multi agent + tools through pinazu client with SSE"""

import pinazu
import yaml

client = pinazu.Client()

# Use the default user_id
user_id = "550e8400-c95b-4444-6666-446655440000"
agent_one, agent_two, tool, thread, task = None, None, None, None, None

try:
    # Create a new first tool
    tool = client.create_tool(
        name="test_tool",
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

    # Create a sub agent
    sub_agent = client.create_agent(
        name="sub_agent",
        description="This is a test sub agent",
        specs=yaml.safe_dump(
            {
                "model": {
                    "provider": "bedrock/anthropic",
                    "model_id": "anthropic.claude-3-haiku-20240307-v1:0",
                    "max_tokens": 2048,
                    "thinking": {"enabled": False},
                    "stream": True,  # Enable streaming,
                },
                "system": "You are a helpful assistant.",
                "tool_refs": [tool.id.hex],
            },
        ),
    )

    # Create a new agent
    agent = client.create_agent(
        name="test_agent",
        description="This is a main agent",
        specs=yaml.safe_dump(
            {
                "model": {
                    "provider": "bedrock/anthropic",
                    "model_id": "apac.anthropic.claude-sonnet-4-20250514-v1:0",
                    "max_tokens": 8192,
                    "thinking": {"enabled": False},
                    "stream": True,  # Enable streaming
                },
                "system": "You are a helpful assistant.",
                "sub_agents": {
                    "configs": {"shared_memory": True},
                    "allows": [sub_agent.id.hex],
                },
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
                    "text": "Ask the sub agent about the test tool result",
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
        if event_type == "sub_task_start":
            print("\n\nSub task started\n\n")
        if event_type == "sub_task_stop":
            print("\n\nSub task stopped\n\n")
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
    # Clean up the agents
    if sub_agent:
        client.delete_agent(sub_agent.id)
    if agent:
        client.delete_agent(agent.id)

    # Clean up the tool
    if tool:
        client.delete_tool(tool.id)

    # Delete thread will also delete messgae CASCADE
    if thread:
        client.delete_thread(thread.id)
