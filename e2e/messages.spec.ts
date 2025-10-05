import { test, expect } from '@playwright/test';

test.describe.serial('Messages API', () => {
  let createdThreadId: string;
  let createdMessageId: string;
  let testUserId: string = '550e8400-c95b-4444-6666-446655440000'; // Admin user UUID from database
  let testSenderId: string = '550e8400-c95b-4444-6666-446655440000'; // Admin user UUID from database
  let testRecipientId: string = '550e8400-c95b-5555-6666-446655440000'; // Supervisor UUID from database

  test.beforeAll(async ({ request }) => {
    // Create a test thread for messages testing
    const threadData = {
      title: 'Test Thread for Messages',
      user_id: testUserId
    };

    const threadResponse = await request.post('/v1/threads', {
      data: threadData
    });

    expect(threadResponse.status()).toBe(201);
    const threadBody = await threadResponse.json();
    createdThreadId = threadBody.id;
    console.log(`Created test thread with ID: ${createdThreadId}`);

    // Clean up any existing test messages in this thread
    const messagesResponse = await request.get(`/v1/threads/${createdThreadId}/messages`);
    if (messagesResponse.ok()) {
      const messagesBody = await messagesResponse.json();
      const messages = messagesBody.messages || [];
      if (Array.isArray(messages)) {
        for (const message of messages) {
          if (message.message && message.message.content && message.message.content.includes('Test Message')) {
            await request.delete(`/v1/threads/${createdThreadId}/messages/${message.id}`);
          }
        }
      }
    }
  });

  test.afterAll(async ({ request }) => {
    // Clean up test thread (this will also clean up all messages in it)
    if (createdThreadId) {
      await request.delete(`/v1/threads/${createdThreadId}`);
    }
  });

  test('should create a new message in a thread', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();

    const messageData = {
      message: {
        role: 'user',
        content: 'Test Message for API Testing',
        timestamp: new Date().toISOString()
      },
      sender_id: testSenderId,
      recipient_id: testRecipientId
    };

    const response = await request.post(`/v1/threads/${createdThreadId}/messages`, {
      data: messageData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('id');
    expect(responseBody.thread_id).toBe(createdThreadId);
    expect(responseBody.message).toEqual(messageData.message);
    expect(responseBody.sender_id).toBe(messageData.sender_id);
    expect(responseBody.recipient_id).toBe(messageData.recipient_id);
    expect(responseBody).toHaveProperty('created_at');
    expect(responseBody).toHaveProperty('updated_at');
    expect(responseBody).toHaveProperty('sender_type');
    expect(responseBody).toHaveProperty('citations');

    createdMessageId = responseBody.id;
  });

  test('should get all messages in a thread', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();

    const response = await request.get(`/v1/threads/${createdThreadId}/messages`);

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('messages');
    expect(responseBody.messages).toBeDefined();
    expect(Array.isArray(responseBody.messages)).toBe(true);
    expect(responseBody.messages.length).toBeGreaterThanOrEqual(1); // At least our created message
  });

  test('should get message by ID', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();
    expect(createdMessageId).toBeTruthy();

    const response = await request.get(`/v1/threads/${createdThreadId}/messages/${createdMessageId}`);

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdMessageId);
    expect(responseBody.thread_id).toBe(createdThreadId);
    expect(responseBody.message.content).toBe('Test Message for API Testing');
    expect(responseBody.sender_id).toBe(testSenderId);
    expect(responseBody.recipient_id).toBe(testRecipientId);
  });

  test('should return 404 for non-existent message', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();

    const nonExistentId = '12345678-1234-1234-1234-123456789012';
    const response = await request.get(`/v1/threads/${createdThreadId}/messages/${nonExistentId}`);

    expect(response.status()).toBe(404);
    
    const responseBody = await response.json();
    expect(responseBody.resource).toBe("Message");
    expect(responseBody.id).toBe(nonExistentId);
  });

  test('should return 404 for message in non-existent thread', async ({ request }) => {
    const nonExistentThreadId = '12345678-1234-1234-1234-123456789013';
    const nonExistentMessageId = '12345678-1234-1234-1234-123456789014';
    
    const response = await request.get(`/v1/threads/${nonExistentThreadId}/messages/${nonExistentMessageId}`);

    expect(response.status()).toBe(404);
  });

  test('should return 400 for invalid message ID format', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();

    const invalidId = 'invalid-uuid';
    const response = await request.get(`/v1/threads/${createdThreadId}/messages/${invalidId}`);

    expect(response.status()).toBe(400);
  });

  test('should return 400 for invalid thread ID format in message request', async ({ request }) => {
    const invalidThreadId = 'invalid-uuid';
    const validMessageId = '12345678-1234-1234-1234-123456789015';
    
    const response = await request.get(`/v1/threads/${invalidThreadId}/messages/${validMessageId}`);

    expect(response.status()).toBe(400);
  });

  test('should return 404 for nil UUID (validation error)', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();

    const nilUuid = '00000000-0000-0000-0000-000000000000';
    const response = await request.get(`/v1/threads/${createdThreadId}/messages/${nilUuid}`);

    expect(response.status()).toBe(404);
    
    const responseBody = await response.json();
    expect(responseBody.resource).toBe("Message");
    expect(responseBody.id).toBe(nilUuid);
  });

  test('should update an existing message', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();
    expect(createdMessageId).toBeTruthy();

    const updateData = {
      message: {
        role: 'user',
        content: 'Updated Test Message Content',
        timestamp: new Date().toISOString(),
        edited: true
      }
    };

    const response = await request.put(`/v1/threads/${createdThreadId}/messages/${createdMessageId}`, {
      data: updateData
    });

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdMessageId);
    expect(responseBody.thread_id).toBe(createdThreadId);
    expect(responseBody.message).toEqual(updateData.message);
    expect(responseBody).toHaveProperty('updated_at');
  });

  test('should return 404 when updating non-existent message', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();

    const nonExistentId = '12345678-1234-1234-1234-123456789abc';
    const updateData = {
      message: {
        role: 'user',
        content: 'Non-existent Message Content'
      }
    };

    const response = await request.put(`/v1/threads/${createdThreadId}/messages/${nonExistentId}`, {
      data: updateData
    });

    expect(response.status()).toBe(404);
  });

  test('should handle validation errors for create message - missing required fields', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();

    const invalidMessageData = {
      // Missing required message field
      sender_id: testSenderId,
      recipient_id: testRecipientId
    };

    const response = await request.post(`/v1/threads/${createdThreadId}/messages`, {
      data: invalidMessageData
    });

    expect(response.status()).toBe(400);
  });

  test('should handle validation errors for invalid sender_id format', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();

    const invalidMessageData = {
      message: {
        role: 'user',
        content: 'Valid message content'
      },
      sender_id: 'invalid-uuid-format',
      recipient_id: testRecipientId
    };

    const response = await request.post(`/v1/threads/${createdThreadId}/messages`, {
      data: invalidMessageData
    });

    expect(response.status()).toBe(400);
  });

  test('should handle validation errors for invalid recipient_id format', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();

    const invalidMessageData = {
      message: {
        role: 'user',
        content: 'Valid message content'
      },
      sender_id: testSenderId,
      recipient_id: 'invalid-uuid-format'
    };

    const response = await request.post(`/v1/threads/${createdThreadId}/messages`, {
      data: invalidMessageData
    });

    expect(response.status()).toBe(400);
  });

  test('should create message with complex JSON content', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();

    const complexMessageData = {
      message: {
        role: 'assistant',
        content: 'Complex message with attachments',
        timestamp: new Date().toISOString(),
        metadata: {
          model: 'claude-3-sonnet',
          temperature: 0.7,
          max_tokens: 4096
        },
        attachments: [
          {
            type: 'file',
            name: 'example.txt',
            size: 1024
          },
          {
            type: 'image',
            name: 'chart.png',
            size: 2048,
            dimensions: { width: 800, height: 600 }
          }
        ],
        tools_used: ['calculator', 'weather_api']
      },
      sender_id: testSenderId,
      recipient_id: testRecipientId
    };

    const response = await request.post(`/v1/threads/${createdThreadId}/messages`, {
      data: complexMessageData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody.message).toEqual(complexMessageData.message);
    expect(responseBody.message.metadata.model).toBe('claude-3-sonnet');
    expect(responseBody.message.attachments).toHaveLength(2);
    expect(responseBody.message.tools_used).toContain('calculator');

    // Clean up the complex message
    await request.delete(`/v1/threads/${createdThreadId}/messages/${responseBody.id}`);
  });

  test('should create message with minimal required fields', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();

    const minimalMessageData = {
      message: {
        role: 'user',
        content: 'Minimal test message'
      },
      sender_id: testSenderId,
      recipient_id: testRecipientId
    };

    const response = await request.post(`/v1/threads/${createdThreadId}/messages`, {
      data: minimalMessageData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody.message.content).toBe(minimalMessageData.message.content);
    expect(responseBody.sender_id).toBe(minimalMessageData.sender_id);
    expect(responseBody.recipient_id).toBe(minimalMessageData.recipient_id);
    expect(responseBody).toHaveProperty('id');

    // Clean up the minimal message
    await request.delete(`/v1/threads/${createdThreadId}/messages/${responseBody.id}`);
  });

  test('should handle validation errors for update message - missing required fields', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();
    expect(createdMessageId).toBeTruthy();

    const invalidUpdateData = {
      // Missing required message field
    };

    const response = await request.put(`/v1/threads/${createdThreadId}/messages/${createdMessageId}`, {
      data: invalidUpdateData
    });

    expect(response.status()).toBe(400);
  });

  test('should return 404 for messages in non-existent thread', async ({ request }) => {
    const nonExistentThreadId = '12345678-1234-1234-1234-123456789016';

    const response = await request.get(`/v1/threads/${nonExistentThreadId}/messages`);

    expect(response.status()).toBe(404);
  });

  test('should return 404 when creating message in non-existent thread', async ({ request }) => {
    const nonExistentThreadId = '12345678-1234-1234-1234-123456789017';
    const messageData = {
      message: {
        role: 'user',
        content: 'Message in non-existent thread'
      },
      sender_id: testSenderId,
      recipient_id: testRecipientId
    };

    const response = await request.post(`/v1/threads/${nonExistentThreadId}/messages`, {
      data: messageData
    });

    expect(response.status()).toBe(404);
  });

  test('should delete a message', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();
    expect(createdMessageId).toBeTruthy();

    const response = await request.delete(`/v1/threads/${createdThreadId}/messages/${createdMessageId}`);

    expect(response.status()).toBe(204);

    // Verify message is deleted by trying to fetch it
    const getResponse = await request.get(`/v1/threads/${createdThreadId}/messages/${createdMessageId}`);
    expect(getResponse.status()).toBe(404);

    // Clear the ID so cleanup doesn't try to delete again
    createdMessageId = '';
  });

  test('should return 404 when deleting non-existent message', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();

    const nonExistentId = '12345678-1234-1234-1234-123456789def';
    const response = await request.delete(`/v1/threads/${createdThreadId}/messages/${nonExistentId}`);

    expect(response.status()).toBe(404);
  });

  test('should return 404 when deleting message from non-existent thread', async ({ request }) => {
    const nonExistentThreadId = '12345678-1234-1234-1234-123456789018';
    const nonExistentMessageId = '12345678-1234-1234-1234-123456789019';

    const response = await request.delete(`/v1/threads/${nonExistentThreadId}/messages/${nonExistentMessageId}`);

    expect(response.status()).toBe(404);
  });
});
