package api

import (
	"context"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// Mock standalone server
// (POST /v1/mock/tool)
func (s Server) MockStandaloneTool(ctx context.Context, request MockStandaloneToolRequestObject) (MockStandaloneToolResponseObject, error) {
	if request.Body.Input == "" {
		return MockStandaloneTool400JSONResponse{Message: "input is required"}, nil
	}
	return MockStandaloneTool200JSONResponse{
		Text:     "Mock tool execute successfully",
		Citation: []anthropic.TextCitationParamUnion{},
	}, nil
}

// Mock standalone server with delay
// (POST /v1/mock/tool_with_delay)
func (s Server) MockStandaloneToolWithDelay(ctx context.Context, request MockStandaloneToolWithDelayRequestObject) (MockStandaloneToolWithDelayResponseObject, error) {
	if request.Body.Input == "" {
		return MockStandaloneToolWithDelay400JSONResponse{Message: "input is required"}, nil
	}
	time.Sleep(5 * time.Second)
	return MockStandaloneToolWithDelay200JSONResponse{
		Text:     "Mock tool execute successfully",
		Citation: []anthropic.TextCitationParamUnion{},
	}, nil
}
