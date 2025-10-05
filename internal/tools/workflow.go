package tools

import (
	"time"

	"github.com/pinazu/internal/service"
)

// executeWorkflowTool publish tool execute event to workflow execution service
func (ts *ToolService) executeWorkflowTool(workflowToolsToExecute []service.FlowRunExecuteRequestEventMessage) {
	if len(workflowToolsToExecute) == 0 {
		return
	}

	ts.log.Info("Publishing workflow tools to execution", "count", len(workflowToolsToExecute))

	for _, workflow := range workflowToolsToExecute {
		// Create event for workflow tool execution
		event := service.NewEvent(
			&workflow,
			&service.EventHeaders{},
			&service.EventMetadata{
				Timestamp: time.Now().UTC(),
			},
		)

		err := event.Publish(ts.s.GetNATS())
		if err != nil {
			ts.log.Error("Failed to publish workflow tool event", "flow_id", workflow.FlowId, "error", err)
		} else {
			ts.log.Debug("Published workflow tool event", "flow_id", workflow.FlowId)
		}
	}
}
