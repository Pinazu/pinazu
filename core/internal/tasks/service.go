package tasks

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/pinazu/core/internal/service"
)

type TaskService struct {
	s   service.Service
	log hclog.Logger
	wg  *sync.WaitGroup
	ctx context.Context
}

// NewService creates a new TaskService instance
func NewService(ctx context.Context, externalDependenciesConfig *service.ExternalDependenciesConfig, log hclog.Logger, wg *sync.WaitGroup) (*TaskService, error) {
	if externalDependenciesConfig == nil {
		return nil, fmt.Errorf("externalDependenciesConfig is nil")
	}

	// Create a new service instance
	config := &service.Config{
		Name:                 "tasks-handler-service",
		Version:              "0.0.1",
		Description:          "Task service for handling task execution, agent context managment, and task completion.",
		ExternalDependencies: externalDependenciesConfig,
		ErrorHandler:         nil,
	}
	s, err := service.NewService(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create task service: %w", err)
	}

	ts := &TaskService{s: s, log: log, wg: wg, ctx: ctx}

	s.RegisterHandler(service.TaskExecuteEventSubject.String(), ts.executeEventCallback)
	s.RegisterHandler(service.TaskHandoffEventSubject.String(), ts.handoffEventCallback)
	s.RegisterHandler(service.TaskFinishEventSubject.String(), ts.finishEventCallback)
	s.RegisterHandler(service.TaskCancelEventSubject.String(), ts.cancelEventCallback)

	// Start a goroutine to wait for context cancellation and then shutdown
	go func() {
		<-ctx.Done()
		ts.log.Warn("Task service shutting down...")
		if err := ts.s.Shutdown(); err != nil {
			ts.log.Error("Error during task service shutdown", "error", err)
		}
		ts.wg.Done()
	}()

	return ts, nil
}
