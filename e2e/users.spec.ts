import { test, expect } from '@playwright/test';

test.describe.serial('Users API', () => {
  let createdUserId: string;
  let createdRoleId: string;

  test.beforeAll(async ({ request }) => {
    // Clean up any existing test users before starting
    const response = await request.get('/v1/users');
    if (response.ok()) {
      const responseBody = await response.json();
      const users = responseBody.users || [];
      if (Array.isArray(users)) {
        for (const user of users) {
          if (user.name && user.name.startsWith('testuser')) {
            await request.delete(`/v1/users/${user.id}`);
          }
        }
      }
    }

    // Create a test role for role assignment tests
    const roleData = {
      name: 'Test Role for User Testing',
      description: 'Role created for testing user role functionality'
    };

    const roleResponse = await request.post('/v1/roles', {
      data: roleData
    });

    if (roleResponse.status() === 201) {
      const roleBody = await roleResponse.json();
      createdRoleId = roleBody.id;
    }
  });

  test.afterAll(async ({ request }) => {
    // Clean up created user
    if (createdUserId) {
      await request.delete(`/v1/users/${createdUserId}`);
    }
    // Clean up created role
    if (createdRoleId) {
      await request.delete(`/v1/roles/${createdRoleId}`);
    }
  });

  test('should create a new user', async ({ request }) => {
    const userData = {
      name: 'testuser_api_testing',
      email: 'testuser@example.com',
      password_hash: '$2a$10$exampleHashForTestingPurposes123456',
      provider_name: 'local',
      additional_info: {
        department: 'Engineering',
        location: 'Remote',
        timezone: 'UTC'
      }
    };

    const response = await request.post('/v1/users', {
      data: userData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('id');
    expect(responseBody.name).toBe(userData.name);
    expect(responseBody.email).toBe(userData.email);
    expect(responseBody.provider_name).toBe(userData.provider_name);
    expect(responseBody).toHaveProperty('created_at');
    expect(responseBody).toHaveProperty('updated_at');
    // Password hash should not be returned in response for security
    expect(responseBody.password_hash).toBeUndefined();

    createdUserId = responseBody.id;
  });

  test('should get all users', async ({ request }) => {
    const response = await request.get('/v1/users');

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('users');
    expect(responseBody.users).toBeDefined();
    expect(Array.isArray(responseBody.users)).toBe(true);
    expect(responseBody.users.length).toBeGreaterThanOrEqual(1); // At least our created user
  });

  test('should get user by ID', async ({ request }) => {
    expect(createdUserId).toBeTruthy();

    const response = await request.get(`/v1/users/${createdUserId}`);

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdUserId);
    expect(responseBody.name).toBe('testuser_api_testing');
    expect(responseBody.email).toBe('testuser@example.com');
    expect(responseBody.provider_name).toBe('local');
  });

  test('should return 404 for non-existent user', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789012';
    const response = await request.get(`/v1/users/${nonExistentId}`);

    expect(response.status()).toBe(404);
    
    const responseBody = await response.json();
    expect(responseBody.resource).toBe("User");
    expect(responseBody.id).toBe(nonExistentId);
  });

  test('should return 400 for invalid user ID format', async ({ request }) => {
    const invalidId = 'invalid-uuid';
    const response = await request.get(`/v1/users/${invalidId}`);

    expect(response.status()).toBe(400);
  });

  test('should return 404 for nil UUID (validation error)', async ({ request }) => {
    const nilUuid = '00000000-0000-0000-0000-000000000000';
    const response = await request.get(`/v1/users/${nilUuid}`);

    expect(response.status()).toBe(404);
    
    const responseBody = await response.json();
    expect(responseBody.resource).toBe("User");
    expect(responseBody.id).toBe(nilUuid);
  });

  test('should update an existing user', async ({ request }) => {
    expect(createdUserId).toBeTruthy();

    const updateData = {
      username: 'updated_testuser',
      email: 'updated_testuser@example.com',
      provider_name: 'google',
      is_online: true,
      additional_info: {
        department: 'Product',
        location: 'San Francisco',
        timezone: 'PST',
        role: 'Senior Engineer'
      }
    };

    const response = await request.put(`/v1/users/${createdUserId}`, {
      data: updateData
    });

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdUserId);
    expect(responseBody.name).toBe(updateData.username);
    expect(responseBody.email).toBe(updateData.email);
    expect(responseBody.provider_name).toBe(updateData.provider_name);
    expect(responseBody.additional_info).toStrictEqual(updateData.additional_info);
  });

  test('should return 404 when updating non-existent user', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789abc';
    const updateData = {
      username: 'non_existent_user',
      email: 'nonexistent@example.com'
    };

    const response = await request.put(`/v1/users/${nonExistentId}`, {
      data: updateData
    });

    expect(response.status()).toBe(404);
  });

  test('should handle validation errors for create user - missing required fields', async ({ request }) => {
    const invalidUserData = {
      // Missing required name, email, and password_hash fields
      provider_name: 'local'
    };

    const response = await request.post('/v1/users', {
      data: invalidUserData
    });

    expect(response.status()).toBe(400);
  });

  test('should handle validation errors for username too long', async ({ request }) => {
    const invalidUserData = {
      name: 'A'.repeat(300), // Exceeds maxLength of 255
      email: 'test@example.com',
      password_hash: '$2a$10$exampleHashForTestingPurposes123456'
    };

    const response = await request.post('/v1/users', {
      data: invalidUserData
    });

    expect(response.status()).toBe(400);
  });

  test('should handle validation errors for invalid email format', async ({ request }) => {
    const invalidUserData = {
      name: 'testuser_invalid_email',
      email: 'invalid-email-format', // Invalid email format
      password_hash: '$2a$10$exampleHashForTestingPurposes123456'
    };

    const response = await request.post('/v1/users', {
      data: invalidUserData
    });

    expect(response.status()).toBe(400);
  });

  test('should create user with minimal required fields', async ({ request }) => {
    const minimalUserData = {
      name: 'minimal_testuser',
      email: 'minimal@example.com',
      password_hash: '$2a$10$exampleHashForTestingPurposes123456'
    };

    const response = await request.post('/v1/users', {
      data: minimalUserData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody.name).toBe(minimalUserData.name);
    expect(responseBody.email).toBe(minimalUserData.email);
    expect(responseBody).toHaveProperty('id');

    // Clean up the minimal user
    await request.delete(`/v1/users/${responseBody.id}`);
  });

  test('should return 409 when creating user with duplicate username', async ({ request }) => {
    // First, create a user
    const userData = {
      name: 'duplicate_username_test',
      email: 'unique_email1@example.com',
      password_hash: '$2a$10$exampleHashForTestingPurposes123456'
    };

    const response1 = await request.post('/v1/users', {
      data: userData
    });

    expect(response1.status()).toBe(201);
    const user1 = await response1.json();

    // Now try to create another user with the same username but different email
    const duplicateUsernameData = {
      name: 'duplicate_username_test', // Same username
      email: 'unique_email2@example.com',   // Different email
      password_hash: '$2a$10$exampleHashForTestingPurposes123456'
    };

    const response2 = await request.post('/v1/users', {
      data: duplicateUsernameData
    });

    expect(response2.status()).toBe(409);
    
    const responseBody = await response2.json();
    expect(responseBody.resource).toBe('User');
    expect(responseBody.message).toContain('Username already exists');

    // Clean up
    await request.delete(`/v1/users/${user1.id}`);
  });

  test('should return 409 when creating user with duplicate email', async ({ request }) => {
    // First, create a user
    const userData = {
      name: 'unique_username1',
      email: 'duplicate_email_test@example.com',
      password_hash: '$2a$10$exampleHashForTestingPurposes123456'
    };

    const response1 = await request.post('/v1/users', {
      data: userData
    });

    expect(response1.status()).toBe(201);
    const user1 = await response1.json();

    // Now try to create another user with different username but same email
    const duplicateEmailData = {
      name: 'unique_username2',               // Different username
      email: 'duplicate_email_test@example.com', // Same email
      password_hash: '$2a$10$exampleHashForTestingPurposes123456'
    };

    const response2 = await request.post('/v1/users', {
      data: duplicateEmailData
    });

    expect(response2.status()).toBe(409);
    
    const responseBody = await response2.json();
    expect(responseBody.resource).toBe('User');
    expect(responseBody.message).toContain('Email already exists');

    // Clean up
    await request.delete(`/v1/users/${user1.id}`);
  });

  test('should update user with partial data', async ({ request }) => {
    expect(createdUserId).toBeTruthy();

    // Only update the username
    const partialUpdateData = {
      username: 'partially_updated_user'
    };

    const response = await request.put(`/v1/users/${createdUserId}`, {
      data: partialUpdateData
    });

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdUserId);
    expect(responseBody.name).toBe(partialUpdateData.username);
    // Email should remain the same from previous update
    expect(responseBody.email).toBe('updated_testuser@example.com');
  });

  test('should return 409 when updating user with existing username', async ({ request }) => {
    // Create two users
    const user1Data = {
      name: 'conflict_user1',
      email: 'conflict1@example.com',
      password_hash: '$2a$10$exampleHashForTestingPurposes123456'
    };

    const user2Data = {
      name: 'conflict_user2',
      email: 'conflict2@example.com',
      password_hash: '$2a$10$exampleHashForTestingPurposes123456'
    };

    const response1 = await request.post('/v1/users', { data: user1Data });
    expect(response1.status()).toBe(201);
    const user1 = await response1.json();

    const response2 = await request.post('/v1/users', { data: user2Data });
    expect(response2.status()).toBe(201);
    const user2 = await response2.json();

    // Try to update user2 with user1's username
    const updateData = {
      username: 'conflict_user1' // This username already exists
    };

    const updateResponse = await request.put(`/v1/users/${user2.id}`, {
      data: updateData
    });

    expect(updateResponse.status()).toBe(409);
    
    const responseBody = await updateResponse.json();
    expect(responseBody.resource).toBe('User');
    expect(responseBody.message).toContain('Username already exists');

    // Clean up
    await request.delete(`/v1/users/${user1.id}`);
    await request.delete(`/v1/users/${user2.id}`);
  });

  test('should return 409 when updating user with existing email', async ({ request }) => {
    // Create two users
    const user1Data = {
      name: 'email_conflict_user1',
      email: 'email_conflict1@example.com',
      password_hash: '$2a$10$exampleHashForTestingPurposes123456'
    };

    const user2Data = {
      name: 'email_conflict_user2',
      email: 'email_conflict2@example.com',
      password_hash: '$2a$10$exampleHashForTestingPurposes123456'
    };

    const response1 = await request.post('/v1/users', { data: user1Data });
    expect(response1.status()).toBe(201);
    const user1 = await response1.json();

    const response2 = await request.post('/v1/users', { data: user2Data });
    expect(response2.status()).toBe(201);
    const user2 = await response2.json();

    // Try to update user2 with user1's email
    const updateData = {
      email: 'email_conflict1@example.com' // This email already exists
    };

    const updateResponse = await request.put(`/v1/users/${user2.id}`, {
      data: updateData
    });

    expect(updateResponse.status()).toBe(409);
    
    const responseBody = await updateResponse.json();
    expect(responseBody.resource).toBe('User');
    expect(responseBody.message).toContain('Email already exists');

    // Clean up
    await request.delete(`/v1/users/${user1.id}`);
    await request.delete(`/v1/users/${user2.id}`);
  });

  test('should return 400 when updating user with invalid field lengths', async ({ request }) => {
    expect(createdUserId).toBeTruthy();

    const invalidUpdateData = {
      username: 'A'.repeat(300), // Exceeds maxLength of 255
      email: 'B'.repeat(250) + '@example.com' // Exceeds maxLength of 255
    };

    const response = await request.put(`/v1/users/${createdUserId}`, {
      data: invalidUpdateData
    });

    expect(response.status()).toBe(400);
  });

  test('should delete a user', async ({ request }) => {
    expect(createdUserId).toBeTruthy();

    const response = await request.delete(`/v1/users/${createdUserId}`);

    expect(response.status()).toBe(204);

    // Verify user is deleted by trying to fetch it
    const getResponse = await request.get(`/v1/users/${createdUserId}`);
    expect(getResponse.status()).toBe(404);

    // Clear the ID so cleanup doesn't try to delete again
    createdUserId = '';
  });

  test('should return 404 when deleting non-existent user', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789def';
    const response = await request.delete(`/v1/users/${nonExistentId}`);

    expect(response.status()).toBe(404);
  });
});

test.describe.serial('User Role Management API', () => {
  let testUserId: string;
  let testRoleId: string;

  test.beforeAll(async ({ request }) => {
    // Create a test user for role testing
    const userData = {
      name: 'testuser_for_roles',
      email: 'roletest@example.com',
      password_hash: '$2a$10$exampleHashForTestingPurposes123456',
      provider_name: 'local'
    };

    const userResponse = await request.post('/v1/users', {
      data: userData
    });

    expect(userResponse.status()).toBe(201);
    const userBody = await userResponse.json();
    testUserId = userBody.id;

    // Create a test role for role assignment testing
    const roleData = {
      name: 'Test Role for User Role Management',
      description: 'Role created for testing user role management functionality'
    };

    const roleResponse = await request.post('/v1/roles', {
      data: roleData
    });

    expect(roleResponse.status()).toBe(201);
    const roleBody = await roleResponse.json();
    testRoleId = roleBody.id;
  });

  test.afterAll(async ({ request }) => {
    // Clean up test user and role
    if (testUserId) {
      await request.delete(`/v1/users/${testUserId}`);
    }
    if (testRoleId) {
      await request.delete(`/v1/roles/${testRoleId}`);
    }
  });

  test('should list roles for user (initially have pending role)', async ({ request }) => {
    expect(testUserId).toBeTruthy();

    const response = await request.get(`/v1/users/${testUserId}/roles`);

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(Array.isArray(responseBody)).toBe(true);
    // Should have 1 role as pending initially
    expect(responseBody).toHaveLength(1);
    expect(responseBody[0].role_id).toBe('550e8400-e29b-41d4-a716-446655440003');
  });

  test('should add role to user', async ({ request }) => {
    expect(testUserId).toBeTruthy();
    expect(testRoleId).toBeTruthy();

    const roleData = {
      role_id: testRoleId
    };

    const response = await request.post(`/v1/users/${testUserId}/roles`, {
      data: roleData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('mapping_id');
    expect(responseBody.user_id).toBe(testUserId);
    expect(responseBody.role_id).toBe(testRoleId);
    expect(responseBody).toHaveProperty('assigned_at');
    expect(responseBody).toHaveProperty('assigned_by');
  });

  test('should list roles for user (after adding)', async ({ request }) => {
    expect(testUserId).toBeTruthy();
    expect(testRoleId).toBeTruthy();

    const response = await request.get(`/v1/users/${testUserId}/roles`);

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(Array.isArray(responseBody)).toBe(true);
    expect(responseBody.length).toBeGreaterThanOrEqual(1);
    
    const roleMapping = responseBody.find((r: any) => r.role_id === testRoleId);
    expect(roleMapping).toBeDefined();
    expect(roleMapping.user_id).toBe(testUserId);
  });

  test('should return 409 when adding duplicate role', async ({ request }) => {
    expect(testUserId).toBeTruthy();
    expect(testRoleId).toBeTruthy();

    const roleData = {
      role_id: testRoleId,
      assigned_by: testUserId
    };

    const response = await request.post(`/v1/users/${testUserId}/roles`, {
      data: roleData
    });

    expect(response.status()).toBe(409);
  });

  test('should return 404 when adding role to non-existent user', async ({ request }) => {
    const nonExistentUserId = '12345678-1234-1234-1234-123456789000';
    expect(testRoleId).toBeTruthy();

    const roleData = {
      role_id: testRoleId
    };

    const response = await request.post(`/v1/users/${nonExistentUserId}/roles`, {
      data: roleData
    });

    expect(response.status()).toBe(404);
  });

  test('should return 404 when adding non-existent role', async ({ request }) => {
    expect(testUserId).toBeTruthy();

    const nonExistentRoleId = '12345678-1234-1234-1234-123456789001';
    const roleData = {
      role_id: nonExistentRoleId
    };

    const response = await request.post(`/v1/users/${testUserId}/roles`, {
      data: roleData
    });

    expect(response.status()).toBe(404);
  });

  test('should handle validation errors for add role - missing fields', async ({ request }) => {
    expect(testUserId).toBeTruthy();

    const invalidData = {
      // Missing required role_id field
    };

    const response = await request.post(`/v1/users/${testUserId}/roles`, {
      data: invalidData
    });

    expect(response.status()).toBe(400);
  });

  test('should remove role from user', async ({ request }) => {
    expect(testUserId).toBeTruthy();
    expect(testRoleId).toBeTruthy();

    const response = await request.delete(`/v1/users/${testUserId}/roles/${testRoleId}`);

    expect(response.status()).toBe(204);

    // Verify role is removed
    const listResponse = await request.get(`/v1/users/${testUserId}/roles`);
    expect(listResponse.status()).toBe(200);
    
    const listBody = await listResponse.json();
    const roleMapping = listBody.find((r: any) => r.role_id === testRoleId);
    expect(roleMapping).toBeUndefined();
  });

  test('should return 404 when removing role from non-existent user', async ({ request }) => {
    const nonExistentUserId = '12345678-1234-1234-1234-123456789002';
    const nonExistentRoleId = '12345678-1234-1234-1234-123456789003';

    const response = await request.delete(`/v1/users/${nonExistentUserId}/roles/${nonExistentRoleId}`);

    expect(response.status()).toBe(404);
  });

  test('should return 404 when removing non-existent role', async ({ request }) => {
    expect(testUserId).toBeTruthy();

    const nonExistentRoleId = '12345678-1234-1234-1234-123456789004';

    const response = await request.delete(`/v1/users/${testUserId}/roles/${nonExistentRoleId}`);

    expect(response.status()).toBe(404);
  });

  test('should return 404 for roles on non-existent user', async ({ request }) => {
    const nonExistentUserId = '12345678-1234-1234-1234-123456789005';

    const response = await request.get(`/v1/users/${nonExistentUserId}/roles`);

    expect(response.status()).toBe(404);
  });
});