import { test, expect } from '@playwright/test';

test.describe.serial('Roles API', () => {
  let createdRoleId: string;

  test.beforeAll(async ({ request }) => {
    // Clean up any existing test roles before starting
    const response = await request.get('/v1/roles');
    if (response.ok()) {
      const responseBody = await response.json();
      const roles = responseBody.roles || [];
      if (Array.isArray(roles)) {
        for (const role of roles) {
          if (role.name && role.name.startsWith('Test Role')) {
            await request.delete(`/v1/roles/${role.id}`);
          }
        }
      }
    }
  });

  test.afterAll(async ({ request }) => {
    // Clean up created role
    if (createdRoleId) {
      await request.delete(`/v1/roles/${createdRoleId}`);
    }
  });

  test('should create a new role', async ({ request }) => {
    const roleData = {
      name: 'Test Role for API Testing',
      description: 'A test role created via Playwright API tests',
      is_system_role: false
    };

    const response = await request.post('/v1/roles', {
      data: roleData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('id');
    expect(responseBody.name).toBe(roleData.name);
    expect(responseBody.description).toBe(roleData.description);
    expect(responseBody.is_system).toBe(roleData.is_system_role);
    expect(responseBody).toHaveProperty('created_at');
    expect(responseBody).toHaveProperty('updated_at');

    createdRoleId = responseBody.id;
  });

  test('should get all roles', async ({ request }) => {
    const response = await request.get('/v1/roles');

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('roles');
    expect(responseBody.roles).toBeDefined();
    expect(Array.isArray(responseBody.roles)).toBe(true);
    expect(responseBody.roles.length).toBeGreaterThanOrEqual(1); // At least our created role
  });

  test('should get role by ID', async ({ request }) => {
    expect(createdRoleId).toBeTruthy();

    const response = await request.get(`/v1/roles/${createdRoleId}`);

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdRoleId);
    expect(responseBody.name).toBe('Test Role for API Testing');
    expect(responseBody.description).toBe('A test role created via Playwright API tests');
    expect(responseBody.is_system).toBe(false);
  });

  test('should return 404 for non-existent role', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789012';
    const response = await request.get(`/v1/roles/${nonExistentId}`);

    expect(response.status()).toBe(404);
    
    const responseBody = await response.json();
    expect(responseBody.resource).toBe("Role");
    expect(responseBody.id).toBe(nonExistentId);
  });

  test('should return 400 for invalid role ID format', async ({ request }) => {
    const invalidId = 'invalid-uuid';
    const response = await request.get(`/v1/roles/${invalidId}`);

    expect(response.status()).toBe(400);
  });

  test('should update an existing role', async ({ request }) => {
    expect(createdRoleId).toBeTruthy();

    const updateData = {
      name: 'Updated Test Role',
      description: 'Updated description for the test role',
      is_system_role: true
    };

    const response = await request.put(`/v1/roles/${createdRoleId}`, {
      data: updateData
    });

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdRoleId);
    expect(responseBody.name).toBe(updateData.name);
    expect(responseBody.description).toBe(updateData.description);
    expect(responseBody.is_system).toBe(updateData.is_system_role);
  });

  test('should return 404 when updating non-existent role', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789abc';
    const updateData = {
      name: 'Non-existent Role',
      description: 'This role does not exist'
    };

    const response = await request.put(`/v1/roles/${nonExistentId}`, {
      data: updateData
    });

    expect(response.status()).toBe(404);
  });

  test('should handle validation errors for create role - missing required fields', async ({ request }) => {
    const invalidRoleData = {
      // Missing required name field
      description: 'Role without name'
    };

    const response = await request.post('/v1/roles', {
      data: invalidRoleData
    });

    expect(response.status()).toBe(400);
  });

  test('should handle validation errors for role name too long', async ({ request }) => {
    const invalidRoleData = {
      name: 'A'.repeat(300), // Exceeds maxLength of 255
      description: 'Role with very long name'
    };

    const response = await request.post('/v1/roles', {
      data: invalidRoleData
    });

    expect(response.status()).toBe(400);
  });

  test('should create role with minimal required fields', async ({ request }) => {
    const minimalRoleData = {
      name: 'Minimal Test Role'
    };

    const response = await request.post('/v1/roles', {
      data: minimalRoleData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody.name).toBe(minimalRoleData.name);
    expect(responseBody).toHaveProperty('id');

    // Clean up the minimal role
    await request.delete(`/v1/roles/${responseBody.id}`);
  });

  test('should update role with partial data', async ({ request }) => {
    expect(createdRoleId).toBeTruthy();

    // Only update the description
    const partialUpdateData = {
      description: 'Partially updated description'
    };

    const response = await request.put(`/v1/roles/${createdRoleId}`, {
      data: partialUpdateData
    });

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdRoleId);
    expect(responseBody.description).toBe(partialUpdateData.description);
    // Name should remain the same from previous update
    expect(responseBody.name).toBe('Updated Test Role');
  });

  test('should delete a role', async ({ request }) => {
    expect(createdRoleId).toBeTruthy();

    const response = await request.delete(`/v1/roles/${createdRoleId}`);

    expect(response.status()).toBe(204);

    // Verify role is deleted by trying to fetch it
    const getResponse = await request.get(`/v1/roles/${createdRoleId}`);
    expect(getResponse.status()).toBe(404);

    // Clear the ID so cleanup doesn't try to delete again
    createdRoleId = '';
  });

  test('should return 404 when deleting non-existent role', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789def';
    const response = await request.delete(`/v1/roles/${nonExistentId}`);

    expect(response.status()).toBe(404);
  });
});

test.describe.serial('Permissions API', () => {
  let createdPermissionId: string;

  test.beforeAll(async ({ request }) => {
    // Clean up any existing test permissions before starting
    const response = await request.get('/v1/permissions');
    if (response.ok()) {
      const responseBody = await response.json();
      const permissions = responseBody.permissions || [];
      if (Array.isArray(permissions)) {
        for (const permission of permissions) {
          if (permission.name && permission.name.startsWith('Test Permission')) {
            await request.delete(`/v1/permissions/${permission.id}`);
          }
        }
      }
    }
  });

  test.afterAll(async ({ request }) => {
    // Clean up created permission
    if (createdPermissionId) {
      await request.delete(`/v1/permissions/${createdPermissionId}`);
    }
  });

  test('should create a new permission', async ({ request }) => {
    const permissionData = {
      name: 'Test Permission for API Testing',
      description: 'A test permission created via Playwright API tests',
      content: {
        "action": "read",
        "resource": "agents",
        "conditions": ["owner_only"]
      }
    };

    const response = await request.post('/v1/permissions', {
      data: permissionData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('id');
    expect(responseBody.name).toBe(permissionData.name);
    expect(responseBody.description).toBe(permissionData.description);
    expect(responseBody.content).toEqual(permissionData.content);
    expect(responseBody).toHaveProperty('created_at');
    expect(responseBody).toHaveProperty('updated_at');

    createdPermissionId = responseBody.id;
  });

  test('should get all permissions', async ({ request }) => {
    const response = await request.get('/v1/permissions');

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('permissions');
    expect(responseBody.permissions).toBeDefined();
    expect(Array.isArray(responseBody.permissions)).toBe(true);
    expect(responseBody.permissions.length).toBeGreaterThanOrEqual(1); // At least our created permission
  });

  test('should get permission by ID', async ({ request }) => {
    expect(createdPermissionId).toBeTruthy();

    const response = await request.get(`/v1/permissions/${createdPermissionId}`);

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdPermissionId);
    expect(responseBody.name).toBe('Test Permission for API Testing');
    expect(responseBody.description).toBe('A test permission created via Playwright API tests');
    expect(responseBody.content.action).toBe('read');
    expect(responseBody.content.resource).toBe('agents');
  });

  test('should return 404 for non-existent permission', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789012';
    const response = await request.get(`/v1/permissions/${nonExistentId}`);

    expect(response.status()).toBe(404);
    
    const responseBody = await response.json();
    expect(responseBody.resource).toBe("Permission");
    expect(responseBody.id).toBe(nonExistentId);
  });

  test('should return 400 for invalid permission ID format', async ({ request }) => {
    const invalidId = 'invalid-uuid';
    const response = await request.get(`/v1/permissions/${invalidId}`);

    expect(response.status()).toBe(400);
  });

  test('should update an existing permission', async ({ request }) => {
    expect(createdPermissionId).toBeTruthy();

    const updateData = {
      name: 'Updated Test Permission',
      description: 'Updated description for the test permission',
      content: {
        "action": "write",
        "resource": "flows",
        "conditions": ["admin_only", "owner_only"]
      }
    };

    const response = await request.put(`/v1/permissions/${createdPermissionId}`, {
      data: updateData
    });

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdPermissionId);
    expect(responseBody.name).toBe(updateData.name);
    expect(responseBody.description).toBe(updateData.description);
    expect(responseBody.content).toEqual(updateData.content);
  });

  test('should return 404 when updating non-existent permission', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789abc';
    const updateData = {
      name: 'Non-existent Permission',
      description: 'This permission does not exist'
    };

    const response = await request.put(`/v1/permissions/${nonExistentId}`, {
      data: updateData
    });

    expect(response.status()).toBe(404);
  });

  test('should handle validation errors for create permission - missing required fields', async ({ request }) => {
    const invalidPermissionData = {
      // Missing required name and content fields
      description: 'Permission without name and content'
    };

    const response = await request.post('/v1/permissions', {
      data: invalidPermissionData
    });

    expect(response.status()).toBe(400);
  });

  test('should handle validation errors for permission name too long', async ({ request }) => {
    const invalidPermissionData = {
      name: 'A'.repeat(300), // Exceeds maxLength of 255
      description: 'Permission with very long name',
      content: { "action": "test" }
    };

    const response = await request.post('/v1/permissions', {
      data: invalidPermissionData
    });

    expect(response.status()).toBe(400);
  });

  test('should create permission with minimal required fields', async ({ request }) => {
    const minimalPermissionData = {
      name: 'Minimal Test Permission',
      content: { "action": "test" }
    };

    const response = await request.post('/v1/permissions', {
      data: minimalPermissionData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody.name).toBe(minimalPermissionData.name);
    expect(responseBody.content).toEqual(minimalPermissionData.content);
    expect(responseBody).toHaveProperty('id');

    // Clean up the minimal permission
    await request.delete(`/v1/permissions/${responseBody.id}`);
  });

  test('should create permission with complex JSON content', async ({ request }) => {
    const complexPermissionData = {
      name: 'Complex Test Permission',
      description: 'Testing complex JSON permission content',
      content: {
        "action": "execute",
        "resource": "workflows",
        "conditions": [
          {
            "type": "time_based",
            "schedule": {
              "start_time": "09:00",
              "end_time": "17:00",
              "timezone": "UTC"
            }
          },
          {
            "type": "role_based",
            "required_roles": ["admin", "workflow_manager"],
            "fallback_permissions": ["read_only"]
          }
        ],
        "metadata": {
          "created_by": "system",
          "version": "1.0",
          "tags": ["workflow", "execution", "restricted"]
        }
      }
    };

    const response = await request.post('/v1/permissions', {
      data: complexPermissionData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody.name).toBe(complexPermissionData.name);
    expect(responseBody.content).toEqual(complexPermissionData.content);
    expect(responseBody.content.conditions).toHaveLength(2);
    expect(responseBody.content.metadata.tags).toContain('workflow');

    // Clean up the complex permission
    await request.delete(`/v1/permissions/${responseBody.id}`);
  });

  test('should update permission with partial data', async ({ request }) => {
    expect(createdPermissionId).toBeTruthy();

    // Only update the description
    const partialUpdateData = {
      description: 'Partially updated description'
    };

    const response = await request.put(`/v1/permissions/${createdPermissionId}`, {
      data: partialUpdateData
    });

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdPermissionId);
    expect(responseBody.description).toBe(partialUpdateData.description);
    // Name should remain the same from previous update
    expect(responseBody.name).toBe('Updated Test Permission');
  });

  test('should delete a permission', async ({ request }) => {
    expect(createdPermissionId).toBeTruthy();

    const response = await request.delete(`/v1/permissions/${createdPermissionId}`);

    expect(response.status()).toBe(204);

    // Verify permission is deleted by trying to fetch it
    const getResponse = await request.get(`/v1/permissions/${createdPermissionId}`);
    expect(getResponse.status()).toBe(404);

    // Clear the ID so cleanup doesn't try to delete again
    createdPermissionId = '';
  });

  test('should return 404 when deleting non-existent permission', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789def';
    const response = await request.delete(`/v1/permissions/${nonExistentId}`);

    expect(response.status()).toBe(404);
  });
});

test.describe.serial('Role Permission Mapping API', () => {
  let testRoleId: string;
  let testPermissionId: string;

  test.beforeAll(async ({ request }) => {
    // Create a test role for role-permission mapping testing
    const roleData = {
      name: 'Test Role for Permissions Mapping',
      description: 'Role created for testing role-permission mapping functionality',
      is_system_role: false
    };

    const roleResponse = await request.post('/v1/roles', {
      data: roleData
    });

    expect(roleResponse.status()).toBe(201);
    const roleBody = await roleResponse.json();
    testRoleId = roleBody.id;
    console.log(`Created test role with ID: ${testRoleId}`);

    // Create a test permission for role-permission mapping testing
    const permissionData = {
      name: 'Test Permission for Role Mapping',
      description: 'Permission created for testing role-permission mapping functionality',
      content: {
        "action": "test_mapping_action",
        "resource": "test_mapping_resource",
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
    // Clean up test role and permission
    if (testRoleId) {
      await request.delete(`/v1/roles/${testRoleId}`);
    }
    if (testPermissionId) {
      await request.delete(`/v1/permissions/${testPermissionId}`);
    }
  });

  test('should list permissions for role (initially empty)', async ({ request }) => {
    expect(testRoleId).toBeTruthy();

    const response = await request.get(`/v1/roles/${testRoleId}/permissions`);

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(Array.isArray(responseBody)).toBe(true);
    // Should be empty initially
    expect(responseBody).toHaveLength(0);
  });

  test('should add permission to role', async ({ request }) => {
    expect(testRoleId).toBeTruthy();
    expect(testPermissionId).toBeTruthy();

    const permissionData = {
      permission_id: testPermissionId
    };
    console.log(`Adding permission ${testPermissionId} to role ${testRoleId}`);

    const response = await request.post(`/v1/roles/${testRoleId}/permissions`, {
      data: permissionData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('mapping_id');
    expect(responseBody.role_id).toBe(testRoleId);
    expect(responseBody.permission_id).toBe(testPermissionId);
    expect(responseBody).toHaveProperty('assigned_at');
    expect(responseBody).toHaveProperty('assigned_by');
  });

  test('should list permissions for role (after adding)', async ({ request }) => {
    expect(testRoleId).toBeTruthy();
    expect(testPermissionId).toBeTruthy();

    const response = await request.get(`/v1/roles/${testRoleId}/permissions`);

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(Array.isArray(responseBody)).toBe(true);
    expect(responseBody.length).toBeGreaterThanOrEqual(1);
    
    const permission = responseBody.find((p: any) => p.permission_id === testPermissionId);
    expect(permission).toBeDefined();
    expect(permission.role_id).toBe(testRoleId);
  });

  test('should return 409 when adding duplicate permission', async ({ request }) => {
    expect(testRoleId).toBeTruthy();
    expect(testPermissionId).toBeTruthy();

    const permissionData = {
      permission_id: testPermissionId
    };

    const response = await request.post(`/v1/roles/${testRoleId}/permissions`, {
      data: permissionData
    });

    expect(response.status()).toBe(409);
  });

  test('should return 404 when adding permission to non-existent role', async ({ request }) => {
    const nonExistentRoleId = '12345678-1234-1234-1234-123456789000';
    expect(testPermissionId).toBeTruthy();

    const permissionData = {
      permission_id: testPermissionId
    };

    const response = await request.post(`/v1/roles/${nonExistentRoleId}/permissions`, {
      data: permissionData
    });

    expect(response.status()).toBe(404);
  });

  test('should return 404 when adding non-existent permission', async ({ request }) => {
    expect(testRoleId).toBeTruthy();

    const nonExistentPermissionId = '12345678-1234-1234-1234-123456789001';
    const permissionData = {
      permission_id: nonExistentPermissionId
    };

    const response = await request.post(`/v1/roles/${testRoleId}/permissions`, {
      data: permissionData
    });

    expect(response.status()).toBe(404);
  });

  test('should handle validation errors for add permission - missing fields', async ({ request }) => {
    expect(testRoleId).toBeTruthy();

    const invalidData = {
      // Missing required permission_id field
    };

    const response = await request.post(`/v1/roles/${testRoleId}/permissions`, {
      data: invalidData
    });

    expect(response.status()).toBe(400);
  });

  test('should remove permission from role', async ({ request }) => {
    expect(testRoleId).toBeTruthy();
    expect(testPermissionId).toBeTruthy();

    const response = await request.delete(`/v1/roles/${testRoleId}/permissions/${testPermissionId}`);

    expect(response.status()).toBe(204);

    // Verify permission is removed
    const listResponse = await request.get(`/v1/roles/${testRoleId}/permissions`);
    expect(listResponse.status()).toBe(200);
    
    const listBody = await listResponse.json();
    const permission = listBody.find((p: any) => p.permission_id === testPermissionId);
    expect(permission).toBeUndefined();
  });

  test('should return 404 when removing permission from non-existent role', async ({ request }) => {
    const nonExistentRoleId = '12345678-1234-1234-1234-123456789002';
    const nonExistentPermissionId = '12345678-1234-1234-1234-123456789003';

    const response = await request.delete(`/v1/roles/${nonExistentRoleId}/permissions/${nonExistentPermissionId}`);

    expect(response.status()).toBe(404);
  });

  test('should return 404 when removing non-existent permission', async ({ request }) => {
    expect(testRoleId).toBeTruthy();

    const nonExistentPermissionId = '12345678-1234-1234-1234-123456789004';

    const response = await request.delete(`/v1/roles/${testRoleId}/permissions/${nonExistentPermissionId}`);

    expect(response.status()).toBe(404);
  });

  test('should return 404 for permissions on non-existent role', async ({ request }) => {
    const nonExistentRoleId = '12345678-1234-1234-1234-123456789005';

    const response = await request.get(`/v1/roles/${nonExistentRoleId}/permissions`);

    expect(response.status()).toBe(404);
  });

  test('should add multiple permissions to role', async ({ request }) => {
    expect(testRoleId).toBeTruthy();

    // Create additional test permissions
    const permission2Data = {
      name: 'Test Permission 2 for Role Mapping',
      description: 'Second permission for role mapping tests',
      content: {
        "action": "create",
        "resource": "workflows",
        "conditions": ["authenticated"]
      }
    };

    const permission3Data = {
      name: 'Test Permission 3 for Role Mapping',
      description: 'Third permission for role mapping tests',
      content: {
        "action": "delete",
        "resource": "agents",
        "conditions": ["admin_only"]
      }
    };

    // Create the additional permissions
    const perm2Response = await request.post('/v1/permissions', {
      data: permission2Data
    });
    expect(perm2Response.status()).toBe(201);
    const perm2Body = await perm2Response.json();
    const permission2Id = perm2Body.id;

    const perm3Response = await request.post('/v1/permissions', {
      data: permission3Data
    });
    expect(perm3Response.status()).toBe(201);
    const perm3Body = await perm3Response.json();
    const permission3Id = perm3Body.id;

    // Add both permissions to the role
    const addPerm2Response = await request.post(`/v1/roles/${testRoleId}/permissions`, {
      data: { permission_id: permission2Id }
    });
    expect(addPerm2Response.status()).toBe(201);

    const addPerm3Response = await request.post(`/v1/roles/${testRoleId}/permissions`, {
      data: { permission_id: permission3Id }
    });
    expect(addPerm3Response.status()).toBe(201);

    // Verify all permissions are listed for the role
    const listResponse = await request.get(`/v1/roles/${testRoleId}/permissions`);
    expect(listResponse.status()).toBe(200);
    
    const listBody = await listResponse.json();
    expect(Array.isArray(listBody)).toBe(true);
    expect(listBody.length).toBe(2); // Should have 2 permissions now

    const permissionIds = listBody.map((p: any) => p.permission_id);
    expect(permissionIds).toContain(permission2Id);
    expect(permissionIds).toContain(permission3Id);

    // Clean up the additional permissions
    await request.delete(`/v1/permissions/${permission2Id}`);
    await request.delete(`/v1/permissions/${permission3Id}`);
  });
});