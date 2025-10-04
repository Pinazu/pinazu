package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/go-hclog"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/service"
)

// ToolProcessResult holds the categorized tools from recursive processing
type ToolProcessResult struct {
	StandaloneTools []service.StandaloneToolRequestEventMessage
	WorkflowTools   []service.FlowRunExecuteRequestEventMessage
	MCPTools        []service.StandaloneToolRequestEventMessage
}

type ToolService struct {
	s   service.Service
	log hclog.Logger
	wg  *sync.WaitGroup
	ctx context.Context
}

// Create a new tool handlers service instance
func NewService(ctx context.Context, externalDependenciesConfig *service.ExternalDependenciesConfig, log hclog.Logger, wg *sync.WaitGroup) (*ToolService, error) {
	if externalDependenciesConfig == nil {
		return nil, fmt.Errorf("externalDependenciesConfig is nil")
	}

	// Create a new service instance
	config := &service.Config{
		Name:                 "tools-handler-service",
		Version:              "0.0.1",
		Description:          "Tool service for handling tool execution, tool context management, and tool completion.",
		ExternalDependencies: externalDependenciesConfig,
		ErrorHandler:         nil,
	}
	s, err := service.NewService(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool service: %w", err)
	}

	ts := &ToolService{s: s, log: log, wg: wg, ctx: ctx}

	s.RegisterHandler(service.ToolDispatchEventSubject.String(), ts.dispatchEventCallback)
	s.RegisterHandler(service.ToolGatherEventSubject.String(), ts.gatherEventCallback)

	// Start a goroutine to wait for context cancellation and then shutdown
	go func() {
		<-ctx.Done()
		ts.log.Warn("Tool service shutting down...")
		if err := ts.s.Shutdown(); err != nil {
			ts.log.Error("Error during tool service shutdown", "error", err)
		}
		ts.wg.Done()
	}()

	return ts, nil
}
