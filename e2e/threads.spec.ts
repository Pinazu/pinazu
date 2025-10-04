import { test, expect } from '@playwright/test';

test.describe.serial('Threads API', () => {
  let createdThreadId: string;
  let testUserId: string = '550e8400-c95b-4444-6666-446655440000'; // Admin user UUID from database

  test.beforeAll(async ({ request }) => {
    // Clean up any existing test threads before starting
    const response = await request.get('/v1/threads');
    if (response.ok()) {
      const responseBody = await response.json();
      const threads = responseBody.threads || [];
      if (Array.isArray(threads)) {
        for (const thread of threads) {
          if (thread.title && thread.title.startsWith('Test Thread')) {
            await request.delete(`/v1/threads/${thread.id}`);
          }
        }
      }
    }
  });

  test.afterAll(async ({ request }) => {
    // Clean up created thread
    if (createdThreadId) {
      await request.delete(`/v1/threads/${createdThreadId}`);
    }
  });

  test('should create a new thread', async ({ request }) => {
    const threadData = {
      title: 'Test Thread for API Testing',
      user_id: testUserId
    };

    const response = await request.post('/v1/threads', {
      data: threadData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('id');
    expect(responseBody.title).toBe(threadData.title);
    expect(responseBody.user_id).toBe(threadData.user_id);
    expect(responseBody).toHaveProperty('created_at');
    expect(responseBody).toHaveProperty('updated_at');

    createdThreadId = responseBody.id;
  });

  test('should get all threads', async ({ request }) => {
    const response = await request.get('/v1/threads');

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('threads');
    expect(responseBody.threads).toBeDefined();
    expect(Array.isArray(responseBody.threads)).toBe(true);
    expect(responseBody.threads.length).toBeGreaterThanOrEqual(1); // At least our created thread
  });

  test('should get thread by ID', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();

    const response = await request.get(`/v1/threads/${createdThreadId}`);

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdThreadId);
    expect(responseBody.title).toBe('Test Thread for API Testing');
    expect(responseBody.user_id).toBe(testUserId);
    expect(responseBody).toHaveProperty('created_at');
    expect(responseBody).toHaveProperty('updated_at');
  });

  test('should return 404 for non-existent thread', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789012';
    const response = await request.get(`/v1/threads/${nonExistentId}`);

    expect(response.status()).toBe(404);
    
    const responseBody = await response.json();
    expect(responseBody.resource).toBe("Thread");
    expect(responseBody.id).toBe(nonExistentId);
  });

  test('should return 400 for invalid thread ID format', async ({ request }) => {
    const invalidId = 'invalid-uuid';
    const response = await request.get(`/v1/threads/${invalidId}`);

    expect(response.status()).toBe(400);
  });

  test('should return 404 for nil UUID (validation error)', async ({ request }) => {
    const nilUuid = '00000000-0000-0000-0000-000000000000';
    const response = await request.get(`/v1/threads/${nilUuid}`);

    expect(response.status()).toBe(404);
    
    const responseBody = await response.json();
    expect(responseBody.resource).toBe("Thread");
    expect(responseBody.id).toBe(nilUuid);
  });

  test('should update an existing thread title', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();

    const updateData = {
      title: 'Updated Test Thread Title'
    };

    const response = await request.put(`/v1/threads/${createdThreadId}`, {
      data: updateData
    });

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdThreadId);
    expect(responseBody.title).toBe(updateData.title);
    expect(responseBody.user_id).toBe(testUserId);
    expect(responseBody).toHaveProperty('updated_at');
  });

  test('should return 404 when updating non-existent thread', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789abc';
    const updateData = {
      title: 'Non-existent Thread Title'
    };

    const response = await request.put(`/v1/threads/${nonExistentId}`, {
      data: updateData
    });

    expect(response.status()).toBe(404);
  });

  test('should handle validation errors for create thread - missing required fields', async ({ request }) => {
    const invalidThreadData = {
      // Missing required title field
      user_id: testUserId
    };

    const response = await request.post('/v1/threads', {
      data: invalidThreadData
    });

    expect(response.status()).toBe(400);
  });

  test('should handle validation errors for thread title too long', async ({ request }) => {
    const invalidThreadData = {
      title: 'A'.repeat(300), // Exceeds maxLength of 255
      user_id: testUserId
    };

    const response = await request.post('/v1/threads', {
      data: invalidThreadData
    });

    expect(response.status()).toBe(400);
  });

  test('should handle validation errors for invalid user_id format', async ({ request }) => {
    const invalidThreadData = {
      title: 'Valid Thread Title',
      user_id: 'invalid-uuid-format'
    };

    const response = await request.post('/v1/threads', {
      data: invalidThreadData
    });

    expect(response.status()).toBe(400);
  });

  test('should create thread with minimal required fields', async ({ request }) => {
    const minimalThreadData = {
      title: 'Minimal Test Thread',
      user_id: testUserId
    };

    const response = await request.post('/v1/threads', {
      data: minimalThreadData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody.title).toBe(minimalThreadData.title);
    expect(responseBody.user_id).toBe(minimalThreadData.user_id);
    expect(responseBody).toHaveProperty('id');

    // Clean up the minimal thread
    await request.delete(`/v1/threads/${responseBody.id}`);
  });

  test('should update thread with partial data', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();

    // Only update the title
    const partialUpdateData = {
      title: 'Partially Updated Thread Title'
    };

    const response = await request.put(`/v1/threads/${createdThreadId}`, {
      data: partialUpdateData
    });

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdThreadId);
    expect(responseBody.title).toBe(partialUpdateData.title);
    // User ID should remain the same
    expect(responseBody.user_id).toBe(testUserId);
  });

  test('should handle validation errors for update thread - missing required fields', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();

    const invalidUpdateData = {
      // Missing required title field
    };

    const response = await request.put(`/v1/threads/${createdThreadId}`, {
      data: invalidUpdateData
    });

    expect(response.status()).toBe(400);
  });

  test('should handle validation errors for update thread title too long', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();

    const invalidUpdateData = {
      title: 'B'.repeat(300) // Exceeds maxLength of 255
    };

    const response = await request.put(`/v1/threads/${createdThreadId}`, {
      data: invalidUpdateData
    });

    expect(response.status()).toBe(400);
  });

  test('should delete a thread', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();

    const response = await request.delete(`/v1/threads/${createdThreadId}`);

    expect(response.status()).toBe(204);

    // Verify thread is deleted by trying to fetch it
    const getResponse = await request.get(`/v1/threads/${createdThreadId}`);
    expect(getResponse.status()).toBe(404);

    // Clear the ID so cleanup doesn't try to delete again
    createdThreadId = '';
  });

  test('should return 404 when deleting non-existent thread', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789def';
    const response = await request.delete(`/v1/threads/${nonExistentId}`);

    expect(response.status()).toBe(404);
  });
});