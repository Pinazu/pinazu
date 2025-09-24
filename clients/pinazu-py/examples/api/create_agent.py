"""Example interact with agent API through pinazu client"""

import pinazu
import yaml

client = pinazu.Client()

# Create spec and convert to yaml file
specs = yaml.safe_dump(
    {
        "model": {
            "provider": "bedrock/anthropic",
            "model_id": "apac.anthropic.claude-sonnet-4-20250514-v1:0",
            "max_tokens": 8192,
            "thinking": {"enabled": False},
        },
        "system": "You are a dynamo query generate agent.",
    }
)

# Create a new agent
agent = client.create_agent(
    name="test_agent",
    description="This is a test description",
    specs=specs,
)

# Print out the new agent info
print(agent)

# Clean up the agent
client.delete_agent(agent.id)
