"""Example interact with tools API through pinazu client"""

import pinazu

client = pinazu.Client()


# Create a new tool with unique name
tool = client.create_tool(
    name="test_tool",
    description="This is a test description",
    config={
        "type": "standalone",
        "url": "http://localhost:9999/mock",
        "params": {
            "type": "object",
            "properties": {
                "query": {
                    "type": "string",
                    "description": "The mock params",
                }
            },
            "required": ["query"],
        },
    },
)

# Print out the new tool info
print(tool)

# Clean up the tool
client.delete_tool(tool.id)
