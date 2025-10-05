import { test, expect } from '@playwright/test';

test.describe.serial('Permissions API Extended Tests', () => {
  let createdPermissionId: string;
  let createdPermissionId2: string;

  test.beforeAll(async ({ request }) => {
    // Clean up any existing test permissions before starting
    const response = await request.get('/v1/permissions');
    if (response.ok()) {
      const responseBody = await response.json();
      const permissions = responseBody.permissions || [];
      if (Array.isArray(permissions)) {
        for (const permission of permissions) {
          if (permission.name && permission.name.includes('Extended Test Permission')) {
            await request.delete(`/v1/permissions/${permission.id}`);
          }
        }
      }
    }
  });

  test.afterAll(async ({ request }) => {
    // Clean up created permissions
    if (createdPermissionId) {
      await request.delete(`/v1/permissions/${createdPermissionId}`);
    }
    if (createdPermissionId2) {
      await request.delete(`/v1/permissions/${createdPermissionId2}`);
    }
  });

  test('should create permission with workflow-based content', async ({ request }) => {
    const permissionData = {
      name: 'Extended Test Permission - Workflow Management',
      description: 'Permission for managing workflow execution',
      content: {
        "action": "execute",
        "resource": "workflows",
        "conditions": [
          {
            "type": "user_role",
            "allowed_roles": ["workflow_admin", "workflow_manager"]
          },
          {
            "type": "resource_ownership",
            "require_owner": true
          }
        ],
        "metadata": {
          "category": "workflow_management",
          "priority": "high",
          "created_by": "system"
        }
      }
    };

    const response = await request.post('/v1/permissions', {
      data: permissionData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('id');
    expect(responseBody.name).toBe(permissionData.name);
    expect(responseBody.content.action).toBe('execute');
    expect(responseBody.content.conditions).toHaveLength(2);
    expect(responseBody.content.metadata.category).toBe('workflow_management');

    createdPermissionId = responseBody.id;
  });

  test('should create permission with agent-based content', async ({ request }) => {
    const permissionData = {
      name: 'Extended Test Permission - Agent Control',
      description: 'Permission for controlling AI agents',
      content: {
        "action": "manage",
        "resource": "agents",
        "scope": {
          "agent_types": ["claude-3-sonnet", "claude-3-opus"],
          "max_tokens": 8000,
          "allowed_tools": ["calculator", "weather", "search"]
        },
        "restrictions": {
          "daily_usage_limit": 100,
          "concurrent_sessions": 5,
          "forbidden_actions": ["delete_agent", "modify_system_prompt"]
        },
        "audit": {
          "log_all_interactions": true,
          "retention_days": 30,
          "anonymize_data": false
        }
      }
    };

    const response = await request.post('/v1/permissions', {
      data: permissionData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('id');
    expect(responseBody.name).toBe(permissionData.name);
    expect(responseBody.content.scope.agent_types).toContain('claude-3-sonnet');
    expect(responseBody.content.restrictions.daily_usage_limit).toBe(100);
    expect(responseBody.content.audit.log_all_interactions).toBe(true);

    createdPermissionId2 = responseBody.id;
  });

  test('should update permission content with complex nested structure', async ({ request }) => {
    expect(createdPermissionId).toBeTruthy();

    const updateData = {
      content: {
        "action": "execute",
        "resource": "workflows",
        "conditions": [
          {
            "type": "user_role",
            "allowed_roles": ["workflow_admin", "workflow_manager", "workflow_operator"]
          },
          {
            "type": "resource_ownership",
            "require_owner": true
          },
          {
            "type": "time_based",
            "allowed_hours": {
              "start": "09:00",
              "end": "17:00",
              "timezone": "UTC"
            },
            "allowed_days": ["monday", "tuesday", "wednesday", "thursday", "friday"]
          }
        ],
        "metadata": {
          "category": "workflow_management",
          "priority": "high",
          "created_by": "system",
          "updated_by": "admin",
          "version": "2.0"
        }
      }
    };

    const response = await request.put(`/v1/permissions/${createdPermissionId}`, {
      data: updateData
    });

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.content.conditions).toHaveLength(3);
    expect(responseBody.content.conditions[2].type).toBe('time_based');
    expect(responseBody.content.conditions[2].allowed_hours.start).toBe('09:00');
    expect(responseBody.content.metadata.version).toBe('2.0');
  });

  test('should create permission with data access control', async ({ request }) => {
    const permissionData = {
      name: 'Extended Test Permission - Data Access',
      description: 'Permission for data access and manipulation',
      content: {
        "action": "read_write",
        "resource": "data",
        "data_classification": {
          "allowed_levels": ["public", "internal"],
          "forbidden_levels": ["confidential", "restricted"]
        },
        "data_types": {
          "allowed": ["logs", "metrics", "configurations"],
          "forbidden": ["user_credentials", "api_keys", "personal_data"]
        },
        "operations": {
          "read": {
            "allowed": true,
            "conditions": ["authenticated", "authorized"]
          },
          "write": {
            "allowed": true,
            "conditions": ["authenticated", "authorized", "owner_or_admin"],
            "require_approval": false
          },
          "delete": {
            "allowed": false,
            "reason": "data_retention_policy"
          }
        },
        "compliance": {
          "gdpr_compliant": true,
          "retention_period_days": 90,
          "audit_required": true
        }
      }
    };

    const response = await request.post('/v1/permissions', {
      data: permissionData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody.content.data_classification.allowed_levels).toContain('public');
    expect(responseBody.content.operations.delete.allowed).toBe(false);
    expect(responseBody.content.compliance.gdpr_compliant).toBe(true);

    // Clean up this permission immediately since it's just for testing
    await request.delete(`/v1/permissions/${responseBody.id}`);
  });

  test('should handle permission content with arrays and nested objects', async ({ request }) => {
    const permissionData = {
      name: 'Extended Test Permission - Complex Structure',
      description: 'Testing complex nested permission structures',
      content: {
        "actions": ["create", "read", "update"],
        "resources": [
          {
            "type": "workflow",
            "id_pattern": "wf-*",
            "properties": {
              "status": ["active", "paused"],
              "priority": {
                "min": 1,
                "max": 10
              }
            }
          },
          {
            "type": "agent",
            "id_pattern": "agent-*",
            "properties": {
              "model_types": ["global.anthropic.claude-sonnet-4-5-20250929-v1:0", "claude-3-haiku"],
              "capabilities": {
                "tools": ["calculator", "search"],
                "max_context": 200000
              }
            }
          }
        ],
        "conditions": {
          "user_attributes": {
            "department": ["engineering", "ai_research"],
            "clearance_level": {
              "min": 2,
              "max": 5
            }
          },
          "environmental": {
            "ip_whitelist": ["192.168.1.0/24", "10.0.0.0/8"],
            "user_agent_patterns": ["*internal*", "*trusted*"]
          }
        },
        "effects": {
          "allow": {
            "logging": {
              "level": "info",
              "include_payload": false
            },
            "monitoring": {
              "track_usage": true,
              "alert_on_anomaly": true
            }
          },
          "deny": {
            "reason_required": true,
            "notify_admin": true
          }
        }
      }
    };

    const response = await request.post('/v1/permissions', {
      data: permissionData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody.content.actions).toContain('create');
    expect(responseBody.content.resources).toHaveLength(2);
    expect(responseBody.content.resources[0].type).toBe('workflow');
    expect(responseBody.content.conditions.user_attributes.department).toContain('engineering');
    expect(responseBody.content.effects.allow.monitoring.track_usage).toBe(true);

    // Clean up this permission immediately
    await request.delete(`/v1/permissions/${responseBody.id}`);
  });

  test('should validate permission content consistency after updates', async ({ request }) => {
    expect(createdPermissionId2).toBeTruthy();

    // First, get the current permission
    const getResponse = await request.get(`/v1/permissions/${createdPermissionId2}`);
    expect(getResponse.status()).toBe(200);
    const originalPermission = await getResponse.json();

    // Update only the description
    const updateData = {
      description: 'Updated description for agent control permission'
    };

    const updateResponse = await request.put(`/v1/permissions/${createdPermissionId2}`, {
      data: updateData
    });

    expect(updateResponse.status()).toBe(200);
    
    const updatedPermission = await updateResponse.json();
    
    // Verify description was updated
    expect(updatedPermission.description).toBe(updateData.description);
    
    // Verify content remained unchanged
    expect(updatedPermission.content).toEqual(originalPermission.content);
    expect(updatedPermission.name).toBe(originalPermission.name);
  });

  test('should create permission with minimal valid JSON content', async ({ request }) => {
    const permissionData = {
      name: 'Extended Test Permission - Minimal',
      content: {
        "action": "test"
      }
    };

    const response = await request.post('/v1/permissions', {
      data: permissionData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody.content.action).toBe('test');
    expect(responseBody.description).toBeNull();

    // Clean up this permission
    await request.delete(`/v1/permissions/${responseBody.id}`);
  });

  test('should handle permission content with boolean and numeric values', async ({ request }) => {
    const permissionData = {
      name: 'Extended Test Permission - Data Types',
      description: 'Testing various data types in permission content',
      content: {
        "enabled": true,
        "disabled_feature": false,
        "max_count": 100,
        "min_count": 0,
        "rate_limit": 10.5,
        "timeout_seconds": 30,
        "config": {
          "auto_retry": true,
          "max_retries": 3,
          "backoff_multiplier": 1.5
        },
        "features": [
          {
            "name": "feature_a",
            "enabled": true,
            "weight": 0.8
          },
          {
            "name": "feature_b",
            "enabled": false,
            "weight": 0.2
          }
        ]
      }
    };

    const response = await request.post('/v1/permissions', {
      data: permissionData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody.content.enabled).toBe(true);
    expect(responseBody.content.max_count).toBe(100);
    expect(responseBody.content.rate_limit).toBe(10.5);
    expect(responseBody.content.config.auto_retry).toBe(true);
    expect(responseBody.content.features[0].weight).toBe(0.8);
    expect(responseBody.content.features[1].enabled).toBe(false);

    // Clean up this permission
    await request.delete(`/v1/permissions/${responseBody.id}`);
  });

  test('should list permissions and verify pagination metadata', async ({ request }) => {
    const response = await request.get('/v1/permissions');

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('permissions');
    expect(Array.isArray(responseBody.permissions)).toBe(true);
    
    // Should have at least our created permissions
    expect(responseBody.permissions.length).toBeGreaterThanOrEqual(2);
    
    // Check if pagination metadata exists (optional based on API implementation)
    if (responseBody.meta) {
      expect(responseBody.meta).toHaveProperty('total');
      expect(typeof responseBody.meta.total).toBe('number');
    }
    
    // Verify our created permissions are in the list
    const permissionNames = responseBody.permissions.map((p: any) => p.name);
    expect(permissionNames).toContain('Extended Test Permission - Workflow Management');
    expect(permissionNames).toContain('Extended Test Permission - Agent Control');
  });

  test('should handle concurrent permission operations', async ({ request }) => {
    // Create multiple permissions concurrently
    const permissionPromises = [];
    
    for (let i = 1; i <= 3; i++) {
      const permissionData = {
        name: `Extended Test Permission - Concurrent ${i}`,
        description: `Concurrent test permission ${i}`,
        content: {
          "action": `test_action_${i}`,
          "resource": `test_resource_${i}`,
          "concurrent_id": i
        }
      };
      
      permissionPromises.push(request.post('/v1/permissions', { data: permissionData }));
    }

    const responses = await Promise.all(permissionPromises);
    
    // Verify all permissions were created successfully
    const createdIds = [];
    for (const response of responses) {
      expect(response.status()).toBe(201);
      const body = await response.json();
      createdIds.push(body.id);
    }

    // Verify we can retrieve all created permissions
    for (const id of createdIds) {
      const getResponse = await request.get(`/v1/permissions/${id}`);
      expect(getResponse.status()).toBe(200);
    }

    // Clean up all created permissions concurrently
    const deletePromises = createdIds.map(id => 
      request.delete(`/v1/permissions/${id}`)
    );
    
    const deleteResponses = await Promise.all(deletePromises);
    
    // Verify all permissions were deleted successfully
    for (const response of deleteResponses) {
      expect(response.status()).toBe(204);
    }
  });
});
