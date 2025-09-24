package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	ws "github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/pinazu/core/internal/api/websocket"
	"github.com/pinazu/core/internal/db"
	"github.com/pinazu/core/internal/service"
	"github.com/pinazu/core/internal/utils"
)

type ApiGatewayService struct {
	s   service.Service
	log hclog.Logger
	wg  *sync.WaitGroup
	ctx context.Context
}

// NewService creates a new ApiGatewayService instance
func NewService(ctx context.Context, externalDependenciesConfig *service.ExternalDependenciesConfig, log hclog.Logger, wg *sync.WaitGroup) (*ApiGatewayService, error) {
	if externalDependenciesConfig == nil {
		return nil, fmt.Errorf("externalDependenciesConfig is nil")
	}

	// Create a new service instance
	config := &service.Config{
		Name:                 "api-gateway-service",
		Version:              "0.0.1",
		Description:          "API Gateway service for handling HTTP requests and routing.",
		ExternalDependencies: externalDependenciesConfig,
		ErrorHandler:         nil,
	}
	s, err := service.NewService(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API gateway service: %w", err)
	}

	// Create WebSocket connections map and handler
	wsConns := utils.NewSyncMap[uuid.UUID, *ws.Conn]()
	wsHandler := websocket.NewHandler(ctx, s.GetDB(), s.GetNATS(), wsConns, log)

	// Create a API Gateway Service
	ags := &ApiGatewayService{s: s, log: log, wg: wg, ctx: ctx}

	s.RegisterHandler("v1.svc.api._info", nil)
	s.RegisterHandler("v1.svc.api._stats", nil)
	// Migrate Database
	if err := db.MigrateDb(s.GetDB()); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}
	// Create HTTP server instance fo API Gateway
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%s", config.ExternalDependencies.Http.Port),
		Handler:      LoadRoutes(s.GetDB(), s.GetNATS(), wsHandler, log),
		ReadTimeout:  120 * time.Second, // Increased for long streaming responses
		WriteTimeout: 120 * time.Second, // Increased for long streaming responses
	}

	// Start a goroutine to wait for context cancellation and then shutdown
	go func() {
		<-ctx.Done()
		if err := ags.s.Shutdown(); err != nil {
			ags.log.Error("Error during API Gateway service shutdown", "error", err)
		}
		ags.wg.Done()
		ags.log.Warn("API Gateway service shutdown complete")
	}()

	go func() {
		ags.log.Info("Frontend available at", "url", fmt.Sprintf("http://0.0.0.0:%s", config.ExternalDependencies.Http.Port))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ags.log.Error("HTTP server error", "error", err)
		}
	}()

	return ags, nil
}
