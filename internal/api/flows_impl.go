package api

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/db"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/service"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/utils"
)

const FLOW_RESOURCE = "Flow"

func (s *Server) CreateFlow(ctx context.Context, req CreateFlowRequestObject) (CreateFlowResponseObject, error) {
	// Validate request
	addInfo, err := db.NewJsonRaw(req.Body.AdditionalInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal additional info: %w", err)
	}
	parametersSchema, err := db.NewJsonRaw(req.Body.ParametersSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters schema: %w", err)
	}
	if req.Body.Name == "" {
		return CreateFlow400JSONResponse(NotFound{
			Resource: FLOW_RESOURCE,
			Message:  "Flow name is required",
		}), nil
	}
	params := &db.CreateFlowParams{
		ID:   uuid.Must(uuid.NewV7()),
		Name: req.Body.Name,
		Description: pgtype.Text{
			String: "",
			Valid:  false,
		},
		ParametersSchema: parametersSchema,
		Engine:           req.Body.Engine,
		AdditionalInfo:   addInfo, // Assuming AdditionalInfo is optional and can be nil
		Tags:             []string{},
		CodeLocation:     pgtype.Text{String: req.Body.CodeLocation, Valid: true},
		Entrypoint:       pgtype.Text{String: req.Body.Entrypoint, Valid: true},
	}
	if req.Body.Tags != nil {
		params.Tags = *req.Body.Tags
	}
	if req.Body.Description != nil {
		params.Description = pgtype.Text{String: *req.Body.Description, Valid: true}
	}
	flow, err := s.queries.CreateFlow(ctx, *params)
	if err != nil {
		return nil, fmt.Errorf("failed to create flow: %w", err)
	}
	return CreateFlow201JSONResponse(flow), nil
}

func (s *Server) GetFlow(ctx context.Context, req GetFlowRequestObject) (GetFlowResponseObject, error) {
	flow, err := s.queries.GetFlowById(ctx, req.FlowId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return GetFlow404JSONResponse(NotFound{
				Resource: FLOW_RESOURCE,
				Id:       req.FlowId,
				Message:  fmt.Sprintf("Flow with ID %s not found", req.FlowId),
			}), nil
		}
		return nil, fmt.Errorf("failed to get flow: %w", err)
	}
	return GetFlow200JSONResponse(flow), nil
}

func (s *Server) UpdateFlow(ctx context.Context, req UpdateFlowRequestObject) (UpdateFlowResponseObject, error) {
	flow_id := req.FlowId
	// Validate request
	flow, err := s.queries.GetFlowById(ctx, flow_id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return UpdateFlow404JSONResponse(NotFound{
				Resource: FLOW_RESOURCE,
				Id:       flow_id,
				Message:  fmt.Sprintf("Flow with ID %s not found", flow_id),
			}), nil
		}
		return nil, fmt.Errorf("failed to get flow: %w", err)
	}
	params := &db.UpdateFlowParams{
		ID:               flow_id,
		Name:             flow.Name,
		Description:      flow.Description,
		ParametersSchema: flow.ParametersSchema,
		Engine:           flow.Engine,
		AdditionalInfo:   flow.AdditionalInfo, // Assuming AdditionalInfo is optional and can be nil
		Tags:             flow.Tags,
		CodeLocation:     flow.CodeLocation,
		Entrypoint:       flow.Entrypoint,
	}
	if req.Body.Name != nil {
		params.Name = *req.Body.Name
	}
	if req.Body.Description != nil {
		params.Description = pgtype.Text{String: *req.Body.Description, Valid: true}
	}
	if req.Body.Engine != nil {
		params.Engine = *req.Body.Engine
	}
	if req.Body.AdditionalInfo != nil {
		addInfo, err := db.NewJsonRaw(req.Body.AdditionalInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal additional info: %w", err)
		}
		params.AdditionalInfo = addInfo
	}
	if req.Body.Tags != nil {
		params.Tags = *req.Body.Tags
	}
	if req.Body.CodeLocation != nil {
		params.CodeLocation = pgtype.Text{String: *req.Body.CodeLocation, Valid: true}
	}
	if req.Body.Entrypoint != nil {
		params.Entrypoint = pgtype.Text{String: *req.Body.Entrypoint, Valid: true}
	}
	updatedFlow, err := s.queries.UpdateFlow(ctx, *params)
	if err != nil {
		return nil, fmt.Errorf("failed to update flow: %w", err)
	}
	return UpdateFlow200JSONResponse(updatedFlow), nil
}

func (s *Server) DeleteFlow(ctx context.Context, req DeleteFlowRequestObject) (DeleteFlowResponseObject, error) {
	flow_id := req.FlowId
	// Validate request
	flow, err := s.queries.GetFlowById(ctx, flow_id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return DeleteFlow404JSONResponse(NotFound{
				Resource: FLOW_RESOURCE,
				Id:       flow_id,
				Message:  fmt.Sprintf("Flow with ID %s not found", flow_id),
			}), nil
		}
		return nil, fmt.Errorf("failed to get flow: %w", err)
	}
	err = s.queries.DeleteFlow(ctx, flow.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete flow: %w", err)
	}
	return DeleteFlow204Response{}, nil
}

func (s *Server) ListFlows(ctx context.Context, req ListFlowsRequestObject) (ListFlowsResponseObject, error) {
	params := db.GetFlowsParams{
		Limit:  10,
		Offset: 0,
	}
	var page int32 = 1
	if req.Params.PerPage != nil {
		params.Limit = *req.Params.PerPage
	}
	if req.Params.Page != nil {
		page = *req.Params.Page
	}
	params.Offset = (page - 1) * params.Limit
	flows, err := s.queries.GetFlows(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list flows: %w", err)
	}

	return ListFlows200JSONResponse(FlowList{
		Flows:      flows,
		Page:       page,
		PerPage:    params.Limit,
		Total:      len(flows),
		TotalPages: (len(flows) + int(params.Limit) - 1) / int(params.Limit),
	}), nil
}

func (s *Server) ExecuteFlow(ctx context.Context, req ExecuteFlowRequestObject) (ExecuteFlowResponseObject, error) {
	flow, err := s.queries.GetFlowById(ctx, req.FlowId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ExecuteFlow404JSONResponse(NotFound{
				Resource: FLOW_RESOURCE,
				Id:       req.FlowId,
				Message:  fmt.Sprintf("Flow with ID %s not found", req.FlowId),
			}), nil
		}
		return nil, fmt.Errorf("failed to get flow: %w", err)
	}
	event := service.Event[*service.FlowRunExecuteRequestEventMessage]{
		H: &service.EventHeaders{
			UserID:       uuid.New(), // TODO: Get from authentication context
			ThreadID:     nil,        // Set to nil - will be omitted from JSON
			ConnectionID: nil,        // Set to nil - will be omitted from JSON
		},
		Msg: &service.FlowRunExecuteRequestEventMessage{
			FlowId:     req.FlowId,
			Parameters: req.Body.Parameters,
			Engine:     flow.Engine,
		},
		M: &service.EventMetadata{
			TraceID:   utils.GenerateTraceID(),
			Timestamp: time.Now().UTC(),
		},
	}
	s.log.Info("Publishing to NATS", "subject", event.Msg.Subject())
	resp, err := service.Request[*service.FlowRunExecuteResponseEventMessage](
		s.nc,
		&event,
		time.Second*5,
	)
	if err != nil {
		s.log.Error("Failed to request flow execution", "error", err)
		return nil, fmt.Errorf("failed to request flow execution: %w", err)
	}

	// Here you would implement the logic to execute the flow.
	// This is a placeholder response.
	return ExecuteFlow200JSONResponse(resp.Msg.FlowRun), nil
}

func (s *Server) GetFlowRun(ctx context.Context, req GetFlowRunRequestObject) (GetFlowRunResponseObject, error) {
	flowRun, err := s.queries.GetFlowRun(ctx, req.FlowRunId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return GetFlowRun404JSONResponse(NotFound{
				Resource: "FlowRun",
				Id:       req.FlowRunId,
				Message:  fmt.Sprintf("FlowRun with ID %s not found", req.FlowRunId),
			}), nil
		}
		return nil, fmt.Errorf("failed to get flow run: %w", err)
	}
	return GetFlowRun200JSONResponse(flowRun), nil
}
