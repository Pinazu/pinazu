package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/getkin/kin-openapi/openapi3"
)

// JSONRaw is a type that represents a JSON object in Go.
type JsonRaw json.RawMessage

func NewJsonRaw(v any) (JsonRaw, error) {
	b, err := json.Marshal(v)
	return JsonRaw(b), err
}

func (j JsonRaw) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.RawMessage(j).MarshalJSON()
}

func (j *JsonRaw) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*j = nil
	case []byte:
		*j = JsonRaw(v)
	case string:
		*j = JsonRaw([]byte(v))
	default:
		return fmt.Errorf("cannot scan type %T into JsonRaw", v)
	}
	return nil
}

func (j JsonRaw) MarshalJSON() ([]byte, error) {
	return json.RawMessage(j).MarshalJSON()
}

func (j *JsonRaw) UnmarshalJSON(data []byte) error {
	*j = JsonRaw(append((*j)[0:0], data...)) // safe reallocation
	return nil
}

type ToolType string

const (
	ToolTypeStandalone ToolType = "standalone"
	ToolTypeWorkflow   ToolType = "workflow"
	ToolTypeMCP        ToolType = "mcp"
	ToolTypeInternal   ToolType = "internal"
	ToolTypeNil        ToolType = ""
)

type ProviderName string

const (
	ProviderNameLocal  ProviderName = "local"
	ProviderNameGoogle ProviderName = "google"
	ProviderNameAzure  ProviderName = "azure"
	ProviderNameGithub ProviderName = "github"
	ProviderNameNil    ProviderName = ""
)

type ProviderModel string

const (
	ProviderModelAnthropic        ProviderModel = "anthropic"
	ProviderModelBedrockAnthropic ProviderModel = "bedrock/anthropic"
	ProviderModelBedrock          ProviderModel = "bedrock"
	ProviderModelGoogle           ProviderModel = "google"
	ProviderModelOpenAI           ProviderModel = "openai"
	ProviderModelNil              ProviderModel = ""
)

type SenderMessageType string

const (
	SenderMessageTypeUser      SenderMessageType = "user"
	SenderMessageTypeAssistant SenderMessageType = "assistant"
	SenderMessageTypeSystem    SenderMessageType = "system"
	SenderMessageTypeResult    SenderMessageType = "result"
	SenderMessageTypeNil       SenderMessageType = ""
)

type ResultMessageType string

const (
	ResultMessageTypeText  ResultMessageType = "text"
	ResultMessageTypeError ResultMessageType = "error"
	ResultMessageTypeCode  ResultMessageType = "code"
	ResultMessageTypeImage ResultMessageType = "image"
	ResultMessageTypeNil   ResultMessageType = ""
)

type Status string

const (
	StatusPending Status = "PENDING"
	StatusRunning Status = "RUNNING"
	StatusSuccess Status = "SUCCESS"
	StatusFailed  Status = "FAILED"
	StatusNil     Status = ""
)

type FlowStatus string

const (
	FlowStatusScheduled FlowStatus = "SCHEDULED"
	FlowStatusPending   FlowStatus = "PENDING"
	FlowStatusRunning   FlowStatus = "RUNNING"
	FlowStatusSuccess   FlowStatus = "SUCCESS"
	FlowStatusFailed    FlowStatus = "FAILED"
	FlowStatusNil       FlowStatus = ""
)

type ToolRunStatus string

const (
	ToolRunStatusPending ToolRunStatus = "PENDING"
	ToolRunStatusRunning ToolRunStatus = "RUNNING"
	ToolRunStatusSuccess ToolRunStatus = "SUCCESS"
	ToolRunStatusFailed  ToolRunStatus = "FAILED"
	ToolRunStatusNil     ToolRunStatus = ""
)

type WorkerStatus string

const (
	WorkerStatusInactive WorkerStatus = "INACTIVE"
	WorkerStatusActive   WorkerStatus = "ACTIVE"
	WorkerStatusFailed   WorkerStatus = "FAILED"
	WorkerStatusNil      WorkerStatus = ""
)

type TaskRunStatus string

const (
	TaskRunStatusScheduled TaskRunStatus = "SCHEDULED"
	TaskRunStatusPending   TaskRunStatus = "PENDING"
	TaskRunStatusRunning   TaskRunStatus = "RUNNING"
	TaskRunStatusFinished  TaskRunStatus = "FINISHED"
	TaskRunStatusFailed    TaskRunStatus = "FAILED"
	TaskRunStatusNil       TaskRunStatus = ""
)

type (
	// EventType is a type alias for string to represent event types
	EventType string
)

// EventType contains all available service message event types
const (
	TaskStart          EventType = "task_start"
	TaskStop           EventType = "task_stop"
	TaskPause          EventType = "task_pause"
	TaskResume         EventType = "task_resume"
	MessageStart       EventType = "message_start"
	MessageDelta       EventType = "message_delta"
	MessageStop        EventType = "message_stop"
	ContentBlockStart  EventType = "content_block_start"
	ContentBlockDelta  EventType = "content_block_delta"
	ContentBlockStop   EventType = "content_block_stop"
	ToolStartEventType EventType = "tool_start"
	ToolDeltaEventType EventType = "tool_delta"
	ToolStopEventType  EventType = "tool_stop"
	NilEventType       EventType = ""
)

type MCPProtocol string

const (
	MCPProtocolStdio MCPProtocol = "stdio"
	MCPProtocolSSE   MCPProtocol = "sse"
	MCPProtocolGRPC  MCPProtocol = "grpc"
	MCPProtocolNil   MCPProtocol = ""
)

type ToolConfigIntf interface {
	GetType() ToolType
	Validate() error
}

type ToolConfigStandalone struct {
	ApiKey *string          `json:"api_key,omitempty"` // Optional API key for the tool, applicable for HTTP-based tools
	Url    string           `json:"url"`
	Params *openapi3.Schema `json:"params"` // Parameter schema for the tool
	// Note: The Param field is required and used to define the parameters for the tool.
	// It should be a valid OpenAPI schema object.
	// Example: {"type": "object", "properties": {"tool_arg1": {"type": "string", "description": "Description of tool_arg1"}}, "required": ["tool_arg1"]}
}

func (t *ToolConfigStandalone) GetType() ToolType {
	return ToolTypeStandalone
}

func (t *ToolConfigStandalone) Validate() error {
	if t.Url == "" {
		return fmt.Errorf("url is required for standalone tool")
	}
	if t.Params == nil {
		return fmt.Errorf("param schema is required for standalone tool")
	}
	// Validate the URL
	if _, err := url.ParseRequestURI(t.Url); err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}
	return nil
}

type ToolConfigWorkflow struct {
	S3Url  string           `json:"s3_url"`
	Params *openapi3.Schema `json:"params"` // Parameter schema for the tool
	// Note: The Param field is required and used to define the parameters for the tool.
	// It should be a valid OpenAPI schema object.
	// Example: {"type": "object", "properties": {"tool_arg1": {"type": "string", "description": "Description of tool_arg1"}}, "required": ["tool_arg1"]}
}

func (t *ToolConfigWorkflow) GetType() ToolType {
	return ToolTypeWorkflow
}

func (t *ToolConfigWorkflow) Validate() error {
	if t.S3Url == "" {
		return fmt.Errorf("s3_url is required for workflow tool")
	}
	// Validate the S3 URL
	url, err := url.ParseRequestURI(t.S3Url)
	if err != nil {
		return fmt.Errorf("invalid S3 URL format: %v", err)
	}
	if url.Scheme != "s3" {
		return fmt.Errorf("S3 URL must have 's3' scheme, got %s", url.Scheme)
	}
	return nil
}

type ToolConfigMCP struct {
	Entrypoint string             `json:"entrypoint"`         // In case of stdio, this is the path to the executable
	Protocol   MCPProtocol        `json:"protocol"`           // Optional field for protocol, one of "stdio", "sse", "grpc"
	EnvVars    *map[string]string `json:"env_vars,omitempty"` // Optional environment variables for the MCP tool, applicable for stdio
	ApiKey     *string            `json:"api_key,omitempty"`  // Optional API key for the MCP tool, applicable for HTTP-based tools, i.e, "sse" or "grpc"
}

func (t *ToolConfigMCP) GetType() ToolType {
	return ToolTypeMCP
}
func (t *ToolConfigMCP) Validate() error {
	if t.Entrypoint == "" {
		return fmt.Errorf("entrypoint is required for MCP tool")
	}
	if t.Protocol == MCPProtocolNil {
		return fmt.Errorf("protocol is required for MCP tool")
	}
	if t.Protocol != MCPProtocolStdio && t.Protocol != MCPProtocolSSE && t.Protocol != MCPProtocolGRPC {
		return fmt.Errorf("invalid protocol: %s, must be one of 'stdio', 'sse', 'grpc'", t.Protocol)
	}
	if t.Protocol == MCPProtocolStdio && t.EnvVars == nil {
		return fmt.Errorf("env_vars are required for stdio protocol")
	}
	return nil
}

type ToolConfigInternal struct {
	Params *openapi3.Schema `json:"params"` // Parameter schema for the tool
	// Note: The Param field is required and used to define the parameters for the tool.
	// It should be a valid OpenAPI schema object.
	// Example: {"type": "object", "properties": {"tool_arg1": {"type": "string", "description": "Description of tool_arg1"}}, "required": ["tool_arg1"]}
}

func (t *ToolConfigInternal) GetType() ToolType {
	return ToolTypeInternal
}
func (t *ToolConfigInternal) Validate() error {
	if t.Params == nil {
		return fmt.Errorf("params is required for internal tool")
	}
	return nil
}

type ToolConfig struct {
	Type ToolType `json:"type"`
	C    ToolConfigIntf
}

func (t *ToolConfig) Validate() error {
	if t.C == nil {
		return fmt.Errorf("tool config is required")
	}
	return t.C.Validate()
}

func (t *ToolConfig) GetStandalone() *ToolConfigStandalone {
	if t.Type != ToolTypeStandalone {
		return nil
	}
	// Handle both pointer and value types
	if standalonePtr, ok := t.C.(*ToolConfigStandalone); ok {
		return standalonePtr
	}
	return nil
}

func (t *ToolConfig) GetWorkflow() *ToolConfigWorkflow {
	if t.Type != ToolTypeWorkflow {
		return nil
	}
	// Handle both pointer and value types
	if workflowPtr, ok := t.C.(*ToolConfigWorkflow); ok {
		return workflowPtr
	}
	return nil
}

func (t *ToolConfig) GetMCP() *ToolConfigMCP {
	if t.Type != ToolTypeMCP {
		return nil
	}
	// Handle both pointer and value types
	if mcpPtr, ok := t.C.(*ToolConfigMCP); ok {
		return mcpPtr
	}
	return nil
}

func (t *ToolConfig) GetInternal() *ToolConfigInternal {
	if t.Type != ToolTypeInternal {
		return nil
	}
	// Handle both pointer and value types
	if internalPtr, ok := t.C.(*ToolConfigInternal); ok {
		return internalPtr
	}
	return nil
}

// Value and Scan methods for ToolConfigWrapper to implement driver.Valuer and sql.Scanner interfaces
func (t ToolConfig) Value() (driver.Value, error) {
	if t.C == nil {
		return []byte(""), nil
	}
	data, err := json.Marshal(t)
	if err != nil {
		return []byte(""), err
	}
	return data, nil
}

func (t *ToolConfig) Scan(src interface{}) error {
	if src == nil {
		t.Type = ToolTypeNil
		t.C = nil
		return nil
	}

	switch v := src.(type) {
	case []byte:
		if len(v) == 0 {
			t.Type = ToolTypeNil
			t.C = nil
			return nil
		}
		return json.Unmarshal(v, t)
	case string:
		if v == "" {
			t.Type = ToolTypeNil
			t.C = nil
			return nil
		}
		return json.Unmarshal([]byte(v), t)
	default:
		return fmt.Errorf("cannot scan type %T into ToolConfigWrapper", v)
	}
}

func (t ToolConfig) MarshalJSON() ([]byte, error) {
	if t.C == nil {
		return json.Marshal(map[string]interface{}{
			"type": t.Type,
		})
	}
	b1, err := json.Marshal(struct {
		Type ToolType `json:"type"`
	}{
		Type: t.Type,
	})
	if err != nil {
		return nil, err
	}
	b2, err := json.Marshal(t.C)
	if err != nil {
		return nil, err
	}
	s1 := string(b1[:len(b1)-1])
	s2 := string(b2[1:])
	return []byte(s1 + ", " + s2), nil
}

func (t *ToolConfig) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		t.Type = ToolTypeNil
		t.C = nil
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if typeData, ok := raw["type"]; ok {
		if err := json.Unmarshal(typeData, &t.Type); err != nil {
			return err
		}
	}

	switch t.Type {
	case ToolTypeStandalone:
		t.C = &ToolConfigStandalone{}
	case ToolTypeWorkflow:
		t.C = &ToolConfigWorkflow{}
	case ToolTypeMCP:
		t.C = &ToolConfigMCP{}
	case ToolTypeInternal:
		t.C = &ToolConfigInternal{}
	default:
		t.C = nil
	}
	return json.Unmarshal(data, &t.C)
}
