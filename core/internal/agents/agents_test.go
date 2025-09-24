package agents

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/pinazu/core/internal/service"
)

func newMockServiceConfig() *service.ExternalDependenciesConfig {
	if err := godotenv.Load("../../.env"); err != nil {
		fmt.Printf("Error loading .env file: %s\n", err)
		fmt.Println("Using environment variables from the system")
	}
	return &service.ExternalDependenciesConfig{
		Debug: true,
		Http:  nil,
		Nats: &service.NatsConfig{
			URL:                    os.Getenv("NATS_URL"),
			JetStreamDefaultConfig: nil,
		},
		Database: &service.DatabaseConfig{
			Host:     os.Getenv("POSTGRES_HOST"),
			Port:     os.Getenv("POSTGRES_PORT"),
			User:     os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			Dbname:   os.Getenv("POSTGRES_DB"),
			SSLMode:  "disable",
		},
		Tracing: nil,
	}
}

var MockServiceConfigs = newMockServiceConfig()
