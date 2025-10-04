import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './',
  outputDir: './test-artifacts',
  timeout: 2 * 60 * 1000, // 2 minutes for SSE tests
  expect: {
    timeout: 5000
  },
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: [['list'], ['html', { outputFolder: './test-results/playwright-report' }]],
  use: {
    baseURL: process.env.BASE_URL || 'http://localhost:8080',
    extraHTTPHeaders: {
      'Content-Type': 'application/json',
    },
  },
  projects: [
    {
      name: 'flows-api',
      testMatch: 'flows.spec.ts',
    },
    {
      name: 'tools-api',
      testMatch: 'tools.spec.ts',
    },
    {
      name: 'tasks-api',
      testMatch: 'tasks.spec.ts',
    },
    {
      name: 'agents-api',
      testMatch: 'agents.spec.ts',
    },
    {
      name: 'threads-api',
      testMatch: 'threads.spec.ts',
    },
    {
      name: 'messages-api',
      testMatch: 'messages.spec.ts',
    },
    {
      name: 'roles-api',
      testMatch: 'roles.spec.ts',
    },
    {
      name: 'permissions-api',
      testMatch: 'permissions.spec.ts',
    },
    {
      name: 'users-api',
      testMatch: 'users.spec.ts',
    },
    {
      name: 'websocket-api',
      testMatch: 'websocket.spec.ts',
    },
    {
      name: 'nats-websocket-e2e',
      testMatch: 'nats-websocket-e2e.spec.ts',
    },
    {
      name: 'nats-tool-websocket-e2e',
      testMatch: 'nats-tools-websocket-e2e.spec.ts',
      // dependencies: ['nats-websocket-e2e']
    },
    {
      name: 'nats-internal-tool-websocket-e2e',
      testMatch: 'nats-internal-tool-websocket-e2e.spec.ts',
    },
    {
      name: 'nats-multi-agents-websocket-e2e',
      testMatch: 'nats-multi-agents-websocket-e2e.spec.ts',
    },
    {
      name: 'structure-output-websocket-e2e',
      testMatch: 'structured-output.spec.ts',
    }
  ],
});