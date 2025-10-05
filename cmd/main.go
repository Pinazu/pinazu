package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/pinazu/cli"
)

func main() {
	// Load environment variables from .env file
	_ = godotenv.Load()

	log := log.Default()
	cmd := cli.CreateCLICommand()

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatalf("Application error occurred: %v", err)
	}
}
