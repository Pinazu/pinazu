# pinazu-py

Python client library for interacting with the Pinazu Server - an AI agent orchestration platform.

## Installation

```bash
pip install pinazu-py
```

## Requirements

- Python 3.10+
- Pinazu Server running (default: `http://localhost:8080`)

## Quick Start

```python
import pinazu
import yaml

# Initialize client
client = pinazu.Client()

# Create an agent
agent = client.create_agent(
    name="my_agent",
    description="A helpful assistant",
    specs=yaml.safe_dump({
        "model": {
            "provider": "bedrock/anthropic",
            "model_id": "anthropic.claude-3-haiku-20240307-v1:0",
            "max_tokens": 8192
        },
        "system": "You are a helpful assistant."
    })
)

# Create a thread and message
thread = client.create_thread(title="conversation")
message = client.create_message(
    thread_id=thread.id,
    message={
        "role": "user",
        "content": [{"type": "text", "text": "Hello!"}]
    },
    recipient_id=agent.id
)

# Execute task with streaming
task = client.create_task(thread_id=thread.id)
stream = client.execute_task(task_id=task.id, agent_id=agent.id, stream=True)

for event in stream:
    if event.get("type") == "content_block_delta":
        print(event.get("delta", {}).get("text", ""), end="", flush=True)

# Cleanup
client.delete_agent(agent.id)
client.delete_thread(thread.id)
```

## Key Features

- **Agent Management**: Create and manage AI agents with various providers
- **Tool Integration**: Connect external tools to agents
- **Thread Management**: Organize conversations in threads
- **Task Execution**: Execute agent tasks with streaming support
- **Flow Orchestration**: Build complex workflows

## Examples

See the [examples/](examples/) directory for more use cases:
- `examples/agents/` - Agent creation and execution
- `examples/api/` - API interactions
- `examples/flows/` - Workflow examples

## License

MIT
