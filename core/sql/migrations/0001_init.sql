-- Database initialization script for PostgreSQL Docker Compose
-- This file will be executed automatically when the PostgreSQL container starts
-- Set client encoding and timezone
-- +goose Up

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =============================================
-- SCHEMA CREATION AND UTILITY FUNCTIONS
-- =============================================
-- Function to automatically update the updated_at timestamp
-- +goose statementbegin
DROP FUNCTION IF EXISTS trigger_set_timestamp () CASCADE;
CREATE OR REPLACE FUNCTION trigger_set_timestamp () RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW(); -- Sets updated_at to the current transaction timestamp
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose statementend

-- =============================================
-- PERMISSIONS TABLES
-- =============================================

-- Table to store permissions
CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT, -- Optional description for the permission
    content JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexing the name column for performance
CREATE INDEX IF NOT EXISTS idx_permissions_name ON permissions (name);

-- Trigger to update the updated_at column on row update for permissions
DROP TRIGGER IF EXISTS set_timestamp_permissions ON permissions;
CREATE TRIGGER set_timestamp_permissions BEFORE
UPDATE ON permissions FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp ();

-- Insert Admin permissions data into permissions table
INSERT INTO permissions (name, description, content)
VALUES
    ('allowAccessGetAgentsEndpoint', 'Permission to access get all agents endpoint', '{"action": ["GET"], "effect": "allow", "resource": ["/v1/agents"]}'),
    ('allowAccessGetAgentByIdEndpoint', 'Permission to access get agent by ID endpoint', '{"action": ["GET"], "effect": "allow", "resource": ["/v1/agents/*"]}'),
    ('allowAccessCreateAgentEndpoint', 'Permission to access create agent endpoint', '{"action": ["POST"], "effect": "allow", "resource": ["/v1/agents"]}'),
    ('allowAccessCreateAgentPermissionsEndpoint', 'Permission to access create agent permissions endpoint', '{"action": ["POST"], "effect": "allow", "resource": ["/v1/agents/*/permissions"]}'),
    ('allowAccessDeleteAgentPermissionsEndpoint', 'Permission to access delete agent permissions endpoint', '{"action": ["DELETE"], "effect": "allow", "resource": ["/v1/agents/*/permissions"]}'),
    ('allowAccessGetAgentPermissionsEndpoint', 'Permission to access get agent permissions endpoint', '{"action": ["GET"], "effect": "allow", "resource": ["/v1/agents/*/permissions"]}'),
    ('allowAccessDeleteAgentEndpoint', 'Permission to access delete agent endpoint', '{"action": ["DELETE"], "effect": "allow", "resource": ["/v1/agents/*"]}'),
    ('allowAccessGetFilesEndpoint', 'Permission to access get all files endpoint', '{"action": ["GET"], "effect": "allow", "resource": ["/v1/files"]}'),
    ('allowAccessGetFileByIdEndpoint', 'Permission to access get file by ID endpoint', '{"action": ["GET"], "effect": "allow", "resource": ["/v1/files/*"]}'),
    ('allowAccessAddFileRoleEndpoint', 'Permission to access add file role endpoint', '{"action": ["POST"], "effect": "allow", "resource": ["/v1/files/*"]}'),
    ('allowAccessDeleteFileRoleEndpoint', 'Permission to access delete file role endpoint', '{"action": ["DELETE"], "effect": "allow", "resource": ["/v1/files/*"]}'),
    ('allowAccessUploadFileEndpoint', 'Permission to access upload file endpoint', '{"action": ["POST"], "effect": "allow", "resource": ["/v1/files"]}'),
    ('allowAccessDeleteFileEndpoint', 'Permission to access delete file endpoint', '{"action": ["DELETE"], "effect": "allow", "resource": ["/v1/files/*"]}'),
    ('allowAccessGetRolesEndpoint', 'Permission to access get all roles endpoint', '{"action": ["GET"], "effect": "allow", "resource": ["/v1/roles"]}'),
    ('allowAccessCreateRolesEndpoint', 'Permission to access create roles endpoint', '{"action": ["POST"], "effect": "allow", "resource": ["/v1/roles"]}'),
    ('allowAccessUpdateRolesEndpoint', 'Permission to access update roles endpoint', '{"action": ["PUT"], "effect": "allow", "resource": ["/v1/roles/*"]}'),
    ('allowAccessDeleteRolesEndpoint', 'Permission to access delete roles endpoint', '{"action": ["DELETE"], "effect": "allow", "resource": ["/v1/roles/*"]}'),
    ('allowAccessGetUsersEndpoint', 'Permission to access get all users endpoint', '{"action": ["GET"], "effect": "allow", "resource": ["/v1/users"]}'),
    ('allowAccessUpdatedUserEndpoint', 'Permission to access update user endpoint', '{"action": ["PUT"], "effect": "allow", "resource": ["/v1/users/*"]}'),
    ('allowAccessDeleteUserEndpoint', 'Permission to access delete user endpoint', '{"action": ["DELETE"], "effect": "allow", "resource": ["/v1/users/*"]}'),
    ('allowAccessAddUserRoleEndpoint', 'Permission to access add user role endpoint', '{"action": ["POST"], "effect": "allow", "resource": ["/v1/users/*/roles"]}'),
    ('allowAccessDeleteUserRolesEndpoint', 'Permission to access delete user roles endpoint', '{"action": ["DELETE"], "effect": "allow", "resource": ["/v1/users/*/roles/*"]}'),
    ('allowAccessAllAgentsEndpoint', 'Permission to access all agents endpoint', '{"action": ["GET", "POST", "PUT", "DELETE", "PATCH"], "effect": "allow", "resource": ["/v1/agents", "/v1/agents/*", "/v1/agents/*/permissions", "/v1/agents/*/permissions/*"]}'),
    ('allowAccessAllRolesEndpoint', 'Permission to access all roles endpoint', '{"action": ["GET", "POST", "PUT", "DELETE"], "effect": "allow", "resource": ["/v1/roles", "/v1/roles/*"]}'),
    ('allowAccessAllPermissionsEndpoint', 'Permission to access all permissions endpoint', '{"action": ["GET", "POST", "PUT", "DELETE"], "effect": "allow", "resource": ["/v1/permissions", "/v1/permissions/*"]}'),
    ('allowAccessAllUsersEndpoint', 'Permission to access all users endpoint', '{"action": ["GET", "POST", "PUT", "DELETE"], "effect": "allow", "resource": ["/v1/users", "/v1/users/*", "/v1/users/*/roles", "/v1/users/*/roles/*"]}'),
    ('allowAccessAllFilesEndpoint', 'Permission to access all files endpoint', '{"action": ["GET", "POST", "PUT", "DELETE"], "effect": "allow", "resource": ["/v1/files", "/v1/files/*"]}'),
    ('allowAccessGetPermissionsEndpoint', 'Permission to access get all permissions endpoint', '{"action": ["GET"], "effect": "allow", "resource": ["/v1/permissions"]}'),
    ('allowAccessCreatePermissionEndpoint', 'Permission to access create permission endpoint', '{"action": ["POST"], "effect": "allow", "resource": ["/v1/permissions"]}'),
    ('allowAccessUpdatePermissionEndpoint', 'Permission to access update permission endpoint', '{"action": ["PUT"], "effect": "allow", "resource": ["/v1/permissions/*"]}'),
    ('allowAccessDeletePermissionEndpoint', 'Permission to access delete permission endpoint', '{"action": ["DELETE"], "effect": "allow", "resource": ["/v1/permissions/*"]}'),
    ('administration', 'Permission to access all endpoints', '{"action": "*", "effect": "allow", "resource": "*"}'),
    
    -- Insert User features permissions data into permissions table
    ('allowAccessBasicChatEndpoint', 'Permission to access basic chat endpoint', '{"action": ["GET", "POST", "PUT", "DELETE"], "effect": "allow", "resource": ["/v1/threads", "/v1/threads/*", "/v1/threads/*/messages", "/v1/threads/*/messages/*"]}'),
    ('allowAccessAllThreadsEndpoint', 'Permission to access all threads endpoint', '{"action": ["GET", "POST", "PUT", "DELETE"], "effect": "allow", "resource": ["/v1/threads", "/v1/threads/*"]}'),
    ('allowAccessAllMessagesEndpoint', 'Permission to access all messages endpoint', '{"action": ["GET", "POST", "PUT", "DELETE"], "effect": "allow", "resource": ["/v1/threads/*/messages", "/v1/threads/*/messages/*"]}'),

    -- Insert permission for access specific model id data into permissions table
    ('allowAccessAllModels', 'Permission to access all models', '{"effect": "allow"}'),
    ('allowAccessAnthropicClaude3Haiku', 'Permission to access Anthropic Claude 3 Haiku model', '{"effect": "allow"}'),
    ('allowAccessAnthropicClaude35Haiku', 'Permission to access Anthropic Claude 3.5 Haiku model', '{"effect": "allow"}'),
    ('allowAccessAnthropicClaude35SonnetV1', 'Permission to access Anthropic Claude 3.5 Sonnet V1 model', '{"effect": "allow"}'),
    ('allowAccessAnthropicClaude35SonnetV2', 'Permission to access Anthropic Claude 3.5 Sonnet V2 model', '{"effect": "allow"}'),
    ('allowAccessAnthropicClaude37Sonnet', 'Permission to access Anthropic Claude 3.7 Sonnet model', '{"effect": "allow"}'),
    ('allowAccessAnthropicClaudeSonnet4', 'Permission to access Anthropic Claude Sonnet 4 model', '{"effect": "allow"}'),
    ('allowAccessAnthropicClaudeOpus4', 'Permission to access Anthropic Claude Opus 4 model', '{"effect": "allow"}'),
    ('allowAccessNovaMicro', 'Permission to access Amazon Nova Micro model', '{"effect": "allow"}'),
    ('allowAccessNovaLite', 'Permission to access Amazon Nova Lite model', '{"effect": "allow"}'),
    ('allowAccessNovaPro', 'Permission to access Amazon Nova Pro model', '{"effect": "allow"}'),
    ('allowAccessLlama4Maverick', 'Permission to access Llama 4 Maverick model', '{"effect": "allow"}'),
    ('allowAccessLlama4Scout', 'Permission to access Llama 4 Scout model', '{"effect": "allow"}'),
    ('allowAccessWriterPalmyraX4', 'Permission to access Writer Palmyra X4 model', '{"effect": "allow"}'),
    ('allowAccessWriterPalmyraX5', 'Permission to access Writer Palmyra X5 model', '{"effect": "allow"}'),
    ('allowAccessDeepSeekR1', 'Permission to access DeepSeek R1 model', '{"effect": "allow"}'),

    -- Insert permission for accessing multi-agent
    ('allowAccessMultiAgents', 'Permission to access multi-agents mode', '{"effect": "allow"}');

-- =============================================
-- ROLE TABLES
-- =============================================

-- Table to store available roles in the system
CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);


-- Trigger to update the updated_at column on row update
DROP TRIGGER IF EXISTS set_timestamp_roles ON roles;
CREATE TRIGGER set_timestamp_roles BEFORE
UPDATE ON roles FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp ();

-- Insert default system roles with specific UUIDs for consistency
INSERT INTO roles (id,name,description,is_system)
VALUES
    ('550e8400-e29b-41d4-a716-446655440001','admin','Administrator with full system access',TRUE),
    ('550e8400-e29b-41d4-a716-446655440002','member','Regular member with standard access',TRUE),
    ('550e8400-e29b-41d4-a716-446655440003','pending','Pending approval for new users',TRUE),
    ('550e8400-e29b-41d4-a716-446655440004','techx','TechX member with access to techx features',TRUE),
    ('550e8400-e29b-41d4-a716-446655440005','dev','Developer with access to development features',TRUE);

-- Index for faster role lookups
CREATE INDEX IF NOT EXISTS idx_roles_name ON roles (name);
CREATE INDEX IF NOT EXISTS idx_roles_system ON roles (is_system);
CREATE INDEX IF NOT EXISTS idx_roles_id ON roles (id);

-- =============================================
-- USER RELATED TABLES
-- =============================================

-- Table to store users information
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    additional_info JSONB, -- Store additional user information in JSONB format
    password_hash TEXT NOT NULL, -- Store securely hashed passwords!
    provider_name VARCHAR(20) NOT NULL CHECK (provider_name in ('local', 'google', 'azure', 'github')) DEFAULT 'local' ,
    is_online BOOLEAN DEFAULT FALSE, -- Track if the user is currently online
    last_login TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Trigger to update the updated_at column on row update
DROP TRIGGER IF EXISTS set_timestamp_users ON users;
CREATE TRIGGER set_timestamp_users BEFORE
UPDATE ON users FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp ();

-- Indexing the email column for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_name ON users (name);

-- Insert system data into users table
INSERT INTO users (id, name, email, password_hash, created_at, updated_at)
VALUES ('550e8400-c95b-4444-6666-000000000000','system','system','1',NOW(),NOW()) 
ON CONFLICT (id) DO NOTHING;

-- Insert admin data into users table
INSERT INTO users (id, name, email, password_hash, created_at, updated_at)
VALUES ('550e8400-c95b-4444-6666-446655440000','admin','admin@email.com','8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92',NOW(),NOW()) 
ON CONFLICT (id) DO NOTHING;

-- Insert member data into users Table
INSERT INTO users (id, name, email, password_hash, created_at, updated_at)
VALUES ('550e8400-c95b-4444-6666-446655440001','member','member@email.com','8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92',NOW(),NOW()) 
ON CONFLICT (id) DO NOTHING;


-- =============================================
-- MAPPING TABLES
-- =============================================

-- Junction table to handle many-to-many relationship between roles and permissions
CREATE TABLE IF NOT EXISTS role_permission_mapping (
    mapping_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID NOT NULL,
    permission_id UUID NOT NULL,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    assigned_by UUID NOT NULL DEFAULT '550e8400-c95b-4444-6666-000000000000', -- Default to system
    
    -- Foreign Key contraint linking to the role table
    CONSTRAINT fk_role_permission_mapping_role 
        FOREIGN KEY (role_id)
        REFERENCES roles (id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,

    -- Foreign Key contraint linking to the permissions table
    CONSTRAINT fk_role_permission_mapping_permission
        FOREIGN KEY (permission_id)
        REFERENCES permissions (id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,

    -- Foreign Key contraint linking to the users table
    CONSTRAINT fk_role_permission_mapping_assigned_by
        FOREIGN KEY (assigned_by)
        REFERENCES users (id)
        ON UPDATE CASCADE
        ON DELETE SET DEFAULT,
    
    -- Ensure unique role-permission combinations
    CONSTRAINT uk_role_permission_mapping UNIQUE (role_id, permission_id)
);

-- Indexes for better performance
CREATE INDEX IF NOT EXISTS idx_role_permission_mapping_role_id ON role_permission_mapping (role_id);
CREATE INDEX IF NOT EXISTS idx_role_permission_mapping_permission_id ON role_permission_mapping (permission_id);
CREATE INDEX IF NOT EXISTS idx_role_permission_mapping_assigned_by ON role_permission_mapping (assigned_by);

-- Insert permissions permissions to admin roles
INSERT INTO role_permission_mapping (role_id, permission_id, assigned_by)
SELECT
    '550e8400-e29b-41d4-a716-446655440001' AS role_id, ip.id,
    '550e8400-c95b-4444-6666-446655440000' AS assigned_by
FROM permissions ip
WHERE ip.name = 'administration'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permission_mapping (role_id, permission_id, assigned_by)
SELECT
    '550e8400-e29b-41d4-a716-446655440001' AS role_id, ip.id,
    '550e8400-c95b-4444-6666-446655440000' AS assigned_by
FROM permissions ip
WHERE ip.name = 'allowAccessAllModels'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Insert permissions permissions to techx roles
INSERT INTO role_permission_mapping (role_id, permission_id, assigned_by)
SELECT
    '550e8400-e29b-41d4-a716-446655440004' AS role_id, ip.id,
    '550e8400-c95b-4444-6666-446655440000' AS assigned_by
FROM permissions ip
WHERE ip.name = 'allowAccessAllFilesEndpoint'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permission_mapping (role_id, permission_id, assigned_by)
SELECT
    '550e8400-e29b-41d4-a716-446655440004' AS role_id, ip.id,
    '550e8400-c95b-4444-6666-446655440000' AS assigned_by
FROM permissions ip
WHERE ip.name = 'allowAccessAnthropicClaudeSonnet4'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permission_mapping (role_id, permission_id, assigned_by)
SELECT
    '550e8400-e29b-41d4-a716-446655440004' AS role_id, ip.id,
    '550e8400-c95b-4444-6666-446655440000' AS assigned_by
FROM permissions ip
WHERE ip.name = 'allowAccessBasicChatEndpoint'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Insert permissions permissions to member roles
INSERT INTO role_permission_mapping (role_id, permission_id, assigned_by)
SELECT
    '550e8400-e29b-41d4-a716-446655440002' AS role_id, ip.id,
    '550e8400-c95b-4444-6666-446655440000' AS assigned_by
FROM permissions ip
WHERE ip.name = 'allowAccessAnthropicClaudeSonnet4'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Junction table to handle many-to-many relationship between users and roles
CREATE TABLE IF NOT EXISTS user_role_mapping (
    mapping_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    role_id UUID NOT NULL,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    assigned_by UUID NOT NULL DEFAULT '550e8400-c95b-4444-6666-000000000000', -- Default to system
    
    -- Foreign Key contraint linking to the users table
    CONSTRAINT fk_user_role_mapping_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,

    -- Foreign Key contraint linking to the roles table
    CONSTRAINT fk_user_role_mapping_role
        FOREIGN KEY (role_id)
        REFERENCES roles (id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,

    -- Foreign Key contraint linking to the users table
    CONSTRAINT fk_user_role_mapping_assigned_by
        FOREIGN KEY (assigned_by)
        REFERENCES users (id)
        ON UPDATE CASCADE
        ON DELETE SET DEFAULT,
    
    -- Ensure unique user-role combinations
    CONSTRAINT uk_user_role_mapping UNIQUE (user_id, role_id)
);

-- Indexes for better performance
CREATE INDEX IF NOT EXISTS idx_user_role_mapping_user_id ON user_role_mapping (user_id);
CREATE INDEX IF NOT EXISTS idx_user_role_mapping_role_id ON user_role_mapping (role_id);
CREATE INDEX IF NOT EXISTS idx_user_role_mapping_assigned_by ON user_role_mapping (assigned_by);

-- Function to automatically assign pending role to new users
-- +goose statementbegin
DROP FUNCTION IF EXISTS assign_pending_role_to_new_user () CASCADE;
CREATE OR REPLACE FUNCTION assign_pending_role_to_new_user () RETURNS TRIGGER AS $$
BEGIN
    -- Automatically assign the 'pending' role as primary when a new user is created
    INSERT INTO user_role_mapping (mapping_id, user_id, role_id)
    VALUES (gen_random_uuid(),NEW.id,'550e8400-e29b-41d4-a716-446655440003')
    ON CONFLICT (user_id, role_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose statementend

-- Trigger to automatically assign pending role when new user registers
DROP TRIGGER IF EXISTS trigger_assign_pending_role_to_new_user ON users;
CREATE TRIGGER trigger_assign_pending_role_to_new_user
AFTER INSERT ON users FOR EACH ROW
EXECUTE FUNCTION assign_pending_role_to_new_user ();

-- Remove the pending role and add admin role to the admin user
DELETE FROM user_role_mapping WHERE user_id = '550e8400-c95b-4444-6666-446655440000' AND role_id = '550e8400-e29b-41d4-a716-446655440003';
INSERT INTO user_role_mapping (mapping_id, user_id, role_id) VALUES (gen_random_uuid(), '550e8400-c95b-4444-6666-446655440000', '550e8400-e29b-41d4-a716-446655440001') ON CONFLICT (user_id, role_id) DO NOTHING;

-- Remove the pending role and add member role to the member user
DELETE FROM user_role_mapping WHERE user_id = '550e8400-c95b-4444-6666-446655440001' AND role_id = '550e8400-e29b-41d4-a716-446655440003';
INSERT INTO user_role_mapping (mapping_id, user_id, role_id) VALUES (gen_random_uuid(), '550e8400-c95b-4444-6666-446655440001', '550e8400-e29b-41d4-a716-446655440002') ON CONFLICT (user_id, role_id) DO NOTHING;


-- =============================================
-- AGENT RELATED TABLES
-- =============================================

-- Table to store agents information
CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    specs TEXT,
    created_by UUID NOT NULL, -- Track who created this agent
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Foreign Key contraint linking to the users table
    CONSTRAINT fk_agent_created_by
        FOREIGN KEY (created_by)
        REFERENCES users (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);

-- Trigger to update the updated_at column on row update
DROP TRIGGER IF EXISTS set_timestamp_agents ON agents;
CREATE TRIGGER set_timestamp_agents BEFORE
UPDATE ON agents FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp ();

-- Table to store agent permissions mapping
CREATE TABLE IF NOT EXISTS agent_permission_mapping (
    mapping_id UUID NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    assigned_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- Track who assigned this role (optional)
    
    -- Foreign Key contraint linking to the agents table
    CONSTRAINT fk_user_role_mapping_agent 
        FOREIGN KEY (agent_id) 
        REFERENCES agents (id) 
        ON UPDATE CASCADE 
        ON DELETE CASCADE,

    -- Foreign Key contraint linking to the permissions table
    CONSTRAINT fk_user_role_mapping_permission 
        FOREIGN KEY (permission_id) 
        REFERENCES permissions (id) 
        ON UPDATE CASCADE 
        ON DELETE CASCADE,

    -- Foreign Key contraint linking to the users table
    CONSTRAINT fk_user_role_mapping_assigned_by 
        FOREIGN KEY (assigned_by) 
        REFERENCES users (id) 
        ON UPDATE CASCADE 
        ON DELETE SET NULL,

    -- Ensure unique agent-permission combinations
    CONSTRAINT uk_agent_permission_mapping UNIQUE (agent_id, permission_id)
);

-- Indexes for better performance
CREATE INDEX IF NOT EXISTS idx_agent_permission_mapping_agent_id ON agent_permission_mapping (agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_permission_mapping_permission_id ON agent_permission_mapping (permission_id);

-- Insert initial agent
INSERT INTO agents (id, name, description, specs, created_by)
VALUES (
    '550e8400-c95b-5555-6666-446655440000',
    'supervisor',
    'Supervisor agent for managing and orchestrating other agents',
    E'model:\n provider: "bedrock/anthropic"\n model_id: "apac.anthropic.claude-sonnet-4-20250514-v1:0"\n max_tokens: 8192\n thinking:\n  enabled: true\n  budget_token: 1024\n stream: true\n\nsystem: |\n Your name is BOB.',
    '550e8400-c95b-4444-6666-446655440000'
)
ON CONFLICT (name) DO NOTHING;

-- Insert agent permissions
INSERT INTO agent_permission_mapping (agent_id, permission_id, assigned_by)
VALUES (
    (SELECT id FROM agents WHERE name = 'supervisor'),
    (SELECT id FROM permissions WHERE name = 'administration'),
    '550e8400-c95b-4444-6666-446655440000'
)
ON CONFLICT (agent_id, permission_id) DO NOTHING;

-- =============================================
-- THREAD RELATED TABLES
-- =============================================

-- Table to store threads information
CREATE TABLE IF NOT EXISTS threads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    user_id UUID NOT NULL,
    
    -- Foreign Key constraint linking to the users table
    CONSTRAINT fk_thread_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE
);

-- Indexing the foreign key is often beneficial for performance
CREATE INDEX IF NOT EXISTS idx_thread_user_id ON threads (user_id);
CREATE INDEX IF NOT EXISTS idx_thread_updated_at ON threads (updated_at);

-- For ordering threads
DROP TRIGGER IF EXISTS set_timestamp_threads ON threads;
CREATE TRIGGER set_timestamp_threads BEFORE
UPDATE ON threads FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp ();

-- Table creation for thread context with vector support
CREATE TABLE IF NOT EXISTS thread_context (
    context_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    summary TEXT,
    action_contexts JSONB,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Trigger to update the updated_at column on row update
DROP TRIGGER IF EXISTS set_timestamp_thread_context ON thread_context;
CREATE TRIGGER set_timestamp_thread_context BEFORE
UPDATE ON thread_context FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp ();

-- Table to store messages within a thread
CREATE TABLE IF NOT EXISTS thread_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id UUID NOT NULL,
    message JSONB NOT NULL,
    sender_type VARCHAR(255) NOT NULL CHECK (sender_type IN ('user', 'assistant', 'system', 'result')),
    result_type VARCHAR(255) CHECK (result_type IN ('text', 'image', 'code')),
    stop_reason VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sender_id UUID NOT NULL, -- User or agent id
    citations JSONB[] DEFAULT NULL, -- Optional citations for the message
    recipient_id UUID NOT NULL, -- User or agent id
    
    -- Foreign Key constraint linking to the thread table
    CONSTRAINT fk_thread_message
        FOREIGN KEY (thread_id)
        REFERENCES threads (id)
        ON DELETE CASCADE
);

-- Indexing the foreign key
CREATE INDEX IF NOT EXISTS idx_thread_message_thread_id ON thread_messages (thread_id);
CREATE INDEX IF NOT EXISTS idx_thread_message_created_at ON thread_messages (created_at);

-- For ordering messages
CREATE INDEX IF NOT EXISTS idx_thread_messages_sender_id ON thread_messages (sender_id);
CREATE INDEX IF NOT EXISTS idx_thread_messages_recipient_id ON thread_messages (recipient_id);
CREATE INDEX IF NOT EXISTS idx_thread_messages_sender_type ON thread_messages (sender_type);

-- Trigger to update the updated_at column on row update
DROP TRIGGER IF EXISTS set_timestamp_thread_messages ON thread_messages;
CREATE TRIGGER set_timestamp_thread_messages BEFORE
UPDATE ON thread_messages FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();

-- =============================================
-- TOOLS TABLES
-- =============================================

CREATE TABLE IF NOT EXISTS tools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    config JSONB, -- Configuration for the tool
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID NOT NULL DEFAULT '550e8400-c95b-4444-6666-000000000000', -- Default to system
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Foreign Key contraint linking to the users table
    CONSTRAINT fk_tools_created_by
        FOREIGN KEY (created_by)
        REFERENCES users(id)
        ON DELETE SET NULL
);

-- Add indexes for performance
CREATE INDEX IF NOT EXISTS idx_tools_name ON tools (name);
CREATE INDEX IF NOT EXISTS idx_tools_created_by ON tools (created_by);

-- Trigger to update the updated_at column on row update
DROP TRIGGER IF EXISTS set_timestamp_tools ON tools;
CREATE TRIGGER set_timestamp_tools BEFORE
UPDATE ON tools FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp ();

-- Add default internal tools
INSERT INTO tools (name, description, config, created_by)
VALUES ('temp_parallel_tool_management', 'System tool for managing parallel tool execution', '{"type": "internal", "params": {}}', '550e8400-c95b-4444-6666-000000000000')
ON CONFLICT (name) DO NOTHING;

INSERT INTO tools (id, name, description, config, created_by)
VALUES ('550e8400-c00b-8888-2222-446655447896', 'batch_tool', 'Invoke multiple other tool calls simultaneously', '{"type": "internal", "params": {"type": "object","properties": {"invocations": {"type": "array","description": "The tool calls to invoke","items": {"types": "object","properties": {"name": {"types": "string","description": "The name of the tool to invoke"},"arguments": {"types": "string","description": "The arguments to the tool"}},"required": ["name", "arguments"]}}},"required": ["invocations"]}}', '550e8400-c95b-4444-6666-000000000000')
ON CONFLICT (name) DO NOTHING;

INSERT INTO tools (id, name, description, config, created_by)
VALUES ('550e8400-c00b-8888-3333-446655447896', 'invoke_agent', 'Invoke another agent with a query', '{"type": "internal", "params": {"type": "object", "properties": {"agent_id": {"type": "string", "description": "UUID of the agent to invoke"}, "query": {"type": "string", "description": "The query or message to send to the agent"}}, "required": ["agent_id", "query"]}}', '550e8400-c95b-4444-6666-000000000000')
ON CONFLICT (name) DO NOTHING;

CREATE TABLE IF NOT EXISTS tool_runs (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    tool_id UUID NOT NULL,
    connection_id UUID NOT NULL,
    thread_id UUID NOT NULL,
    agent_id UUID NOT NULL,
    recipient_id UUID NOT NULL,
    input JSONB,
    result JSONB,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'RUNNING', 'SUCCESS', 'FAILED')),
    duration FLOAT,
    parent_run_id TEXT, -- Reference to the parent tool_run_id if this is a child tool
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Foreign Key contraint linking to the tools table
    CONSTRAINT fk_tool_runs_tool_id
        FOREIGN KEY (tool_id)
        REFERENCES tools (id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,

    -- Foreign Key constraint linking to the agents table
    CONSTRAINT fk_tool_runs_agent_id 
        FOREIGN KEY (agent_id) 
        REFERENCES agents (id) 
        ON UPDATE CASCADE
        ON DELETE CASCADE,
    
    -- Foreign Key constraint linking to the threads table
    CONSTRAINT fk_tool_runs_thread_id 
        FOREIGN KEY (thread_id) 
        REFERENCES threads (id) 
        ON UPDATE CASCADE
        ON DELETE CASCADE,

    -- Foreign Key constraint linking to parent tool
    CONSTRAINT fk_tool_runs_parent_tool 
        FOREIGN KEY (parent_run_id)
        REFERENCES tool_runs (id) 
        ON UPDATE CASCADE
        ON DELETE CASCADE
);

--Add indexes for performance
CREATE INDEX IF NOT EXISTS idx_tool_runs_thread_id ON tool_runs (thread_id);
CREATE INDEX IF NOT EXISTS idx_tool_runs_agent_id ON tool_runs (agent_id);
CREATE INDEX IF NOT EXISTS idx_tool_runs_recipient_id ON tool_runs (recipient_id);
CREATE INDEX IF NOT EXISTS idx_tool_runs_parent_run_id ON tool_runs (parent_run_id);

-- Trigger to update the updated_at column on row update
DROP TRIGGER IF EXISTS set_timestamp_tool_runs ON tool_runs;
CREATE TRIGGER set_timestamp_tool_runs BEFORE
UPDATE ON tool_runs FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp ();

-- =============================================
-- TASKS TABLES
-- =============================================

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    thread_id UUID NOT NULL,
    max_request_loop INTEGER NOT NULL DEFAULT 20,
    additional_info JSONB, -- Metadata for the tasks
    parent_task_id TEXT, -- Reference to the parent task_id if this is a task handoffs to another agent
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID NOT NULL DEFAULT '550e8400-c95b-4444-6666-000000000000', -- Default to system
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Foreign Key contraint linking to the threads table
    CONSTRAINT fk_tasks_thread_id
        FOREIGN KEY (thread_id)
        REFERENCES threads(id)
        ON DELETE CASCADE,

    -- Foreign Key contraint linking to the users table
    CONSTRAINT fk_tasks_created_by
        FOREIGN KEY (created_by)
        REFERENCES users(id)
        ON DELETE SET NULL,

    -- Foreign Key constraint linking to the tasks table
    CONSTRAINT fk_tasks_parent_task_id
        FOREIGN KEY (parent_task_id)
        REFERENCES tasks(id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tasks_thread_id ON tasks (thread_id);
CREATE INDEX IF NOT EXISTS idx_tasks_created_by ON tasks (created_by);
CREATE INDEX IF NOT EXISTS idx_tasks_parent_task_id ON tasks (parent_task_id);

-- Trigger to update the updated_at column on row update
DROP TRIGGER IF EXISTS set_timestamp_tasks ON tasks;
CREATE TRIGGER set_timestamp_tasks BEFORE
UPDATE ON tasks FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp ();

CREATE TABLE IF NOT EXISTS tasks_runs (
    task_run_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'SCHEDULED' CHECK (status IN ('SCHEDULED', 'PENDING', 'RUNNING', 'FINISHED', 'FAILED')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    current_loops INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,

    -- Foreign Key contraint linking to the tasks table
    CONSTRAINT fk_tasks_runs_task_id
        FOREIGN KEY (task_id)
        REFERENCES tasks (id)
        ON UPDATE CASCADE
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tasks_runs_task_id ON tasks_runs (task_id);

-- Trigger to update the updated_at column on row update
DROP TRIGGER IF EXISTS set_timestamp_tasks_runs ON tasks_runs;
CREATE TRIGGER set_timestamp_tasks_runs BEFORE
UPDATE ON tasks_runs FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp ();

-- =============================================
-- WEBSOCKET TABLES
-- =============================================

-- Table to store websocket sessions
CREATE TABLE IF NOT EXISTS sessions (
    token TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Table to store user connections
CREATE TABLE IF NOT EXISTS user_connections (
    connection_id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexing the user_id column for faster lookups
CREATE INDEX IF NOT EXISTS idx_user_connections_user_id ON user_connections (user_id);

-- +goose Down
DROP EXTENSION IF EXISTS "pgcrypto";

DROP FUNCTION IF EXISTS trigger_set_timestamp () CASCADE;
DROP TRIGGER IF EXISTS set_timestamp_permissions ON permissions;
DROP TRIGGER IF EXISTS set_timestamp_roles ON roles;
DROP TRIGGER IF EXISTS set_timestamp_users ON users;
DROP TRIGGER IF EXISTS trigger_assign_pending_role_to_new_user ON users;
DROP FUNCTION IF EXISTS assign_pending_role_to_new_user () CASCADE;
DROP TRIGGER IF EXISTS set_timestamp_agents ON agents;
DROP TRIGGER IF EXISTS set_timestamp_threads ON threads;
DROP TRIGGER IF EXISTS set_timestamp_thread_context ON thread_context;
DROP TRIGGER IF EXISTS set_timestamp_thread_messages ON thread_messages;
DROP TRIGGER IF EXISTS set_timestamp_tools ON tools;
DROP TRIGGER IF EXISTS set_timestamp_tool_runs ON tool_runs;
DROP TRIGGER IF EXISTS set_timestamp_tasks ON tasks;
DROP TRIGGER IF EXISTS set_timestamp_tasks_runs ON tasks_runs;

DROP INDEX IF EXISTS idx_permissions_name;
DROP INDEX IF EXISTS idx_roles_id;
DROP INDEX IF EXISTS idx_roles_name;
DROP INDEX IF EXISTS idx_roles_system;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_name;
DROP INDEX IF EXISTS idx_role_permission_mapping_role_id;
DROP INDEX IF EXISTS idx_role_permission_mapping_permission_id;
DROP INDEX IF EXISTS idx_role_permission_mapping_assigned_by;
DROP INDEX IF EXISTS idx_user_role_mapping_user_id;
DROP INDEX IF EXISTS idx_user_role_mapping_role_id;
DROP INDEX IF EXISTS idx_user_role_mapping_assigned_by;
DROP INDEX IF EXISTS idx_agent_permission_mapping_agent_id;
DROP INDEX IF EXISTS idx_agent_permission_mapping_permission_id;
DROP INDEX IF EXISTS idx_thread_user_id;
DROP INDEX IF EXISTS idx_thread_updated_at;
DROP INDEX IF EXISTS idx_thread_message_thread_id;
DROP INDEX IF EXISTS idx_thread_message_created_at;
DROP INDEX IF EXISTS idx_thread_messages_sender_id;
DROP INDEX IF EXISTS idx_thread_messages_recipient_id;
DROP INDEX IF EXISTS idx_thread_messages_sender_type;
DROP INDEX IF EXISTS idx_tools_name;
DROP INDEX IF EXISTS idx_tools_created_by;
DROP INDEX IF EXISTS idx_tool_runs_request_id;
DROP INDEX IF EXISTS idx_tool_runs_thread_id;
DROP INDEX IF EXISTS idx_tool_runs_agent_id;
DROP INDEX IF EXISTS idx_tool_runs_recipient_id;
DROP INDEX IF EXISTS idx_tool_runs_parent_run_id;
DROP INDEX IF EXISTS idx_tasks_created_by;
DROP INDEX IF EXISTS idx_tasks_thread_id;
DROP INDEX IF EXISTS idx_tasks_runs_task_id;
DROP INDEX IF EXISTS idx_user_connections_user_id;


DROP TABLE IF EXISTS roles CASCADE;
DROP TABLE IF EXISTS permissions CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS role_permission_mapping CASCADE;
DROP TABLE IF EXISTS user_role_mapping CASCADE;
DROP TABLE IF EXISTS agents CASCADE;
DROP TABLE IF EXISTS agent_permission_mapping CASCADE;
DROP TABLE IF EXISTS threads CASCADE;
DROP TABLE IF EXISTS thread_context CASCADE;
DROP TABLE IF EXISTS thread_messages CASCADE;
DROP TABLE IF EXISTS tools CASCADE;
DROP TABLE IF EXISTS tool_runs CASCADE;
DROP TABLE IF EXISTS tasks CASCADE;
DROP TABLE IF EXISTS tasks_runs CASCADE;
DROP TABLE IF EXISTS user_connections CASCADE;
DROP TABLE IF EXISTS sessions CASCADE;