import { test, expect } from '@playwright/test';
import WebSocket from 'ws';
// @ts-ignore - NATS types not available
import { connect, NatsConnection } from 'nats';

// Test configuration constants
const WEBSOCKET_URL = 'ws://localhost:8080/v1/ws';
const NATS_URL = 'nats://localhost:4222';
const DEFAULT_TIMEOUT = 15000;
const SHORT_TIMEOUT = 5000;

test.describe.serial('NATS Tools Websocket E2E Integration', () => {
    let createdAgentId: string;
    let createdThreadId: string;
    let createdToolOneId: string;
    let createdToolTwoId: string;
    let natsConnection: NatsConnection;
    let testUserId: string = '550e8400-c95b-4444-6666-446655440000'; // Admin user UUID

    test.beforeAll(async ({ request }) => {
        // Connect to NATS
        natsConnection = await connect({ servers: NATS_URL });
        console.log('Connected to NATS server');

        // Clean up any existing test data
        await cleanupTestData(request);

        // Create the first test tool for testing
        const toolOneData = {
            name: `test_tool_one`,
            description: 'A test standalone tool created via Playwright API tests',
            config: {
                type: 'standalone',
                url: 'http://localhost:8080/v1/mock/tool',
                params: {
                    type: 'object',
                    properties: {
                        input: {
                            type: 'string',
                            description: 'Mock input for the tool'
                        }
                    },
                    required: ['input']
                }
            }
        }

        const responseToolOne = await request.post('/v1/tools', {
            data: toolOneData
        });
        
        if (responseToolOne.status() !== 201) {
            console.error('First tool creation failed with status:', responseToolOne.status());
            const errorBody = await responseToolOne.json().catch(() => responseToolOne.text());
            console.error('Error response:', errorBody);
        }
        expect(responseToolOne.status()).toBe(201);
        
        const responseToolOneBody = await responseToolOne.json();
        createdToolOneId = responseToolOneBody.id

        // Create the second test tool for testing
        const toolTwoData = {
            name: `test_tool_two`,
            description: 'A test second standalone tool created via Playwright API test for parallel execution testing. This tool exepected to return failed.`',
            config: {
                type: 'standalone',
                url: 'http://undefined_url:0000',
                params: {
                    type: 'object',
                    properties: {
                        input: {
                            type: 'string',
                            description: 'Mock input for the tool'
                        }
                    },
                    required: ['input']
                }
            }
        }

        const responseToolTwo = await request.post('/v1/tools', {
            data: toolTwoData
        });

        if (responseToolTwo.status() !== 201) {
            console.error('Second tool creation failed with status:', responseToolTwo.status());
            const errorBody = await responseToolTwo.json().catch(() => responseToolTwo.text());
            console.error('Error response:', errorBody);
        }
        expect(responseToolTwo.status()).toBe(201);

        const responseToolTwoBody = await responseToolTwo.json();
        createdToolTwoId = responseToolTwoBody.id

        // Create a test agent for testing
        const agentData = {
            name: 'Test NATS WebSocket Agent',
            description: 'A test agent for NATS WebSocket e2e testing',
            specs: `
model:
  provider: "bedrock/anthropic"
  model_id: "apac.anthropic.claude-sonnet-4-20250514-v1:0"
  max_tokens: 2048
  stream: true
  thinking:
    enabled: true
    budget_token: 1024

system: |
  You are a helpful AI assistant for tool e2e testing.
  You MUST use the available tools when asked to demonstrate them.
  Keep your final response short after using tools.

tool_refs:
  - ${createdToolOneId}
  - ${createdToolTwoId}
`
        };

        const agentResponse = await request.post('/v1/agents', {
            data: agentData
        });

        expect(agentResponse.status()).toBe(201);
        const agentBody = await agentResponse.json();
        createdAgentId = agentBody.id;
        console.log(`Created test agent with ID: ${createdAgentId}`);

        // Create a test thread
        const threadData = {
            title: 'Test NATS WebSocket Thread',
            user_id: testUserId
        };

        const threadResponse = await request.post('/v1/threads', {
            data: threadData
        });

        expect(threadResponse.status()).toBe(201);
        const threadBody = await threadResponse.json();
        createdThreadId = threadBody.id;
        console.log(`Created test thread with ID: ${createdThreadId}`);
    });

    test.afterAll(async ({ request }) => {
        // Clean up created resources
        if (createdThreadId) {
            await request.delete(`/v1/threads/${createdThreadId}`);
        }
        if (createdAgentId) {
            await request.delete(`/v1/agents/${createdAgentId}`);
        }
        if (createdToolOneId) {
            await request.delete(`/v1/tools/${createdToolOneId}`);
        }
        if (createdToolTwoId) {
            await request.delete(`/v1/tools/${createdToolTwoId}`);
        }

        // Close NATS connection
        if (natsConnection) {
            await natsConnection.close();
            console.log('NATS connection closed');
        }
    });

    // Helper function to clean up existing test data
    async function cleanupTestData(request: any) {
        // Clean up test agents
        const agentsResponse = await request.get('/v1/agents');
        if (agentsResponse.ok()) {
            const agentsBody = await agentsResponse.json();
            const agents = agentsBody.agents || [];
            if (Array.isArray(agents)) {
                for (const agent of agents) {
                    if (agent.name && agent.name.includes('Test NATS WebSocket Agent')) {
                        await request.delete(`/v1/agents/${agent.id}`);
                    }
                }
            }
        }

        // Clean up test threads
        const threadsResponse = await request.get('/v1/threads');
        if (threadsResponse.ok()) {
            const threadsBody = await threadsResponse.json();
            const threads = threadsBody.threads || [];
            if (Array.isArray(threads)) {
                for (const thread of threads) {
                    if (thread.title && thread.title.includes('Test NATS WebSocket Thread')) {
                        await request.delete(`/v1/threads/${thread.id}`);
                    }
                }
            }
        }

        // Clean up test tools
        const toolsResponse = await request.get('/v1/tools');
        if (toolsResponse.ok()) {
            const tools = await toolsResponse.json();
            if (Array.isArray(tools)) {
                for (const tool of tools) {
                    if (tool.name && tool.name.includes('test_tool_')) {
                        await request.delete(`/v1/tools/${tool.id}`);
                    }
                }
            }
        }
    }

    // Helper function to create WebSocket connection
    async function createWebSocketConnection(url: string = WEBSOCKET_URL): Promise<WebSocket> {
        return new Promise<WebSocket>((resolve, reject) => {
            const ws = new WebSocket(url);
            const connectionTimeout = setTimeout(() => {
                ws.terminate();
                reject(new Error('WebSocket connection timeout'));
            }, DEFAULT_TIMEOUT);

            ws.on('open', () => {
                clearTimeout(connectionTimeout);
                resolve(ws);
            });

            ws.on('error', (error) => {
                clearTimeout(connectionTimeout);
                reject(error);
            });
        });
    }

    // Helper function to safely close WebSocket
    async function safeCloseConnection(ws: WebSocket): Promise<void> {
        return new Promise((resolve) => {
            if (ws.readyState === WebSocket.CLOSED) {
                resolve();
                return;
            }

            const closeTimeout = setTimeout(() => {
                ws.terminate();
                resolve();
            }, SHORT_TIMEOUT);

            ws.once('close', () => {
                clearTimeout(closeTimeout);
                resolve();
            });

            if (ws.readyState === WebSocket.OPEN) {
                ws.close(1000, 'Test completed');
            } else {
                ws.terminate();
            }
        });
    }

    test('should complete full NATS WebSocket tool flow with multiple message cycles: WebSocket → Task → Agent → Tool Call → Tool Response → Agent → Final Response → WebSocket', async () => {
        expect(createdAgentId).toBeTruthy();
        expect(createdThreadId).toBeTruthy();

        // WebSocket response tracking
        const receivedWebSocketMessages: string[] = [];
        const eventTypes: string[] = [];
        let streamingStarted = false;
        let streamingCompleted = false;
        let contentReceived = false;

        // Order validation tracking
        let orderValidationErrors: string[] = [];
        let taskStartReceived = false;
        let messageStartCount = 0;
        let messageStopCount = 0;

        // Create WebSocket connection and monitor streaming responses
        const ws = await createWebSocketConnection();

        try {
            // Set up WebSocket message handler to collect streaming responses
            ws.on('message', (data: WebSocket.Data) => {
                const message = data.toString();
                receivedWebSocketMessages.push(message);
                console.log(`⚙️ WebSocket received streaming message: ${message.substring(0, 100)}...`);

                try {
                    const parsedMessage = JSON.parse(message);
                    
                    // Handle error messages
                    if (parsedMessage.error) {
                        console.log(`❌ Error received: ${parsedMessage.error}`);
                        return;
                    }
                    
                    // Skip messages without message property
                    if (!parsedMessage.message || !parsedMessage.message.type) {
                        console.log(`⚠️ Skipping message without type:`, parsedMessage);
                        return;
                    }
                    
                    const messageType = parsedMessage.message.type;
                    eventTypes.push(messageType);

                    // Validate message order - handle multiple content block sequences
                    const validateMessageOrder = (type: string, events: string[]) => {
                        const lastEvent = events[events.length - 2]; // Previous event

                        switch (type) {
                            case 'task_start':
                                // Must be the very first event
                                return events.length === 1;

                            case 'message_start':
                                // Must come after task_start or after message_stop (new stream)
                                return lastEvent === 'task_start' || lastEvent === 'message_stop';

                            case 'content_block_start':
                                // Must come after message_start or content_block_stop
                                return lastEvent === 'message_start' || lastEvent === 'content_block_stop';

                            case 'content_block_delta':
                                // Must come after content_block_start or another content_block_delta
                                return lastEvent === 'content_block_start' || lastEvent === 'content_block_delta';

                            case 'content_block_stop':
                                // Must come after content_block_delta
                                return lastEvent === 'content_block_delta';

                            case 'message_delta':
                                // Must come after content_block_stop (last content block finished)
                                return lastEvent === 'content_block_stop';

                            case 'message_stop':
                                // Must come after message_delta
                                return lastEvent === 'message_delta';

                            case 'task_stop':
                                // Must come after message_stop
                                return lastEvent === 'message_stop';

                            default:
                                return false;
                        }
                    };

                    if (eventTypes.length > 1 && !validateMessageOrder(messageType, eventTypes)) {
                        const lastEvent = eventTypes[eventTypes.length - 2];
                        orderValidationErrors.push(`Invalid order: ${messageType} cannot follow ${lastEvent}`);
                    }

                    // Validate streaming event types and format
                    if (messageType === 'task_start') {
                        taskStartReceived = true;
                        console.log('✓ Task Event: task_start received');
                        expect(parsedMessage.message.task_id).toBeDefined();
                    } else if (messageType === 'message_start') {
                        messageStartCount++;
                        streamingStarted = true;
                        console.log(`✓ Stream Event: message_start received (count: ${messageStartCount})`);
                        expect(parsedMessage.message.message).toBeDefined();
                        expect(parsedMessage.message.provider).toBeDefined();
                    } else if (messageType === 'content_block_start') {
                        console.log('✓ Stream Event: content_block_start received');
                        expect(parsedMessage.message.content_block).toBeDefined();
                    } else if (messageType === 'content_block_delta') {
                        contentReceived = true;
                        console.log('✓ Stream Event: content_block_delta received with text:', parsedMessage.message.delta?.text || 'no text');
                        expect(parsedMessage.message.delta).toBeDefined();
                    } else if (messageType === 'content_block_stop') {
                        console.log('✓ Stream Event: content_block_stop received');
                    } else if (messageType === 'message_delta') {
                        console.log('✓ Stream Event: message_delta received');
                        expect(parsedMessage.message.delta).toBeDefined();
                    } else if (messageType === 'message_stop') {
                        messageStopCount++;
                        streamingCompleted = true;
                        console.log(`✓ Stream Event: message_stop received (count: ${messageStopCount})`);
                    } else if (messageType === 'task_stop') {
                        console.log('✓ Task Event: task_stop received');
                    }

                    // Validate all streaming messages have required fields
                    expect(parsedMessage.message.type).toBeDefined();

                    // Only validate provider for AI streaming events, not task lifecycle events
                    if (messageType !== 'task_start' && messageType !== 'task_stop') {
                        expect(parsedMessage.message.provider).toBe('anthropic');
                    }

                } catch (error) {
                    console.error(error);
                }
            });

            // Step 5: Send test message via WebSocket to initiate the full flow
            const testMessage = {
                agent_id: createdAgentId,
                thread_id: createdThreadId,
                messages: [
                    {
                        role: "user",
                        content: [{
                            type: "text",
                            text: "Demonstrate the test_tool_one tool only for me"
                        }]
                    }
                ]
            };

            console.log('🚀 Starting full NATS WebSocket tool flow test...');
            console.log('Sending WebSocket message to initiate: WebSocket → Task → Agent → Tool Execution → Final Response → WebSocket');
            ws.send(JSON.stringify(testMessage));

            // Wait for the complete flow to process (longer timeout for tool execution)
            await new Promise(resolve => setTimeout(resolve, 20000));

            // Verify Step 1: Complete WebSocket streaming flow
            expect(receivedWebSocketMessages.length).toBeGreaterThan(0);
            console.log('✅ Step 1 Verified: WebSocket client received streaming responses');
            console.log(`Total WebSocket messages received: ${receivedWebSocketMessages.length}`);

            // Verify task_start event is received first
            expect(taskStartReceived).toBe(true);
            console.log('✅ Task Event: task_start event verified as first event');
            expect(eventTypes[0]).toBe('task_start');
            console.log('✅ Event Order: task_start is the first event in sequence');
            expect(eventTypes[eventTypes.length - 1]).toBe('task_stop');
            console.log('✅ Event Order: task_stop is the last event in sequence');

            // Verify streaming event sequence and format
            expect(eventTypes).toContain('message_start');
            console.log('✅ Streaming Format: message_start event verified');

            // Verify message order is correct
            expect(orderValidationErrors).toEqual([]);
            if (orderValidationErrors.length === 0) {
                console.log('✅ Message Order: All streaming events received in correct order');
            } else {
                console.log('❌ Message Order Errors:', orderValidationErrors);
            }
            console.log('📋 Event Sequence:', eventTypes.join(' → '));

            if (taskStartReceived) {
                console.log('✅ Task Flow: Task started correctly');
            }

            if (streamingStarted) {
                console.log('✅ Streaming Flow: Streaming started correctly');
            }

            if (contentReceived) {
                console.log('✅ Streaming Content: Content delta received');
            }

            if (streamingCompleted) {
                console.log('✅ Streaming Flow: Streaming completed correctly');
            }

            // Verify tool execution - should have multiple message cycles (tool call + response)
            expect(messageStartCount).toBeGreaterThanOrEqual(2);
            expect(messageStopCount).toBeGreaterThanOrEqual(2);
            console.log(`✅ Tool Execution: Detected ${messageStartCount} message_start and ${messageStopCount} message_stop events`);
            
            // Verify message start/stop balance (should be close to equal)
            const messageDelta = Math.abs(messageStartCount - messageStopCount);
            expect(messageDelta).toBeLessThanOrEqual(1); // Should be balanced or off by 1
            console.log(`✅ Message Balance: Start/Stop delta is ${messageDelta} (expected ≤ 1)`);

            // Verify unique event types received
            const uniqueEventTypes = [...new Set(eventTypes)];
            console.log('✅ Event Types Received:', uniqueEventTypes);
            expect(uniqueEventTypes.length).toBeGreaterThan(1);

            // Log complete flow success
            console.log('\n🎉 COMPLETE TOOL FLOW VERIFICATION SUCCESS:');
            console.log(`   📊 Total streaming messages: ${receivedWebSocketMessages.length}`);
            console.log(`   📋 Event types: ${uniqueEventTypes.join(', ')}`);
            console.log(`   🔧 Tool execution cycles: ${messageStartCount} starts, ${messageStopCount} stops`);
            console.log(`   ⚖️ Message balance: ${messageDelta} delta (balanced execution)`);

        } finally {
            await safeCloseConnection(ws);
        }
    });

    test('should handle parallel tool calling (without batch_tool)', async () => {
        expect(createdAgentId).toBeTruthy();
        expect(createdThreadId).toBeTruthy();

        // WebSocket response tracking
        const receivedWebSocketMessages: string[] = [];
        const eventTypes: string[] = [];
        let streamingStarted = false;
        let streamingCompleted = false;
        let contentReceived = false;

        // Order validation tracking
        let orderValidationErrors: string[] = [];
        let taskStartReceived = false;
        let messageStartCount = 0;
        let messageStopCount = 0;

        // Create WebSocket connection and monitor streaming responses
        const ws = await createWebSocketConnection();

        try {
            // Set up WebSocket message handler to collect streaming responses
            ws.on('message', (data: WebSocket.Data) => {
                const message = data.toString();
                receivedWebSocketMessages.push(message);
                console.log(`⚙️ WebSocket received streaming message: ${message.substring(0, 100)}...`);

                try {
                    const parsedMessage = JSON.parse(message);
                    
                    // Handle error messages
                    if (parsedMessage.error) {
                        console.log(`❌ Error received: ${parsedMessage.error}`);
                        return;
                    }
                    
                    // Skip messages without message property
                    if (!parsedMessage.message || !parsedMessage.message.type) {
                        console.log(`⚠️ Skipping message without type:`, parsedMessage);
                        return;
                    }
                    
                    const messageType = parsedMessage.message.type;
                    eventTypes.push(messageType);

                    // Validate message order - handle multiple content block sequences
                    const validateMessageOrder = (type: string, events: string[]) => {
                        const lastEvent = events[events.length - 2]; // Previous event

                        switch (type) {
                            case 'task_start':
                                // Must be the very first event
                                return events.length === 1;

                            case 'message_start':
                                // Must come after task_start or after message_stop (new stream)
                                return lastEvent === 'task_start' || lastEvent === 'message_stop';

                            case 'content_block_start':
                                // Must come after message_start or content_block_stop
                                return lastEvent === 'message_start' || lastEvent === 'content_block_stop';

                            case 'content_block_delta':
                                // Must come after content_block_start or another content_block_delta
                                return lastEvent === 'content_block_start' || lastEvent === 'content_block_delta';

                            case 'content_block_stop':
                                // Must come after content_block_delta
                                return lastEvent === 'content_block_delta';

                            case 'message_delta':
                                // Must come after content_block_stop (last content block finished)
                                return lastEvent === 'content_block_stop';

                            case 'message_stop':
                                // Must come after message_delta
                                return lastEvent === 'message_delta';

                            case 'task_stop':
                                // Must come after message_stop
                                return lastEvent === 'message_stop';

                            default:
                                return false;
                        }
                    };

                    if (eventTypes.length > 1 && !validateMessageOrder(messageType, eventTypes)) {
                        const lastEvent = eventTypes[eventTypes.length - 2];
                        orderValidationErrors.push(`Invalid order: ${messageType} cannot follow ${lastEvent}`);
                    }

                    // Validate streaming event types and format
                    if (messageType === 'task_start') {
                        taskStartReceived = true;
                        console.log('✓ Task Event: task_start received');
                        expect(parsedMessage.message.task_id).toBeDefined();
                    } else if (messageType === 'message_start') {
                        messageStartCount++;
                        streamingStarted = true;
                        console.log(`✓ Stream Event: message_start received (count: ${messageStartCount})`);
                        expect(parsedMessage.message.message).toBeDefined();
                        expect(parsedMessage.message.provider).toBeDefined();
                    } else if (messageType === 'content_block_start') {
                        console.log('✓ Stream Event: content_block_start received');
                        expect(parsedMessage.message.content_block).toBeDefined();
                    } else if (messageType === 'content_block_delta') {
                        contentReceived = true;
                        console.log('✓ Stream Event: content_block_delta received with text:', parsedMessage.message.delta?.text || 'no text');
                        expect(parsedMessage.message.delta).toBeDefined();
                    } else if (messageType === 'content_block_stop') {
                        console.log('✓ Stream Event: content_block_stop received');
                    } else if (messageType === 'message_delta') {
                        console.log('✓ Stream Event: message_delta received');
                        expect(parsedMessage.message.delta).toBeDefined();
                    } else if (messageType === 'message_stop') {
                        messageStopCount++;
                        streamingCompleted = true;
                        console.log(`✓ Stream Event: message_stop received (count: ${messageStopCount})`);
                    } else if (messageType === 'task_stop') {
                        console.log('✓ Task Event: task_stop received');
                    }

                    // Validate all streaming messages have required fields
                    expect(parsedMessage.message.type).toBeDefined();

                    // Only validate provider for AI streaming events, not task lifecycle events
                    if (messageType !== 'task_start' && messageType !== 'task_stop') {
                        expect(parsedMessage.message.provider).toBe('anthropic');
                    }

                } catch (error) {
                    console.error(error);
                }
            });

            // Step 5: Send test message via WebSocket to initiate the full flow
            const testMessage = {
                agent_id: createdAgentId,
                thread_id: createdThreadId,
                messages: [
                    {
                        role: "user",
                        content: [{
                            type: "text",
                            text: "Demonstrate the parallel execution for all current tools"
                        }]
                    }
                ]
            };

            console.log('🚀 Starting full NATS WebSocket tool flow test...');
            console.log('Sending WebSocket message to initiate: WebSocket → Task → Agent → Tool Execution → Final Response → WebSocket');
            ws.send(JSON.stringify(testMessage));

            // Wait for the complete flow to process (longer timeout for tool execution)
            await new Promise(resolve => setTimeout(resolve, 20000));

            // Verify Step 1: Complete WebSocket streaming flow
            expect(receivedWebSocketMessages.length).toBeGreaterThan(0);
            console.log('✅ Step 1 Verified: WebSocket client received streaming responses');
            console.log(`Total WebSocket messages received: ${receivedWebSocketMessages.length}`);

            // Verify task_start event is received first
            expect(taskStartReceived).toBe(true);
            console.log('✅ Task Event: task_start event verified as first event');
            expect(eventTypes[0]).toBe('task_start');
            console.log('✅ Event Order: task_start is the first event in sequence');
            expect(eventTypes[eventTypes.length - 1]).toBe('task_stop');
            console.log('✅ Event Order: task_stop is the last event in sequence');

            // Verify streaming event sequence and format
            expect(eventTypes).toContain('message_start');
            console.log('✅ Streaming Format: message_start event verified');

            // Verify message order is correct
            expect(orderValidationErrors).toEqual([]);
            if (orderValidationErrors.length === 0) {
                console.log('✅ Message Order: All streaming events received in correct order');
            } else {
                console.log('❌ Message Order Errors:', orderValidationErrors);
            }
            console.log('📋 Event Sequence:', eventTypes.join(' → '));

            if (taskStartReceived) {
                console.log('✅ Task Flow: Task started correctly');
            }

            if (streamingStarted) {
                console.log('✅ Streaming Flow: Streaming started correctly');
            }

            if (contentReceived) {
                console.log('✅ Streaming Content: Content delta received');
            }

            if (streamingCompleted) {
                console.log('✅ Streaming Flow: Streaming completed correctly');
            }

            // Verify tool execution - should have multiple message cycles (tool call + response)
            expect(messageStartCount).toBeGreaterThanOrEqual(2);
            expect(messageStopCount).toBeGreaterThanOrEqual(2);
            console.log(`✅ Tool Execution: Detected ${messageStartCount} message_start and ${messageStopCount} message_stop events`);
            
            // Verify message start/stop balance (should be close to equal)
            const messageDelta = Math.abs(messageStartCount - messageStopCount);
            expect(messageDelta).toBeLessThanOrEqual(1); // Should be balanced or off by 1
            console.log(`✅ Message Balance: Start/Stop delta is ${messageDelta} (expected ≤ 1)`);

            // Verify unique event types received
            const uniqueEventTypes = [...new Set(eventTypes)];
            console.log('✅ Event Types Received:', uniqueEventTypes);
            expect(uniqueEventTypes.length).toBeGreaterThan(1);

            // Log complete flow success
            console.log('\n🎉 COMPLETE TOOL FLOW VERIFICATION SUCCESS:');
            console.log(`   📊 Total streaming messages: ${receivedWebSocketMessages.length}`);
            console.log(`   📋 Event types: ${uniqueEventTypes.join(', ')}`);
            console.log(`   🔧 Tool execution cycles: ${messageStartCount} starts, ${messageStopCount} stops`);
            console.log(`   ⚖️ Message balance: ${messageDelta} delta (balanced execution)`);

        } finally {
            await safeCloseConnection(ws);
        }
    });
});