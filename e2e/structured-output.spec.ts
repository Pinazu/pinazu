import { test, expect } from '@playwright/test';
// @ts-ignore - WebSocket import configuration
import WebSocket from 'ws';
// @ts-ignore - NATS types not available
import { connect, NatsConnection } from 'nats';

// Test configuration constants
const WEBSOCKET_URL = 'ws://localhost:8080/v1/ws';
const NATS_URL = 'nats://localhost:4222';
const DEFAULT_TIMEOUT = 60000;
const SHORT_TIMEOUT = 10000;

test.describe.serial('Structured Output E2E Integration', () => {
  let natsConnection: NatsConnection;
  let testUserId: string = '550e8400-c95b-4444-6666-446655440000'; // Admin user UUID
  let createdAgentIds: string[] = [];
  let createdThreadId: string;

  // Schema for person information
  const personSchema = {
    type: "object",
    properties: {
      name: {
        type: "string",
        description: "Full name of the person"
      },
      age: {
        type: "integer",
        minimum: 0,
        maximum: 150,
        description: "Age in years"
      },
      email: {
        type: "string",
        format: "email",
        description: "Email address"
      },
      occupation: {
        type: "string",
        description: "Job title or profession"
      },
      skills: {
        type: "array",
        items: {
          type: "string"
        },
        description: "List of skills"
      }
    },
    required: ["name", "age", "email", "occupation", "skills"],
    additionalProperties: false
  };

  // Agent configurations for testing structured output
  const structuredOutputAgents = [
    {
      name: 'Structured Output Anthropic Test Agent',
      description: 'Anthropic Claude with structured JSON output',
      specs: `
model:
  provider: "bedrock/anthropic"
  model_id: "anthropic.claude-3-haiku-20240307-v1:0"
  max_tokens: 2048
  stream: true
  response_format: ${JSON.stringify(personSchema, null, 2)}

system: |
  You are a helpful AI assistant that generates structured data.
  When asked to create a person, generate realistic sample data.

tool_refs: []
`,
      expectedProvider: 'anthropic'
    }
  ];

  test.beforeAll(async ({ request }) => {
    // Connect to NATS
    natsConnection = await connect({ servers: NATS_URL });
    console.log('Connected to NATS server');

    // Create a thread for testing
    const threadData = {
      title: 'Structured Output Test Thread',
      user_id: testUserId
    };

    const threadResponse = await request.post('/v1/threads', {
      data: threadData
    });

    expect(threadResponse.status()).toBe(201);
    const threadBody = await threadResponse.json();
    createdThreadId = threadBody.id;
    console.log(`Created test thread with ID: ${createdThreadId}`);

    // Clean up any existing test agents
    const agentsResponse = await request.get('/v1/agents');
    if (agentsResponse.ok()) {
      const agentsBody = await agentsResponse.json();
      const agents = agentsBody.agents || [];
      for (const agent of agents) {
        if (agent.name && (
          agent.name.includes('Structured Output') && agent.name.includes('Test Agent') ||
          agent.name.includes('Complex Structured Output Test Agent')
        )) {
          await request.delete(`/v1/agents/${agent.id}`);
          console.log(`Cleaned up existing test agent: ${agent.name}`);
        }
      }
    }

    // Create test agents for structured output
    for (const agentConfig of structuredOutputAgents) {
      const response = await request.post('/v1/agents', {
        data: agentConfig
      });

      expect(response.status()).toBe(201);
      const responseBody = await response.json();
      createdAgentIds.push(responseBody.id);
      console.log(`Created agent ${agentConfig.name} with ID: ${responseBody.id}`);
    }
  });

  test.afterAll(async ({ request }) => {
    // Clean up created agents
    for (const agentId of createdAgentIds) {
      await request.delete(`/v1/agents/${agentId}`);
      console.log(`Cleaned up agent: ${agentId}`);
    }

    // Clean up thread
    if (createdThreadId) {
      await request.delete(`/v1/threads/${createdThreadId}`);
      console.log(`Cleaned up thread: ${createdThreadId}`);
    }

    // Close NATS connection
    if (natsConnection) {
      await natsConnection.close();
      console.log('Closed NATS connection');
    }
  });

  // Helper function to validate JSON structure against schema
  function validatePersonSchema(jsonData: any): boolean {
    if (typeof jsonData !== 'object' || jsonData === null) return false;
    
    // Check required fields
    const requiredFields = ['name', 'age', 'email', 'occupation', 'skills'];
    for (const field of requiredFields) {
      if (!(field in jsonData)) {
        console.error(`Missing required field: ${field}`);
        return false;
      }
    }

    // Validate field types
    if (typeof jsonData.name !== 'string') {
      console.error('name must be a string');
      return false;
    }

    if (typeof jsonData.age !== 'number' || !Number.isInteger(jsonData.age) || jsonData.age < 0 || jsonData.age > 150) {
      console.error('age must be an integer between 0 and 150');
      return false;
    }

    if (typeof jsonData.email !== 'string' || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(jsonData.email)) {
      console.error('email must be a valid email format');
      return false;
    }

    if (typeof jsonData.occupation !== 'string') {
      console.error('occupation must be a string');
      return false;
    }

    if (!Array.isArray(jsonData.skills) || !jsonData.skills.every((skill: any) => typeof skill === 'string')) {
      console.error('skills must be an array of strings');
      return false;
    }

    return true;
  }

  // Helper function to test structured output via WebSocket
  async function testStructuredOutputWebSocket(agentId: string, agentName: string): Promise<void> {
    return new Promise((resolve, reject) => {
      const ws = new WebSocket(WEBSOCKET_URL);
      let accumulatedContent = '{'; // Initialize with opening brace since prefill is not captured in deltas
      let messageReceived = false;
      let jsonValidated = false;

      const timeout = setTimeout(() => {
        ws.close();
        if (!messageReceived) {
          reject(new Error(`No response received from ${agentName} within timeout`));
        } else if (!jsonValidated) {
          reject(new Error(`Invalid or incomplete JSON received from ${agentName}: ${accumulatedContent}`));
        }
      }, DEFAULT_TIMEOUT);

      ws.on('error', (error: Error) => {
        clearTimeout(timeout);
        reject(new Error(`WebSocket error for ${agentName}: ${error.message}`));
      });

      ws.on('open', () => {
        console.log(`WebSocket connected for ${agentName}`);
        
        const message = {
          agent_id: agentId,
          thread_id: createdThreadId,
          messages: [{
            role: 'user',
            content: [{type:'text', text: 'Create a person profile for John Smith, a 30-year-old software engineer who knows Python, JavaScript, and Go. Use john.smith@example.com as email.'}]
          }]
        };

        ws.send(JSON.stringify(message));
        console.log(`Sent structured output request to ${agentName}`);
      });

      ws.on('message', (data: WebSocket.RawData) => {
        try {
          const response = JSON.parse(data.toString());
          const messageType = response.message?.type;
          console.log(`Received message for ${agentName}:`, messageType);

          if (messageType === 'content_block_delta') {
            console.log(`Full content_block_delta message for ${agentName}:`, JSON.stringify(response.message, null, 2));
            
            // Handle different provider formats
            const provider = response.message?.provider;
            let deltaText = '';
            
            if (provider === 'anthropic') {
              // Anthropic format: response.message.delta.text
              if (response.message?.delta?.text) {
                deltaText = response.message.delta.text;
              }
            } else if (provider === 'bedrock') {
              // Bedrock format might have delta structured differently
              // Based on the debug output, it seems Bedrock has empty delta.text
              // For now, we'll try to extract from other fields or skip empty deltas
              if (response.message?.delta?.text) {
                deltaText = response.message.delta.text;
              } else if (response.message?.delta?.partial_json) {
                deltaText = response.message.delta.partial_json;
              }
              // If still no text and this is bedrock, it might be an empty delta - skip it
              if (!deltaText) {
                console.log(`Skipping empty delta for Bedrock provider in ${agentName}`);
                return;
              }
            }
            
            if (deltaText) {
              console.log(`Content delta for ${agentName} (${provider}):`, JSON.stringify(deltaText));
              accumulatedContent += deltaText;
              messageReceived = true;
              console.log(`Current accumulated content for ${agentName} (${accumulatedContent.length} chars):`, JSON.stringify(accumulatedContent.substring(0, 50)) + (accumulatedContent.length > 50 ? '...' : ''));
            } else {
              console.log(`No delta text found in content_block_delta for ${agentName} (${provider})`);
            }
          }

          // Check if we have complete JSON when message stops
          if (messageType === 'message_stop') {
            console.log(`Final accumulated content for ${agentName}:`, accumulatedContent);
            
            try {
              // Try to parse the accumulated content as JSON
              const jsonData = JSON.parse(accumulatedContent.trim());
              console.log(`Parsed JSON for ${agentName}:`, jsonData);

              // Validate against schema
              if (validatePersonSchema(jsonData)) {
                console.log(`✅ Valid JSON structure received from ${agentName}`);
                jsonValidated = true;
                clearTimeout(timeout);
                ws.close();
                resolve();
              } else {
                throw new Error('JSON does not match expected schema');
              }
            } catch (parseError) {
              console.error(`❌ Failed to parse JSON from ${agentName}:`, parseError);
              console.error('Raw content:', accumulatedContent);
              clearTimeout(timeout);
              ws.close();
              const errorMessage = parseError instanceof Error ? parseError.message : String(parseError);
              reject(new Error(`Invalid JSON from ${agentName}: ${errorMessage}`));
            }
          }
        } catch (error) {
          console.error(`Error processing message from ${agentName}:`, error);
        }
      });

      ws.on('close', () => {
        console.log(`WebSocket closed for ${agentName}`);
        if (!jsonValidated && messageReceived) {
          clearTimeout(timeout);
          reject(new Error(`Connection closed before valid JSON was received from ${agentName}`));
        }
      });
    });
  }

  // Test structured output for each provider
  for (let i = 0; i < structuredOutputAgents.length; i++) {
    const agentConfig = structuredOutputAgents[i];
    
    test(`should generate valid JSON structure via WebSocket - ${agentConfig.name}`, async () => {
      const agentId = createdAgentIds[i];
      expect(agentId).toBeTruthy();

      console.log(`\\n🧪 Testing structured output for: ${agentConfig.name}`);
      console.log(`Agent ID: ${agentId}`);
      console.log(`Expected Provider: ${agentConfig.expectedProvider}`);

      await testStructuredOutputWebSocket(agentId, agentConfig.name);
    });
  }

  // Test edge cases
  test('should handle complex nested JSON structure - Anthropic', async ({ request }) => {
    // Create agent with complex schema
    const complexSchema = {
      type: "object",
      properties: {
        company: {
          type: "object",
          properties: {
            name: { type: "string" },
            employees: {
              type: "array",
              items: {
                type: "object",
                properties: {
                  name: { type: "string" },
                  department: { type: "string" },
                  salary: { type: "number" }
                },
                required: ["name", "department", "salary"]
              }
            }
          },
          required: ["name", "employees"]
        },
        metadata: {
          type: "object",
          properties: {
            created_at: { type: "string", format: "date-time" },
            version: { type: "string" }
          },
          required: ["created_at", "version"]
        }
      },
      required: ["company", "metadata"]
    };

    const complexAgentData = {
      name: `Complex Structured Output Test Agent ${Date.now()}`,
      description: 'Agent for testing complex nested JSON structures',
      specs: `
model:
  provider: "bedrock/anthropic"
  model_id: "anthropic.claude-3-haiku-20240307-v1:0"
  max_tokens: 4096
  stream: true
  response_format: ${JSON.stringify(complexSchema, null, 2)}

system: |
  You are a helpful AI assistant that generates complex structured data.
  You must respond with valid JSON that matches the provided schema exactly.
  Generate realistic sample data for a small tech company.

tool_refs: []
`
    };

    const response = await request.post('/v1/agents', {
      data: complexAgentData
    });

    expect(response.status()).toBe(201);
    const agentBody = await response.json();
    const complexAgentId = agentBody.id;

    try {
      await new Promise<void>((resolve, reject) => {
        const ws = new WebSocket(WEBSOCKET_URL);
        let accumulatedContent = '{'; // Initialize with opening brace since prefill is not captured in deltas

        const timeout = setTimeout(() => {
          ws.close();
          reject(new Error('Complex JSON test timeout'));
        }, DEFAULT_TIMEOUT);

        ws.on('open', () => {
          const message = {
            agent_id: complexAgentId,
            thread_id: createdThreadId,
            messages: [{
              role: 'user',
              content: [{
                type: 'text',
                text: 'Generate data for a tech company called "TechCorp" with 3 employees in different departments.'
              }]
            }]
          };
          ws.send(JSON.stringify(message));
        });

        ws.on('message', (data: WebSocket.RawData) => {
          const response = JSON.parse(data.toString());
          const messageType = response.message?.type;
          
          if (messageType === 'content_block_delta') {
            // Handle different provider formats
            const provider = response.message?.provider;
            let deltaText = '';
            
            if (provider === 'anthropic') {
              if (response.message?.delta?.text) {
                deltaText = response.message.delta.text;
              }
            } else if (provider === 'bedrock') {
              if (response.message?.delta?.text) {
                deltaText = response.message.delta.text;
              } else if (response.message?.delta?.partial_json) {
                deltaText = response.message.delta.partial_json;
              }
            }
            
            if (deltaText) {
              accumulatedContent += deltaText;
            }
          }

          if (messageType === 'message_stop') {
            try {
              const jsonData = JSON.parse(accumulatedContent.trim());
              
              // Validate complex structure
              expect(jsonData).toHaveProperty('company');
              expect(jsonData.company).toHaveProperty('name');
              expect(jsonData.company).toHaveProperty('employees');
              expect(Array.isArray(jsonData.company.employees)).toBe(true);
              expect(jsonData.company.employees.length).toBeGreaterThan(0);
              
              // Validate employee objects
              for (const employee of jsonData.company.employees) {
                expect(employee).toHaveProperty('name');
                expect(employee).toHaveProperty('department');
                expect(employee).toHaveProperty('salary');
                expect(typeof employee.salary).toBe('number');
              }

              expect(jsonData).toHaveProperty('metadata');
              expect(jsonData.metadata).toHaveProperty('created_at');
              expect(jsonData.metadata).toHaveProperty('version');

              console.log('✅ Complex JSON structure validated successfully');
              clearTimeout(timeout);
              ws.close();
              resolve();
            } catch (error) {
              clearTimeout(timeout);
              ws.close();
              const errorMessage = error instanceof Error ? error.message : String(error);
              reject(new Error(`Complex JSON validation failed: ${errorMessage}`));
            }
          }
        });

        ws.on('error', (error: Error) => {
          clearTimeout(timeout);
          reject(error);
        });
      });
    } finally {
      // Clean up complex agent
      await request.delete(`/v1/agents/${complexAgentId}`);
    }
  });

  test('should handle malformed schema gracefully', async ({ request }) => {
    // This test ensures the system handles edge cases properly
    const malformedAgentData = {
      name: `Malformed Schema Test Agent ${Date.now()}`,
      description: 'Agent with empty response format',
      specs: `
model:
  provider: "bedrock/anthropic"
  model_id: "anthropic.claude-3-haiku-20240307-v1:0"
  max_tokens: 1024
  stream: true
  response_format: {}

system: |
  You are a helpful AI assistant.
  Respond normally since no specific schema is provided.

tool_refs: []
`
    };

    const response = await request.post('/v1/agents', {
      data: malformedAgentData
    });

    expect(response.status()).toBe(201);
    const agentBody = await response.json();
    const malformedAgentId = agentBody.id;

    try {
      await new Promise<void>((resolve, reject) => {
        const ws = new WebSocket(WEBSOCKET_URL);
        let responseReceived = false;

        const timeout = setTimeout(() => {
          ws.close();
          if (!responseReceived) {
            reject(new Error('No response received for malformed schema test'));
          }
        }, SHORT_TIMEOUT);

        ws.on('open', () => {
          const message = {
            agent_id: malformedAgentId,
            thread_id: createdThreadId,
            messages: [{
              role: 'user',
              content: [{
                type: 'text',
                text: 'Hello, how are you?'
              }]
            }]
          };
          ws.send(JSON.stringify(message));
        });

        ws.on('message', (data: WebSocket.RawData) => {
          const response = JSON.parse(data.toString());
          const messageType = response.message?.type;
          
          if (messageType === 'content_block_delta') {
            responseReceived = true;
          }

          if (messageType === 'message_stop') {
            console.log('✅ Malformed schema handled gracefully - agent responded normally');
            clearTimeout(timeout);
            ws.close();
            resolve();
          }
        });

        ws.on('error', (error: Error) => {
          clearTimeout(timeout);
          reject(error);
        });
      });
    } finally {
      // Clean up malformed agent
      await request.delete(`/v1/agents/${malformedAgentId}`);
    }
  });
});