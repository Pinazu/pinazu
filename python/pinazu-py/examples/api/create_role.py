"""Example interact with roles API through pinazu client"""

import pinazu

client = pinazu.Client()

# Use the default user_id
user_id = "550e8400-c95b-4444-6666-446655440000"

# Create a new thread
thread = client.create_permission(
    name="example_permission",
    description="example_description",
    user_id=user_id,
)

# Create a new message
message = client.create_message(
    thread_id=thread.id,
    message={"role": "user", "content": [{"type": "text", "text": "Hello"}]},
    recipient_id=user_id,
    sender_id=user_id,
)

# Print out the thread and the new message
print("New thread created:", thread)
print("New message created:", message)

# Print out the list of messages in the thread
print("Messages in thread:", client.list_messages(thread.id).messages)

# Clean up
client.delete_message(thread.id, message.id)
client.delete_thread(thread.id)
