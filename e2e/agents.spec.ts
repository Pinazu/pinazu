import { test, expect } from '@playwright/test';

test.describe.serial('Agents API', () => {
  let createdAgentId: string;
  let createdPermissionId: string;

  test.beforeAll(async ({ request }) => {
    // Clean up any existing test agents before starting
    const response = await request.get('/v1/agents');
    if (response.ok()) {
      const responseBody = await response.json();
      const agents = responseBody.agents || [];
      if (Array.isArray(agents)) {
        for (const agent of agents) {
          if (agent.name && agent.name.startsWith('Test Agent')) {
            await request.delete(`/v1/agents/${agent.id}`);
          }
        }
      }
    }
  });

  test.afterAll(async ({ request }) => {
    // Clean up created agent
    if (createdAgentId) {
      await request.delete(`/v1/agents/${createdAgentId}`);
    }
  });

  test('should create a new agent', async ({ request }) => {
    const agentData = {
      name: 'Test Agent for API Testing',
      description: 'A test agent created via Playwright API tests',
      specs: `
model:
  provider: "anthropic"
  model_id: "claude-3-sonnet"
  max_tokens: 4096
  temperature: 0.7

system: |
  You are a helpful AI assistant for testing purposes.
  Respond clearly and concisely to user queries.
  
tools:
  - name: "calculator"
    description: "Perform basic arithmetic operations"
  - name: "weather"
    description: "Get weather information"

parameters:
  thinking_enabled: true
  debug_mode: false
      `
    };

    const response = await request.post('/v1/agents', {
      data: agentData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('id');
    expect(responseBody.name).toBe(agentData.name);
    expect(responseBody.description).toBe(agentData.description);
    expect(responseBody.specs).toBe(agentData.specs);
    expect(responseBody).toHaveProperty('created_at');
    expect(responseBody).toHaveProperty('updated_at');
    expect(responseBody).toHaveProperty('created_by');

    createdAgentId = responseBody.id;
  });

  test('should get all agents', async ({ request }) => {
    const response = await request.get('/v1/agents');

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('agents');
    expect(responseBody.agents).toBeDefined();
    expect(Array.isArray(responseBody.agents)).toBe(true);
    expect(responseBody.agents.length).toBeGreaterThanOrEqual(1); // At least our created agent
  });

  test('should get agent by ID', async ({ request }) => {
    expect(createdAgentId).toBeTruthy();

    const response = await request.get(`/v1/agents/${createdAgentId}`);

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdAgentId);
    expect(responseBody.name).toBe('Test Agent for API Testing');
    expect(responseBody.description).toBe('A test agent created via Playwright API tests');
    expect(responseBody.specs).toContain('claude-3-sonnet');
    expect(responseBody.specs).toContain('calculator');
  });

  test('should return 404 for non-existent agent', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789012';
    const response = await request.get(`/v1/agents/${nonExistentId}`);

    expect(response.status()).toBe(404);
    
    const responseBody = await response.json();
    expect(responseBody.resource).toBe("Agent");
    expect(responseBody.id).toBe(nonExistentId);
  });

  test('should return 400 for invalid agent ID format', async ({ request }) => {
    const invalidId = 'invalid-uuid';
    const response = await request.get(`/v1/agents/${invalidId}`);

    expect(response.status()).toBe(400);
  });

  test('should return 404 for nil UUID (validation error)', async ({ request }) => {
    const nilUuid = '00000000-0000-0000-0000-000000000000';
    const response = await request.get(`/v1/agents/${nilUuid}`);

    expect(response.status()).toBe(404);
    
    const responseBody = await response.json();
    expect(responseBody.resource).toBe("Agent");
    expect(responseBody.id).toBe(nilUuid);
  });

  test('should update an existing agent', async ({ request }) => {
    expect(createdAgentId).toBeTruthy();

    const updateData = {
      name: 'Updated Test Agent',
      description: 'Updated description for the test agent',
      specs: `
model:
  provider: "anthropic"
  model_id: "claude-3-opus"
  max_tokens: 8192
  temperature: 0.3

system: |
  You are an updated AI assistant for testing purposes.
  You have been modified with new capabilities.
  
tools:
  - name: "calculator"
    description: "Perform advanced arithmetic operations"
  - name: "weather"
    description: "Get detailed weather information"
  - name: "translator"
    description: "Translate text between languages"

parameters:
  thinking_enabled: true
  debug_mode: true
  max_iterations: 10
      `
    };

    const response = await request.put(`/v1/agents/${createdAgentId}`, {
      data: updateData
    });

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdAgentId);
    expect(responseBody.name).toBe(updateData.name);
    expect(responseBody.description).toBe(updateData.description);
    expect(responseBody.specs).toBe(updateData.specs);
    expect(responseBody.specs).toContain('claude-3-opus');
    expect(responseBody.specs).toContain('translator');
  });

  test('should return 404 when updating non-existent agent', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789abc';
    const updateData = {
      name: 'Non-existent Agent',
      description: 'This agent does not exist'
    };

    const response = await request.put(`/v1/agents/${nonExistentId}`, {
      data: updateData
    });

    expect(response.status()).toBe(404);
  });

  test('should handle validation errors for create agent - missing required fields', async ({ request }) => {
    const invalidAgentData = {
      // Missing required name field
      description: 'Agent without name',
      specs: 'system: "Test system prompt"'
    };

    const response = await request.post('/v1/agents', {
      data: invalidAgentData
    });

    expect(response.status()).toBe(400);
  });

  test('should handle validation errors for agent name too long', async ({ request }) => {
    const invalidAgentData = {
      name: 'A'.repeat(300), // Exceeds maxLength of 255
      description: 'Agent with very long name',
      specs: 'system: "Test system prompt"'
    };

    const response = await request.post('/v1/agents', {
      data: invalidAgentData
    });

    expect(response.status()).toBe(400);
  });

  test('should create agent with minimal required fields', async ({ request }) => {
    const minimalAgentData = {
      name: 'Minimal Test Agent'
    };

    const response = await request.post('/v1/agents', {
      data: minimalAgentData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody.name).toBe(minimalAgentData.name);
    expect(responseBody).toHaveProperty('id');

    // Clean up the minimal agent
    await request.delete(`/v1/agents/${responseBody.id}`);
  });

  test('should create agent with complex YAML specs', async ({ request }) => {
    const complexAgentData = {
      name: 'Complex YAML Test Agent',
      description: 'Testing complex YAML specifications',
      specs: `
model:
  provider: "anthropic"
  model_id: "claude-3-sonnet"
  max_tokens: 4096
  temperature: 0.5
  top_p: 0.9
  thinking:
    enabled: true
    budget_token: 2048

system: |
  You are a specialized AI assistant with the following capabilities:
  
  1. Code Analysis and Generation
  2. Data Processing and Transformation
  3. Multi-language Support
  
  Guidelines:
  - Always validate input parameters
  - Provide clear error messages
  - Use structured output formats
  
  Available Functions:
  - analyze_code(language, code)
  - process_data(format, data)
  - translate_text(source_lang, target_lang, text)

tools:
  - name: "code_analyzer"
    description: "Analyze code quality and suggest improvements"
    parameters:
      type: "object"
      properties:
        language:
          type: "string"
          enum: ["python", "javascript", "go", "rust"]
        code:
          type: "string"
          description: "The code to analyze"
      required: ["language", "code"]
      
  - name: "data_processor"
    description: "Process and transform structured data"
    parameters:
      type: "object"
      properties:
        format:
          type: "string"
          enum: ["json", "csv", "xml", "yaml"]
        operation:
          type: "string"
          enum: ["validate", "transform", "summarize"]
        data:
          type: "string"
          description: "The data to process"
      required: ["format", "operation", "data"]

parameters:
  thinking_enabled: true
  debug_mode: false
  max_iterations: 5
  timeout_seconds: 30
  retry_attempts: 3
  
context:
  domain: "software_development"
  expertise_level: "expert"
  output_format: "structured"
  
constraints:
  - "Do not execute arbitrary code"
  - "Validate all inputs thoroughly"
  - "Respect rate limits and quotas"
  - "Maintain user privacy and data security"
      `
    };

    const response = await request.post('/v1/agents', {
      data: complexAgentData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody.name).toBe(complexAgentData.name);
    expect(responseBody.specs).toBe(complexAgentData.specs);
    expect(responseBody.specs).toContain('code_analyzer');
    expect(responseBody.specs).toContain('data_processor');
    expect(responseBody.specs).toContain('software_development');

    // Clean up the complex agent
    await request.delete(`/v1/agents/${responseBody.id}`);
  });

  test('should handle invalid YAML specs gracefully', async ({ request }) => {
    const invalidYamlAgentData = {
      name: 'Invalid YAML Test Agent',
      description: 'Testing invalid YAML handling',
      specs: `
model:
  provider: "anthropic"
  model_id: "claude-3-sonnet"
  invalid_yaml: [unclosed array
system: |
  This YAML has syntax errors
      `
    };

    const response = await request.post('/v1/agents', {
      data: invalidYamlAgentData
    });

    // Should either accept it (since it's just stored as text) or return validation error
    expect([201, 400]).toContain(response.status());
    
    if (response.status() === 201) {
      const responseBody = await response.json();
      // Clean up if created
      await request.delete(`/v1/agents/${responseBody.id}`);
    }
  });

  test('should update agent with partial data', async ({ request }) => {
    expect(createdAgentId).toBeTruthy();

    // Only update the description
    const partialUpdateData = {
      description: 'Partially updated description'
    };

    const response = await request.put(`/v1/agents/${createdAgentId}`, {
      data: partialUpdateData
    });

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdAgentId);
    expect(responseBody.description).toBe(partialUpdateData.description);
    // Name should remain the same from previous update
    expect(responseBody.name).toBe('Updated Test Agent');
  });

  test('should delete an agent', async ({ request }) => {
    expect(createdAgentId).toBeTruthy();

    const response = await request.delete(`/v1/agents/${createdAgentId}`);

    expect(response.status()).toBe(204);

    // Verify agent is deleted by trying to fetch it
    const getResponse = await request.get(`/v1/agents/${createdAgentId}`);
    expect(getResponse.status()).toBe(404);

    // Clear the ID so cleanup doesn't try to delete again
    createdAgentId = '';
  });

  test('should return 404 when deleting non-existent agent', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789def';
    const response = await request.delete(`/v1/agents/${nonExistentId}`);

    expect(response.status()).toBe(404);
  });
});

test.describe.serial('Agent Permissions API', () => {
  let testAgentId: string;
  let testPermissionId: string;
  let testMappingId: string;

  test.beforeAll(async ({ request }) => {
    // Create a test agent for permissions testing
    const agentData = {
      name: 'Test Agent for Permissions',
      description: 'Agent created for testing permissions functionality',
      specs: 'system: "Test agent for permissions testing"'
    };

    const agentResponse = await request.post('/v1/agents', {
      data: agentData
    });

    expect(agentResponse.status()).toBe(201);
    const agentBody = await agentResponse.json();
    testAgentId = agentBody.id;
    console.log(`Created test agent with ID: ${testAgentId}`);

    // Test if we can retrieve the created agent
    const getResponse = await request.get(`/v1/agents/${testAgentId}`);
    expect(getResponse.status()).toBe(200);
    const getBody = await getResponse.json();
    expect(getBody.id).toBe(testAgentId);

    // Create a test permission for permissions testing
    const permissionData = {
      name: 'Test Permission for Agent Testing',
      description: 'Permission created for testing agent permissions functionality',
      content: {
        "action": "test_action",
        "resource": "test_resource",
        "conditions": []
      }
    };

    const permissionResponse = await request.post('/v1/permissions', {
      data: permissionData
    });

    expect(permissionResponse.status()).toBe(201);
    const permissionBody = await permissionResponse.json();
    testPermissionId = permissionBody.id;
    console.log(`Created test permission with ID: ${testPermissionId}`);
  });

  test.afterAll(async ({ request }) => {
    // Clean up test agent and permission
    if (testAgentId) {
      await request.delete(`/v1/agents/${testAgentId}`);
    }
    if (testPermissionId) {
      await request.delete(`/v1/permissions/${testPermissionId}`);
    }
  });

  test('should list permissions for agent (initially empty)', async ({ request }) => {
    expect(testAgentId).toBeTruthy();

    const response = await request.get(`/v1/agents/${testAgentId}/permissions`);

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(Array.isArray(responseBody.permissionMappings)).toBe(true);
    // Should be empty initially
    expect(responseBody.permissionMappings).toHaveLength(0);
  });

  test('should add permission to agent', async ({ request }) => {
    expect(testAgentId).toBeTruthy();
    expect(testPermissionId).toBeTruthy();

    const permissionData = {
      permission_id: testPermissionId
    };
    console.log(`Adding permission ${testPermissionId} to agent ${testAgentId}`);

    const response = await request.post(`/v1/agents/${testAgentId}/permissions`, {
      data: permissionData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('mapping_id');
    expect(responseBody.agent_id).toBe(testAgentId);
    expect(responseBody.permission_id).toBe(testPermissionId);
    expect(responseBody).toHaveProperty('assigned_at');
    expect(responseBody).toHaveProperty('assigned_by');

    testMappingId = responseBody.mapping_id;
  });

  test('should list permissions for agent (after adding)', async ({ request }) => {
    expect(testAgentId).toBeTruthy();
    expect(testPermissionId).toBeTruthy();

    const response = await request.get(`/v1/agents/${testAgentId}/permissions`);

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(Array.isArray(responseBody.permissionMappings)).toBe(true);
    expect(responseBody.permissionMappings.length).toBeGreaterThanOrEqual(1);

    const permission = responseBody.permissionMappings.find((p: any) => p.permission_id === testPermissionId);
    expect(permission).toBeDefined();
    expect(permission.agent_id).toBe(testAgentId);
  });

  test('should return 409 when adding duplicate permission', async ({ request }) => {
    expect(testAgentId).toBeTruthy();
    expect(testPermissionId).toBeTruthy();

    const permissionData = {
      permission_id: testPermissionId
    };

    const response = await request.post(`/v1/agents/${testAgentId}/permissions`, {
      data: permissionData
    });

    expect(response.status()).toBe(409);
  });

  test('should return 404 when adding permission to non-existent agent', async ({ request }) => {
    const nonExistentAgentId = '12345678-1234-1234-1234-123456789000';
    expect(testPermissionId).toBeTruthy();

    const permissionData = {
      permission_id: testPermissionId
    };

    const response = await request.post(`/v1/agents/${nonExistentAgentId}/permissions`, {
      data: permissionData
    });

    expect(response.status()).toBe(404);
  });

  test('should return 404 when adding non-existent permission', async ({ request }) => {
    expect(testAgentId).toBeTruthy();

    const nonExistentPermissionId = '12345678-1234-1234-1234-123456789001';
    const permissionData = {
      permission_id: nonExistentPermissionId
    };

    const response = await request.post(`/v1/agents/${testAgentId}/permissions`, {
      data: permissionData
    });

    expect(response.status()).toBe(404);
  });

  test('should handle validation errors for add permission - missing fields', async ({ request }) => {
    expect(testAgentId).toBeTruthy();

    const invalidData = {
      // Missing required permission_id field
    };

    const response = await request.post(`/v1/agents/${testAgentId}/permissions`, {
      data: invalidData
    });

    expect(response.status()).toBe(400);
  });

  test('should remove permission from agent', async ({ request }) => {
    expect(testAgentId).toBeTruthy();
    expect(testPermissionId).toBeTruthy();

    const response = await request.delete(`/v1/agents/${testAgentId}/permissions/${testPermissionId}`);

    expect(response.status()).toBe(204);

    // Verify permission is removed
    const listResponse = await request.get(`/v1/agents/${testAgentId}/permissions`);
    expect(listResponse.status()).toBe(200);
    
    const listBody = await listResponse.json();
    const permission = listBody.permissionMappings.find((p: any) => p.permission_id === testPermissionId);
    expect(permission).toBeUndefined();
  });

  test('should return 404 when removing permission from non-existent agent', async ({ request }) => {
    const nonExistentAgentId = '12345678-1234-1234-1234-123456789002';
    const nonExistentPermissionId = '12345678-1234-1234-1234-123456789003';

    const response = await request.delete(`/v1/agents/${nonExistentAgentId}/permissions/${nonExistentPermissionId}`);

    expect(response.status()).toBe(404);
  });

  test('should return 404 when removing non-existent permission', async ({ request }) => {
    expect(testAgentId).toBeTruthy();

    const nonExistentPermissionId = '12345678-1234-1234-1234-123456789004';

    const response = await request.delete(`/v1/agents/${testAgentId}/permissions/${nonExistentPermissionId}`);

    expect(response.status()).toBe(404);
  });

  test('should return 404 for permissions on non-existent agent', async ({ request }) => {
    const nonExistentAgentId = '12345678-1234-1234-1234-123456789005';

    const response = await request.get(`/v1/agents/${nonExistentAgentId}/permissions`);

    expect(response.status()).toBe(404);
  });
});