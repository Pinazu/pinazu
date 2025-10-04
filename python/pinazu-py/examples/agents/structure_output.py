"""Example structure output for agent from pinazu client with SSE"""

import pinazu
import yaml
import json

client = pinazu.Client()

# Use the default user_id
user_id = "550e8400-c95b-4444-6666-446655440000"
agent, thread, task = None, None, None

try:
    # Create a new agent
    agent = client.create_agent(
        name="test_agent",
        description="This is a test description",
        specs=yaml.safe_dump(
            {
                "model": {
                    "provider": "bedrock/anthropic",
                    "model_id": "apac.anthropic.claude-sonnet-4-20250514-v1:0",
                    "max_tokens": 8192,
                    "thinking": {
                        "enabled": False,  # Change to True to enable though
                        "budget_token": 1024,
                    },
                    "response_format": {
                        "type": "array",
                        "items": {
                            "type": "object",
                            "properties": {
                                "name": {
                                    "type": "string",
                                    "description": "The name of the person",
                                },
                                "age": {
                                    "type": "integer",
                                    "description": "The age of the person",
                                },
                            },
                        },
                    },
                    "stream": True,  # Enable streaming
                },
                "system": "You are a extractor, based on the paragraph. Please extract all their name and the age.",  # noqa; E501
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
                    "text": "Yesterday I met Samantha, who just turned 29, and her cousin Leo, only 17 years old but already taller than her. Their grandfather, Mr. Albert Johnson, born in 1942, proudly told me that his youngest granddaughter, Maya, will be 12 next month. By the way, Samantha’s cat is also named Leo, but of course he’s not 17—he’s just a 3-year-old tabby. Oh, and don’t confuse this Leo with Samantha’s uncle Leonard, who is 47 but prefers people not to mention his age.",  # noqa; E501
                }
            ],
        },
        recipient_id=agent.id,
        sender_id=user_id,
    )

    # Create a new task
    task = client.create_task(thread_id=thread.id, max_request_loop=1)

    # Execute the task with streaming
    stream = client.execute_task(
        task_id=task.id,
        agent_id=agent.id,
        stream=True,
    )

    # Handle the SSE response
    json_response = ""
    for event in stream:
        event_type = event.get("type", "")
        if event_type == "content_block_start":
            print("\n\nText started\n\n")
        if event_type == "content_block_delta":
            delta = event.get("delta", {})
            text = delta.get("text", "")
            if text:
                print(text, end="", flush=True)
                json_response += text

    # Get the response and json it.
    result = json.loads("{" + json_response)
    print("\n\n")
    print(result.get("answer"))


finally:
    # Clean up the agent
    if agent:
        client.delete_agent(agent.id)

    # Delete thread will also delete messgae CASCADE
    if thread:
        client.delete_thread(thread.id)
