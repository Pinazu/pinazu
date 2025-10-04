package tools

import "gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/service"

// executeMCPTool publish tool execute event to MCP server
func (ts *ToolService) executeMCPTool(mcpToolsToExecute []service.StandaloneToolRequestEventMessage) {
	if len(mcpToolsToExecute) == 0 {
		return
	}
	// TODO: Have not support MCP server
	ts.log.Error("No support MCP tool server connection")
}
