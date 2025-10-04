package db

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func Test_ToolConfig(t *testing.T) {
	t.Parallel()

	var (
		exampleParameters = openapi3.Schema{
			Type: &openapi3.Types{"object"},
			Properties: map[string]*openapi3.SchemaRef{
				"tool_arg1": {
					Value: &openapi3.Schema{
						Type:        &openapi3.Types{"string"},
						Description: "Description of tool_arg1",
					},
				},
			},
			Required: []string{"tool_arg1"},
		}
		exampleParametersString = `{"properties":{"tool_arg1":{"description":"Description of tool_arg1","type":"string"}},"required":["tool_arg1"],"type":"object"}`
	)
	//     {"type":"standalone","api_key":"test_api_key","url":"https://example.com","param":{"properties":{"tool_arg1":{"type":"string","description":"Description of tool_arg1"}},"required":["tool_arg1"],"type":"object"}},
	// got {"type":"standalone","api_key":"test_api_key","url":"https://example.com","param":{"properties":{"tool_arg1":{"description":"Description of tool_arg1","type":"string"}},"required":["tool_arg1"],"type":"object"}}
	toolApiKey := "test_api_key"
	tests := []struct {
		name           string
		input          ToolType
		config         ToolConfigIntf
		expected       string
		expectedConfig string
	}{
		{"Standalone", ToolTypeStandalone, &ToolConfigStandalone{ApiKey: &toolApiKey, Url: "https://example.com", Params: &exampleParameters}, "standalone", `{"type":"standalone","api_key":"test_api_key","url":"https://example.com","params":` + exampleParametersString + `}`},
		{"Workflow", ToolTypeWorkflow, &ToolConfigWorkflow{S3Url: "s3://path/to/workflow", Params: &exampleParameters}, "workflow", `{"type":"workflow","s3_url":"s3://path/to/workflow","params":` + exampleParametersString + `}`},
		{"MCP", ToolTypeMCP, &ToolConfigMCP{ApiKey: &toolApiKey, Entrypoint: "https://mcp.example.com", Protocol: MCPProtocolSSE}, "mcp", `{"type":"mcp","entrypoint":"https://mcp.example.com","protocol":"sse","api_key":"test_api_key"}`},
		{"Nil", ToolTypeNil, nil, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapper := ToolConfig{
				Type: tt.input,
				C:    tt.config,
			}

			if wrapper.Type != tt.input {
				t.Errorf("expected type %v, got %v", tt.input, wrapper.Type)
			}

			if tt.config != nil {
				if wrapper.C == nil {
					t.Error("expected non-nil config")
				} else if wrapper.C.GetType() != tt.input {
					t.Errorf("expected config type %v, got %v", tt.input, wrapper.C.GetType())
				}
			} else {
				if wrapper.C != nil {
					t.Error("expected nil config")
				}
			}

			value, err := wrapper.Value()
			if err != nil {
				t.Fatalf("Value() error: %v", err)
			}

			expectedValue := tt.expectedConfig
			if string(value.([]byte)) != expectedValue {
				t.Errorf("expected value %s, got %s", expectedValue, string(value.([]byte)))
			}

			var scannedWrapper ToolConfig
			if err := scannedWrapper.Scan(value); err != nil {
				t.Fatalf("Scan() error: %v", err)
			}

			if scannedWrapper.Type != tt.input {
				t.Errorf("expected scanned type %v, got %v", tt.input, scannedWrapper.Type)
			}

			if tt.config != nil && scannedWrapper.C == nil {
				t.Error("expected non-nil scanned config")
			} else if tt.config == nil && scannedWrapper.C != nil {
				t.Error("expected nil scanned config")
			}
		})
	}
	// nil test for Scan method
	t.Run("NilScan", func(t *testing.T) {
		var wrapper ToolConfig
		if err := wrapper.Scan(nil); err != nil {
			t.Fatalf("Scan() error: %v", err)
		}
		if wrapper.Type != ToolTypeNil || wrapper.C != nil {
			t.Errorf("expected nil type and config, got type %v, config %v", wrapper.Type, wrapper.C)
		}
	})
}
