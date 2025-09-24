package db

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	pq_compat "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
)

// Common test setup for database queries
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	db_pool, err := pgxpool.New(context.Background(), os.Getenv("POSTGRES_URL"))
	if err != nil {
		t.Fatalf("Failed to connect to the database: %v", err)
	}
	return db_pool
}

func TestMain(m *testing.M) {
	// Set up any global test configurations or connections here
	// For example, you might want to set up a test database connection
	// or initialize a NATS connection if needed.
	// Load environment variables if needed
	if err := godotenv.Load("../../.env"); err != nil {
		fmt.Printf("Error loading .env file: %s\n", err)
		fmt.Println("Using environment variables from the system")
	}
	fmt.Println("Starting tests...")
	// Run Migration if needed
	pool, err := pgxpool.New(context.Background(), os.Getenv("POSTGRES_URL"))
	if err != nil {
		panic(fmt.Errorf("failed to connect to the database: %w", err))
	}
	defer pool.Close()
	goose.SetBaseFS(os.DirFS("../../sql"))
	if err := goose.SetDialect("postgres"); err != nil {
		panic(fmt.Errorf("failed to set goose dialect: %w", err))
	}
	if err := goose.Up(pq_compat.OpenDBFromPool(pool), "migrations"); err != nil {
		panic(fmt.Errorf("failed to run migrations: %w", err))
	}
	code := m.Run() // Run the tests
	os.Exit(code)   // Exit with the code returned by m.Run()
}

func TestQueries(t *testing.T) {
	t.Parallel()
	db_pool := setupTestDB(t)
	defer db_pool.Close()
	queries := New(db_pool)
	roles, err := queries.GetAllRoles(context.Background())
	if err != nil {
		t.Fatalf("Failed to get roles: %v", err)
	}
	assert.NotEmpty(t, roles, "Roles should not be empty")
	// At least one role should exist in the test database
	assert.Greater(t, len(roles), 0, "There should be at least one role in the database")
	assert.NotNil(t, queries, "Queries should not be nil")
	assert.Equal(t, 1, 1, "This is a simple test to ensure the test framework is working")
	// Add more tests for individual query methods as needed
}

func TestNatsSendMessage(t *testing.T) {
	t.Parallel()
	nc, err := nats.Connect(os.Getenv("NATS_URL"))
	if err != nil {
		t.Fatalf("Failed to connect to NATS server: %v", err)
	}
	defer nc.Close()
	stream_ctx, err := nc.JetStream(nats.PublishAsyncMaxPending(1000))
	if err != nil {
		t.Fatalf("Failed to create JetStream context: %v", err)
	}
	_, err = stream_ctx.StreamInfo("test_stream")
	if err != nil {
		// Stream does not exist, create it
		_, err = stream_ctx.AddStream(&nats.StreamConfig{
			Name:      "test_stream",
			Subjects:  []string{"test.subject"},
			Storage:   nats.FileStorage,
			Retention: nats.LimitsPolicy,
			MaxAge:    24 * time.Hour,
			MaxMsgs:   1000,
			MaxBytes:  10 * 1024 * 1024, // 10 MB
			Discard:   nats.DiscardOld,
			Replicas:  1,
			NoAck:     false,
		})
		if err != nil {
			t.Fatalf("Failed to create NATS stream: %v", err)
		}
	}
	// Publish a test message
	pub_ack, err := stream_ctx.Publish("test.subject", []byte("test message"))
	if err != nil {
		t.Fatalf("Failed to publish message to NATS: %v", err)
	}
	assert.NotNil(t, pub_ack, "Publish acknowledgment should not be nil")
	assert.Equal(t, "test_stream", pub_ack.Stream, "Publish acknowledgment should indicate success")
	// Subscribe to stream to verify message receipt
	consumer, err := stream_ctx.AddConsumer("test_stream", &nats.ConsumerConfig{
		Durable:       "test_consumer",
		AckPolicy:     nats.AckExplicitPolicy,
		FilterSubject: "test.subject",
		MaxDeliver:    5,
		DeliverPolicy: nats.DeliverAllPolicy,
	})
	if err != nil {
		t.Fatalf("Failed to add consumer: %v", err)
	}
	assert.NotNil(t, consumer, "Consumer should not be nil")

	sub, err := stream_ctx.PullSubscribe("test.subject", "test_consumer")
	if err != nil {
		t.Fatalf("Failed to subscribe to NATS subject: %v", err)
	}
	msgs, err := sub.Fetch(1, nats.MaxWait(1*time.Second))
	if err != nil {
		t.Fatalf("Failed to fetch messages from NATS: %v", err)
	}
	// Receive at least 1 message
	assert.Greater(t, len(msgs), 0, "Should receive at least one message from NATS")
	for _, msg := range msgs {
		assert.Equal(t, "test message", string(msg.Data), "The message sent should match the received message")
		// Acknowledge the message
		if err := msg.Ack(); err != nil {
			t.Fatalf("Failed to acknowledge message: %v", err)
		}
	}
}
