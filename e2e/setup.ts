import { test as base, expect } from '@playwright/test';

// Test configuration and utilities
export const test = base.extend({
  // Add any custom fixtures here if needed in the future
});

export const API_ENDPOINTS = {
  FLOWS: '/api/v1/flows',
  FLOW_BY_ID: (id: string) => `/api/v1/flows/${id}`,
  EXECUTE_FLOW: (id: string) => `/api/v1/flows/${id}/execute`,
  FLOW_STATUS: (id: string) => `/api/v1/flows/${id}/status`,
  FLOW_RUN_STATUS: (runId: string) => `/api/v1/flows/runs/${runId}/status`,
} as const;

export const TEST_FLOW_DATA = {
  BASIC: {
    name: 'Test Flow for API Testing',
    description: 'A test flow created via Playwright API tests',
    engine: 'python',
    tags: ['test', 'api', 'automation']
  },
  WITH_SCHEMA: {
    name: 'Test Flow with Schema',
    description: 'A test flow with parameter schema',
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
    additional_info: {
      version: '1.0.0',
      author: 'API Test Suite'
    },
    tags: ['test', 'api', 'schema']
  }
} as const;

export const INVALID_UUID = '00000000-0000-0000-0000-000000000000';
export const MALFORMED_UUID = 'invalid-uuid-format';

export function expectSuccessResponse(responseBody: any) {
  return {
    success: true,
    data: expect.any(Object),
    message: expect.any(String)
  };
}

export function expectErrorResponse(responseBody: any) {
  return {
    success: false,
    error: expect.any(Object)
  };
}