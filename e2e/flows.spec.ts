import { test, expect } from '@playwright/test';

test.describe.serial('Flows API', () => {
  let createdFlowId: string;

  test.beforeAll(async ({ request }) => {
    // Clean up any existing test flows before starting
    const response = await request.get('/v1/flows');
    if (response.ok()) {
      const responseBody = await response.json();
      const flows = responseBody.data || [];
      if (Array.isArray(flows)) {
        for (const flow of flows) {
          if (flow.name && flow.name.startsWith('Test Flow')) {
            await request.delete(`/v1/flows/${flow.id}`);
          }
        }
      }
    }
  });

  test.afterAll(async ({ request }) => {
    // Clean up created flow
    if (createdFlowId) {
      await request.delete(`/v1/flows/${createdFlowId}`);
    }
  });

  test('should create a new flow', async ({ request }) => {
    const flowData = {
      name: 'Test Flow for API Testing',
      description: 'A test flow created via Playwright API tests',
      parameters_schema: {
        type: 'object',
        properties: {
          input_text: {
            type: 'string',
            description: 'Input text to process'
          }
        },
        required: ['input_text']
      },
      engine: 'python',
      code_location: 's3://test-bucket/test-flow.py',
      entrypoint: 'main',
      additional_info: {
        version: '1.0.0',
        author: 'API Test Suite'
      },
      tags: ['test', 'api', 'automation']
    };

    const response = await request.post('/v1/flows', {
      data: flowData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('id');
    expect(responseBody.name).toBe(flowData.name);
    expect(responseBody.description).toBe(flowData.description);
    expect(responseBody.engine).toBe(flowData.engine);

    createdFlowId = responseBody.id;
  });

  test('should get all flows', async ({ request }) => {
    const response = await request.get('/v1/flows');

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('flows');
    expect(responseBody.flows).toBeDefined();
    expect(Array.isArray(responseBody.flows)).toBe(true);
  });

  test('should get flow by ID', async ({ request }) => {
    expect(createdFlowId).toBeTruthy();

    const response = await request.get(`/v1/flows/${createdFlowId}`);

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdFlowId);
    expect(responseBody.name).toBe('Test Flow for API Testing');
  });

  test('should return 404 for non-existent flow', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789012';
    const response = await request.get(`/v1/flows/${nonExistentId}`);

    expect(response.status()).toBe(404);
  });

  test('should return 400 for invalid flow ID format', async ({ request }) => {
    const invalidId = 'invalid-uuid';
    const response = await request.get(`/v1/flows/${invalidId}`);

    expect(response.status()).toBe(400);
  });

  test('should return 404 for nil UUID (validation error)', async ({ request }) => {
    const nilUuid = '00000000-0000-0000-0000-000000000000';
    const response = await request.get(`/v1/flows/${nilUuid}`);

    expect(response.status()).toBe(404);
    
    const responseBody = await response.json();
    expect(responseBody.resource).toBe("Flow");
    expect(responseBody.id).toBe(nilUuid);
  });

  test('should update an existing flow', async ({ request }) => {
    expect(createdFlowId).toBeTruthy();

    const updateData = {
      name: 'Updated Test Flow',
      description: 'Updated description for the test flow',
      parameters_schema: {
        type: 'object',
        properties: {
          input_text: {
            type: 'string',
            description: 'Updated input text description'
          },
          optional_param: {
            type: 'string',
            description: 'Optional parameter'
          }
        },
        required: ['input_text']
      },
      engine: 'python',
      additional_info: {
        version: '1.1.0',
        author: 'API Test Suite',
        updated: true
      },
      tags: ['test', 'api', 'automation', 'updated']
    };

    const response = await request.put(`/v1/flows/${createdFlowId}`, {
      data: updateData
    });

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
  });

  test('should return 404 when updating non-existent flow', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789abc';
    const updateData = {
      name: 'Non-existent Flow',
      engine: 'python'
    };

    const response = await request.put(`/v1/flows/${nonExistentId}`, {
      data: updateData
    });

    expect(response.status()).toBe(404);
  });

  test('should return 404 when executing non-existent flow', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789def';
    const executeData = {
      parameters: {
        input_text: 'Test execution'
      }
    };

    const response = await request.post(`/v1/flows/${nonExistentId}/execute`, {
      data: executeData
    });

    expect(response.status()).toBe(404);
  });

  test('should return 404 for non-existent flow run status', async ({ request }) => {
    const nonExistentRunId = '12345678-1234-1234-1234-123456789def';
    const response = await request.get(`/v1/flows/${nonExistentRunId}/status`);

    expect(response.status()).toBe(404);
  });

  test('should handle validation errors for create flow', async ({ request }) => {
    const invalidFlowData = {
      // Missing required name field
      description: 'Flow without name',
      engine: 'python'
    };

    const response = await request.post('/v1/flows', {
      data: invalidFlowData
    });

    expect(response.status()).toBe(400);
  });
  
  test('should delete a flow', async ({ request }) => {
    expect(createdFlowId).toBeTruthy();

    const response = await request.delete(`/v1/flows/${createdFlowId}`);

    expect(response.status()).toBe(204);

    // Verify flow is deleted by trying to fetch it
    const getResponse = await request.get(`/v1/flows/${createdFlowId}`);
    expect(getResponse.status()).toBe(404);

    // Clear the ID so cleanup doesn't try to delete again
    createdFlowId = '';
  });
});