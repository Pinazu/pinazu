# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.5] - 2025-08-14

### Added
- **OpenAPI Code Generation**: Automated API generation using OpenAPI/Swagger specifications with `oapi-codegen`
- **Comprehensive Permissions System**: Granular access control for all API endpoints with model-specific permissions
- **Enhanced Tool Management**: Tool execution tracking with parent-child relationships and duration monitoring
- **Worker Health Monitoring**: Added worker heartbeats table for service health tracking
- **Multi-Provider Agent Testing**: Extended testing support for external AI providers (AWS Bedrock, Google Gemini)
- **Tools Management Table**: Dedicated database table for comprehensive tool configuration and tracking

### Changed
- **API Development Workflow**: Transitioned to OpenAPI-first development with automated code generation
- **Frontend Architecture**: Updated SvelteKit web interface with enhanced basic frontend capabilities
- **Database Schema Refactoring**: 
  - Renamed `tool_results` table to `tool_runs` for consistency
  - Renamed `task_runs` to `flow_task_runs` for workflow clarity
  - Updated tools schema with `tool_type` field replacing `is_mcp` boolean
- **API Schema Updates**: Enhanced OpenAPI specifications with proper Go type mappings
- **CLAUDE.md Documentation**: Comprehensive updates with API development workflow and build instructions
- **Agent Provider Support**: Enhanced testing and integration for AWS Bedrock and Google Gemini agents

### Fixed
- **Database Migration**: Proper CASCADE handling for function drops and enhanced indexing

## [0.0.4] - 2025-08-01

### Added
- **PendingPanel Component**: New Svelte component for handling pending states in the web frontend
- **Comprehensive REST API Handlers**: Implemented complete REST API handlers for all core entities (agents, flows, tasks, tools, messages, users)

### Changed
- **Frontend Type Definitions**: Updated `web/src/lib/types.ts` with enhanced type definitions for Svelte migration
- **Main Page Component**: Modified `web/src/routes/+page.svelte` for improved Svelte framework compatibility
- ***Database Mapping**: Updated mapping type between Postgresql JSONB type and Golang type
  - The old version has db.JsonRaw type map[string]any in golang, which has limit
  - The new version has db.JsonRaw type json.RawMessage, which supports more generally any valid JSON value
  like array, object, string, number, etc.

## [0.0.2] - 2025-07-31

### Added
- **SvelteKit Web Frontend**: Comprehensive migration to SvelteKit frontend framework
  - Established modular frontend architecture in `web/src/`
  - Implemented new routing system for chat (`/chat/`) and login pages
  - Created type-safe API integration layer in `web/src/lib/api/`
  - Added reusable Svelte components in `web/src/lib/components/`
  - Introduced utility functions and type definitions to enhance developer experience
    - Created `types.ts` for centralized type management
    - Developed utility functions in `utils.ts` for common operations

### Changed
- **Anthropic Agent Handling**: Updated dynamic payload handling for Anthropic agents
  - Enhanced support for dynamic agent configuration
- **Frontend Architecture**: Transitioned from previous frontend template to modern SvelteKit framework

## [0.0.1] - 2025-07-29

### Added
- **Generic Message Types**: Implemented generic message struct types for different AI providers to enable consistent message handling across Anthropic, Bedrock, Gemini, and OpenAI agents
- **YAML Configuration Loading**: Agent service now dynamically loads agent specifications from YAML configuration files stored in the database via `queries.GetAgentSpecsByID()`
- **Enhanced Configuration Structure**: Added support for detailed YAML configuration including:
  - Model specifications (provider, model_id, max_tokens)
  - Optional parameters (temperature, top_p, top_k)
  - Thinking configuration (enabled, budget_token)
  - System prompts and custom agent behaviors
- **Multi-Provider YAML Support**: Enhanced provider support with proper YAML-based routing for:
  - bedrock/anthropic
  - bedrock
  - openai
  - google

### Changed
- **Agent Handler Signatures**: Updated all agent handlers (Anthropic, AWS Nova, Gemini, OpenAI) to accept AgentSpecs parameter
- **Configuration Loading Mechanism**: Agent service now unmarshals YAML configurations into `AgentSpecs` struct in `internal/agents/service.go`
  - When optional parameters not provided in YAML, API uses default values
  - Dynamic configuration loading replaces hardcoded agent specifications

### Fixed
- **YAML Struct Tag Issue**: Resolved panic in `gopkg.in/yaml.v3` by removing unsupported `omitzero` flags from struct tags
  - Fixed YAML unmarshaling errors that were preventing proper configuration loading
  - Ensures stable YAML parsing for agent specifications

### Fixed
- **YAML Struct Tag Issue**: Resolved panic in `gopkg.in/yaml.v3` by removing unsupported `omitzero` flags from struct tags
