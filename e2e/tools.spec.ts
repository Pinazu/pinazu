import { test, expect } from '@playwright/test';
import { randomInt } from 'crypto';

test.describe.serial('Tools API', () => {
  let createdStandaloneToolId: string;
  let createdWorkflowToolId: string;
  let createdMCPToolId: string;

  test.beforeAll(async ({ request }) => {
    // Clean up any existing test tools before starting
    const response = await request.get('/v1/tools');
    if (response.ok()) {
      const tools = await response.json();
      if (Array.isArray(tools)) {
        for (const tool of tools) {
          if (tool.name && tool.name.startsWith('Test Tool')) {
            await request.delete(`/v1/tools/${tool.id}`);
          }
        }
      }
    }
  });

  test.afterAll(async ({ request }) => {
    // Clean up created tools
    const toolIds = [createdStandaloneToolId, createdWorkflowToolId, createdMCPToolId].filter(Boolean);
    for (const toolId of toolIds) {
      await request.delete(`/v1/tools/${toolId}`);
    }
  });

  test('should create a new standalone tool', async ({ request }) => {
    const toolData = {
      name: `Test Tool - Standalone API`,
      description: 'A test standalone tool created via Playwright API tests',
      config: {
        type: 'standalone',
        url: 'https://api.example.com/tool',
        params: {
          type: 'object',
          properties: {
            input: {
              type: 'string',
              description: 'Input parameter for the tool'
            },
            config: {
              type: 'object',
              description: 'Configuration object'
            }
          },
          required: ['input']
        },
        api_key: 'test-api-key-123'
      }
    };

    const response = await request.post('/v1/tools', {
      data: toolData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    console.log('Created standalone tool:', responseBody);
    expect(responseBody).toHaveProperty('id');
    expect(responseBody.name).toBe(toolData.name);
    expect(responseBody.description).toBe(toolData.description);
    expect(responseBody.config.type).toBe('standalone');
    expect(responseBody.config.url).toBe(toolData.config.url);

    createdStandaloneToolId = responseBody.id;
  });

  test('should create a new workflow tool', async ({ request }) => {
    const toolData = {
      name: 'Test Tool - Workflow API',
      description: 'A test workflow tool created via Playwright API tests',
      config: {
        type: 'workflow',
        s3_url: 's3://test-bucket/workflow.py',
        params: {
          type: 'object',
          properties: {
            workflow_input: {
              type: 'string',
              description: 'Input for the workflow'
            },
            execution_mode: {
              type: 'string',
              enum: ['sync', 'async'],
              description: 'Execution mode'
            }
          },
          required: ['workflow_input']
        }
      }
    };

    const response = await request.post('/v1/tools', {
      data: toolData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('id');
    expect(responseBody.name).toBe(toolData.name);
    expect(responseBody.description).toBe(toolData.description);
    expect(responseBody.config.type).toBe('workflow');
    expect(responseBody.config.s3_url).toBe(toolData.config.s3_url);

    createdWorkflowToolId = responseBody.id;
  });

  test('should create a new MCP tool', async ({ request }) => {
    const toolData = {
      name: 'Test Tool - MCP API',
      description: 'A test MCP tool created via Playwright API tests',
      config: {
        type: 'mcp',
        entrypoint: '/usr/bin/python',
        protocol: 'stdio',
        env_vars: {
          'TOOL_CONFIG': 'production',
          'DEBUG_MODE': 'false'
        },
        api_key: 'mcp-api-key-456'
      }
    };

    const response = await request.post('/v1/tools', {
      data: toolData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('id');
    expect(responseBody.name).toBe(toolData.name);
    expect(responseBody.description).toBe(toolData.description);
    expect(responseBody.config.type).toBe('mcp');
    expect(responseBody.config.entrypoint).toBe(toolData.config.entrypoint);
    expect(responseBody.config.protocol).toBe(toolData.config.protocol);

    createdMCPToolId = responseBody.id;
  });

  test('should get all tools', async ({ request }) => {
    const response = await request.get('/v1/tools');

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('tools');
    expect(Array.isArray(responseBody.tools)).toBe(true);
    expect(responseBody.tools.length).toBeGreaterThanOrEqual(3); // At least our created tools
  });

  test('should get standalone tool by ID', async ({ request }) => {
    expect(createdStandaloneToolId).toBeTruthy();

    const response = await request.get(`/v1/tools/${createdStandaloneToolId}`);

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdStandaloneToolId);
    expect(responseBody.name).toBe('Test Tool - Standalone API');
    expect(responseBody.config.type).toBe('standalone');
  });

  test('should get workflow tool by ID', async ({ request }) => {
    expect(createdWorkflowToolId).toBeTruthy();

    const response = await request.get(`/v1/tools/${createdWorkflowToolId}`);

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdWorkflowToolId);
    expect(responseBody.name).toBe('Test Tool - Workflow API');
    expect(responseBody.config.type).toBe('workflow');
  });

  test('should get MCP tool by ID', async ({ request }) => {
    expect(createdMCPToolId).toBeTruthy();

    const response = await request.get(`/v1/tools/${createdMCPToolId}`);

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdMCPToolId);
    expect(responseBody.name).toBe('Test Tool - MCP API');
    expect(responseBody.config.type).toBe('mcp');
  });

  test('should return 404 for non-existent tool', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789012';
    const response = await request.get(`/v1/tools/${nonExistentId}`);

    expect(response.status()).toBe(404);
    
    const responseBody = await response.json();
    expect(responseBody.resource).toBe("Tool");
    expect(responseBody.id).toBe(nonExistentId);
  });

  test('should return 400 for invalid tool ID format', async ({ request }) => {
    const invalidId = 'invalid-uuid';
    const response = await request.get(`/v1/tools/${invalidId}`);

    expect(response.status()).toBe(400);
  });

  test('should return 404 for nil UUID (validation error)', async ({ request }) => {
    const nilUuid = '00000000-0000-0000-0000-000000000000';
    const response = await request.get(`/v1/tools/${nilUuid}`);

    expect(response.status()).toBe(404);
    
    const responseBody = await response.json();
    expect(responseBody.resource).toBe("Tool");
    expect(responseBody.id).toBe(nilUuid);
  });

  test('should update a standalone tool', async ({ request }) => {
    expect(createdStandaloneToolId).toBeTruthy();

    const updateData = {
      description: 'Updated description for standalone tool',
      config: {
        type: 'standalone',
        url: 'https://api.updated-example.com/tool',
        params: {
          type: 'object',
          properties: {
            input: {
              type: 'string',
              description: 'Updated input parameter'
            },
            config: {
              type: 'object',
              description: 'Updated configuration object'
            },
            new_param: {
              type: 'boolean',
              description: 'New boolean parameter'
            }
          },
          required: ['input']
        },
        api_key: 'updated-api-key-123'
      }
    };

    const response = await request.put(`/v1/tools/${createdStandaloneToolId}`, {
      data: updateData
    });

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdStandaloneToolId);
    expect(responseBody.description).toBe(updateData.description);
    expect(responseBody.config.url).toBe(updateData.config.url);
    expect(responseBody.config.api_key).toBe(updateData.config.api_key);
  });

  test('should update a workflow tool', async ({ request }) => {
    expect(createdWorkflowToolId).toBeTruthy();

    const updateData = {
      description: 'Updated description for workflow tool',
      config: {
        type: 'workflow',
        s3_url: 's3://updated-bucket/updated-workflow.py',
        params: {
          type: 'object',
          properties: {
            workflow_input: {
              type: 'string',
              description: 'Updated workflow input'
            },
            execution_mode: {
              type: 'string',
              enum: ['sync', 'async', 'batch'],
              description: 'Updated execution mode with batch option'
            },
            timeout: {
              type: 'integer',
              description: 'Timeout in seconds'
            }
          },
          required: ['workflow_input']
        }
      }
    };

    const response = await request.put(`/v1/tools/${createdWorkflowToolId}`, {
      data: updateData
    });

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdWorkflowToolId);
    expect(responseBody.description).toBe(updateData.description);
    expect(responseBody.config.s3_url).toBe(updateData.config.s3_url);
  });

  test('should update an MCP tool', async ({ request }) => {
    expect(createdMCPToolId).toBeTruthy();

    const updateData = {
      description: 'Updated description for MCP tool',
      config: {
        type: 'mcp',
        entrypoint: '/usr/bin/python3',
        protocol: 'grpc',
        env_vars: {
          'TOOL_CONFIG': 'staging',
          'DEBUG_MODE': 'true',
          'LOG_LEVEL': 'debug'
        },
        api_key: 'updated-mcp-api-key-789'
      }
    };

    const response = await request.put(`/v1/tools/${createdMCPToolId}`, {
      data: updateData
    });

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdMCPToolId);
    expect(responseBody.description).toBe(updateData.description);
    expect(responseBody.config.entrypoint).toBe(updateData.config.entrypoint);
    expect(responseBody.config.protocol).toBe(updateData.config.protocol);
  });

  test('should return 404 when updating non-existent tool', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789abc';
    const updateData = {
      description: 'Non-existent tool update',
      config: {
        type: 'standalone',
        url: 'https://api.example.com/tool',
        params: {
          type: 'object',
          properties: {}
        }
      }
    };

    const response = await request.put(`/v1/tools/${nonExistentId}`, {
      data: updateData
    });

    expect(response.status()).toBe(404);
  });

  test('should handle validation errors for create tool - missing required fields', async ({ request }) => {
    const invalidToolData = {
      // Missing required name field
      description: 'Tool without name',
      config: {
        type: 'standalone',
        url: 'https://api.example.com/tool'
        // Missing required params field
      }
    };

    const response = await request.post('/v1/tools', {
      data: invalidToolData
    });

    expect(response.status()).toBe(400);
  });

  test('should handle validation errors for invalid tool type', async ({ request }) => {
    const invalidToolData = {
      name: 'Invalid Tool Type',
      description: 'Tool with invalid type',
      config: {
        type: 'invalid_type', // Invalid tool type
        url: 'https://api.example.com/tool',
        params: {
          type: 'object',
          properties: {}
        }
      }
    };

    const response = await request.post('/v1/tools', {
      data: invalidToolData
    });

    expect(response.status()).toBe(400);
  });

  test('should handle validation errors for invalid URL format in standalone tool', async ({ request }) => {
    const invalidToolData = {
      name: 'Invalid URL Tool',
      description: 'Tool with invalid URL',
      config: {
        type: 'standalone',
        url: 'invalid-url-format', // Invalid URL
        params: {
          type: 'object',
          properties: {}
        }
      }
    };

    const response = await request.post('/v1/tools', {
      data: invalidToolData
    });

    expect(response.status()).toBe(400);
  });

  test('should handle validation errors for invalid S3 URL in workflow tool', async ({ request }) => {
    const invalidToolData = {
      name: 'Invalid S3 URL Tool',
      description: 'Tool with invalid S3 URL',
      config: {
        type: 'workflow',
        s3_url: 'invalid-s3-url', // Invalid S3 URL format
        params: {
          type: 'object',
          properties: {}
        }
      }
    };

    const response = await request.post('/v1/tools', {
      data: invalidToolData
    });

    expect(response.status()).toBe(400);
  });

  test('should handle validation errors for invalid MCP protocol', async ({ request }) => {
    const invalidToolData = {
      name: 'Invalid MCP Protocol Tool',
      description: 'Tool with invalid MCP protocol',
      config: {
        type: 'mcp',
        entrypoint: '/usr/bin/python',
        protocol: 'invalid_protocol' // Invalid protocol
      }
    };

    const response = await request.post('/v1/tools', {
      data: invalidToolData
    });

    expect(response.status()).toBe(400);
  });

  test('should delete a standalone tool', async ({ request }) => {
    expect(createdStandaloneToolId).toBeTruthy();

    const response = await request.delete(`/v1/tools/${createdStandaloneToolId}`);

    expect(response.status()).toBe(204);

    // Verify tool is deleted by trying to fetch it
    const getResponse = await request.get(`/v1/tools/${createdStandaloneToolId}`);
    expect(getResponse.status()).toBe(404);

    // Clear the ID so cleanup doesn't try to delete again
    createdStandaloneToolId = '';
  });

  test('should delete a workflow tool', async ({ request }) => {
    expect(createdWorkflowToolId).toBeTruthy();

    const response = await request.delete(`/v1/tools/${createdWorkflowToolId}`);

    expect(response.status()).toBe(204);

    // Verify tool is deleted by trying to fetch it
    const getResponse = await request.get(`/v1/tools/${createdWorkflowToolId}`);
    expect(getResponse.status()).toBe(404);

    // Clear the ID so cleanup doesn't try to delete again
    createdWorkflowToolId = '';
  });

  test('should delete an MCP tool', async ({ request }) => {
    expect(createdMCPToolId).toBeTruthy();

    const response = await request.delete(`/v1/tools/${createdMCPToolId}`);

    expect(response.status()).toBe(204);

    // Verify tool is deleted by trying to fetch it
    const getResponse = await request.get(`/v1/tools/${createdMCPToolId}`);
    expect(getResponse.status()).toBe(404);

    // Clear the ID so cleanup doesn't try to delete again
    createdMCPToolId = '';
  });

  test('should return 404 when deleting non-existent tool', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789abc';
    const response = await request.delete(`/v1/tools/${nonExistentId}`);

    expect(response.status()).toBe(404);
  });
});