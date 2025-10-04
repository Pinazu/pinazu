package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/urfave/cli/v3"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/agents"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/api"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/flows"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/service"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/tasks"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/tools"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/worker"
)

func CreateCLICommand() *cli.Command {
	return &cli.Command{
		Name:  "pinazu",
		Usage: "GenAI Core Program CLI",
		Commands: []*cli.Command{
			{
				Name:    "serve",
				Aliases: []string{"s"},
				Usage:   "Start and serve the core server",
				Commands: []*cli.Command{
					{
						Name:   "all",
						Usage:  "Run all services",
						Flags:  createServeFlags(),
						Action: createServeAction(),
					},
					{
						Name:   "agent",
						Usage:  "Run the agent service",
						Flags:  createServeFlags(),
						Action: createServeAgentAction(),
					},
					{
						Name:   "api",
						Usage:  "Run the API gateway service",
						Flags:  createServeFlags(),
						Action: createServeAPIGatewayAction(),
					},
					{
						Name:   "flows",
						Usage:  "Run the workflow service",
						Flags:  createServeFlags(),
						Action: createServeFlowsAction(),
					},
					{
						Name:   "tasks",
						Usage:  "Run the task service",
						Flags:  createServeFlags(),
						Action: createServeTasksAction(),
					},
					{
						Name:   "tools",
						Usage:  "Run the tools service",
						Flags:  createServeFlags(),
						Action: createServeToolsAction(),
					},
					{
						Name:   "worker",
						Usage:  "Run the worker service",
						Flags:  createServeFlags(),
						Action: createServeWorkerAction(),
					},
				},
			},
			{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "Display the version of the application",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Println("0.0.1")
					return nil
				},
			},
		},
	}
}

// createServeFlags defines the flags used for the serve command and its subcommands.
func createServeFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Path to configuration file",
		},
		&cli.StringFlag{
			Name:    "port",
			Aliases: []string{"p"},
			Usage:   "Server port",
		},
		&cli.StringFlag{
			Name:  "nats-url",
			Usage: "NATS server URL",
		},
		&cli.StringFlag{
			Name:  "db-host",
			Usage: "Database host (PostgreSQL)",
		},
		&cli.StringFlag{
			Name:  "db-port",
			Usage: "Database port (PostgreSQL)",
		},
		&cli.StringFlag{
			Name:  "db-user",
			Usage: "Database user (PostgreSQL)",
		},
		&cli.StringFlag{
			Name:  "db-password",
			Usage: "Database password (PostgreSQL)",
		},
		&cli.StringFlag{
			Name:  "db-name",
			Usage: "Database name (PostgreSQL)",
		},
	}
}

func createServeAction() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		// Load the YAML configuration file if provided from `config` flag
		config, err := service.LoadExternalConfigFile(cmd.String("config"), cmd)
		if err != nil {
			return err
		}

		// Create logger with appropriate level based on debug configuration
		log := config.CreateLogger()
		log.Info("Starting Pinazu Core...")

		// Create a context that listens for OS signals for graceful shutdown
		signalCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()

		// Create the application instance
		wg := &sync.WaitGroup{}
		wg.Add(6)

		// Create services - they handle their own lifecycle via service.Service
		_, err = agents.NewService(signalCtx, config, log, wg)
		if err != nil {
			return fmt.Errorf("failed to create agents handler service: %w", err)
		}
		_, err = api.NewService(signalCtx, config, log, wg)
		if err != nil {
			return fmt.Errorf("failed to create API gateway service: %w", err)
		}
		_, err = flows.NewService(signalCtx, config, log, wg)
		if err != nil {
			return fmt.Errorf("failed to create flows orchestration service: %w", err)
		}
		_, err = tasks.NewService(signalCtx, config, log, wg)
		if err != nil {
			return fmt.Errorf("failed to create tasks handler service: %w", err)
		}
		_, err = tools.NewService(signalCtx, config, log, wg)
		if err != nil {
			return fmt.Errorf("failed to create tools handler service: %w", err)
		}
		_, err = worker.NewService(signalCtx, config, log, wg)
		if err != nil {
			return fmt.Errorf("failed to create worker service: %w", err)
		}
		log.Info("All services started successfully, waiting for shutdown signal...")

		// Wait for shutdown signal
		<-signalCtx.Done()
		log.Warn("Shutdown signal received, waiting for services to complete...")

		wg.Wait()
		log.Info("All services shut down.")
		return nil
	}
}

// createServeAgentAction is a placeholder for the agent service action.
func createServeAgentAction() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		// Load the YAML configuration file if provided from `config` flag
		config, err := service.LoadExternalConfigFile(cmd.String("config"), cmd)
		if err != nil {
			return err
		}

		// Create logger with appropriate level based on debug configuration
		log := config.CreateLogger()
		log.Info("Starting agent handler service...")

		// Create a context that listens for OS signals for graceful shutdown
		signalCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()

		// Create the application instance
		wg := &sync.WaitGroup{}
		wg.Add(1)
		_, err = agents.NewService(signalCtx, config, log, wg)
		if err != nil {
			return fmt.Errorf("failed to create agent service: %w", err)
		}

		log.Info("Agent service started successfully, waiting for shutdown signal...")

		// Wait for shutdown signal
		<-signalCtx.Done()
		log.Warn("Shutdown signal received, waiting for service to complete...")

		wg.Wait()
		log.Info("Agent handler service shut down complete.")
		return nil
	}
}

func createServeAPIGatewayAction() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		// Load the YAML configuration file if provided from `config` flag
		config, err := service.LoadExternalConfigFile(cmd.String("config"), cmd)
		if err != nil {
			return err
		}

		// Create logger with appropriate level based on debug configuration
		log := config.CreateLogger()
		log.Info("Starting API gateway service...")

		// Create a context that listens for OS signals for graceful shutdown
		signalCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()

		// Create the application instance
		wg := &sync.WaitGroup{}
		wg.Add(1)
		_, err = api.NewService(signalCtx, config, log, wg)
		if err != nil {
			return fmt.Errorf("failed to create API gateway service: %w", err)
		}

		log.Info("API gateway service started successfully, waiting for shutdown signal...")

		// Wait for shutdown signal
		<-signalCtx.Done()
		log.Warn("Shutdown signal received, waiting for service to complete...")

		wg.Wait()
		log.Info("API gateway service shut down complete.")
		return nil
	}
}

func createServeFlowsAction() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		// Load the YAML configuration file if provided from `config` flag
		config, err := service.LoadExternalConfigFile(cmd.String("config"), cmd)
		if err != nil {
			return err
		}

		// Create logger with appropriate level based on debug configuration
		log := config.CreateLogger()
		log.Info("Starting flows management service...")

		// Create a context that listens for OS signals for graceful shutdown
		signalCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()

		// Create the application instance
		wg := &sync.WaitGroup{}
		wg.Add(1)
		_, err = flows.NewService(signalCtx, config, log, wg)
		if err != nil {
			return fmt.Errorf("failed to create flow service: %w", err)
		}

		log.Info("Flow service started successfully, waiting for shutdown signal...")

		// Wait for shutdown signal
		<-signalCtx.Done()
		log.Warn("Shutdown signal received, waiting for service to complete...")

		wg.Wait()
		log.Info("Flow management service shut down complete.")
		return nil
	}
}

func createServeTasksAction() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		// Load the YAML configuration file if provided from `config` flag
		config, err := service.LoadExternalConfigFile(cmd.String("config"), cmd)
		if err != nil {
			return err
		}

		// Create logger with appropriate level based on debug configuration
		log := config.CreateLogger()
		log.Info("Starting tasks handler service...")

		// Create a context that listens for OS signals for graceful shutdown
		signalCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()

		// Create the application instance
		wg := &sync.WaitGroup{}
		wg.Add(1)
		_, err = tasks.NewService(signalCtx, config, log, wg)
		if err != nil {
			return fmt.Errorf("failed to create task service: %w", err)
		}

		log.Info("Task service started successfully, waiting for shutdown signal...")

		// Wait for shutdown signal
		<-signalCtx.Done()
		log.Warn("Shutdown signal received, waiting for service to complete...")

		wg.Wait()
		log.Info("Tasks handler service shut down complete.")
		return nil
	}
}

func createServeToolsAction() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		// Load the YAML configuration file if provided from `config` flag
		config, err := service.LoadExternalConfigFile(cmd.String("config"), cmd)
		if err != nil {
			return err
		}

		// Create logger with appropriate level based on debug configuration
		log := config.CreateLogger()
		log.Info("Starting tools handler service...")

		// Create a context that listens for OS signals for graceful shutdown
		signalCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()

		// Create the application instance
		wg := &sync.WaitGroup{}
		wg.Add(1)
		_, err = tools.NewService(signalCtx, config, log, wg)
		if err != nil {
			return fmt.Errorf("failed to create tool service: %w", err)
		}

		log.Info("Tool service started successfully, waiting for shutdown signal...")

		// Wait for shutdown signal
		<-signalCtx.Done()
		log.Warn("Shutdown signal received, waiting for service to complete...")

		wg.Wait()
		log.Info("Tools handler service shut down complete.")
		return nil
	}
}

func createServeWorkerAction() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		// Load the YAML configuration file if provided from `config` flag
		config, err := service.LoadExternalConfigFile(cmd.String("config"), cmd)
		if err != nil {
			return err
		}

		// Create logger with appropriate level based on debug configuration
		log := config.CreateLogger()
		log.Info("Starting worker service...")

		// Create a context that listens for OS signals for graceful shutdown
		signalCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()

		// Create the application instance
		wg := &sync.WaitGroup{}
		wg.Add(1)
		_, err = worker.NewService(signalCtx, config, log, wg)
		if err != nil {
			return fmt.Errorf("failed to create worker service: %w", err)
		}

		log.Info("Worker service started successfully, waiting for shutdown signal...")

		// Wait for shutdown signal
		<-signalCtx.Done()
		log.Warn("Shutdown signal received, waiting for service to complete...")

		wg.Wait()
		log.Info("Worker service shut down complete.")
		return nil
	}
}
